package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/git-maks/pl-app/apps/api/internal/game"
	"github.com/git-maks/pl-app/apps/api/internal/qr"
)

const ratingLock = 15 * time.Minute

type ScanRequest struct {
	Payload   string     `json:"payload"`
	ScannedAt *time.Time `json:"scannedAt"`
	Queued    bool       `json:"queued"`
}

// resolveStation validates a scanned payload and returns the station plus the trust level.
func (s *Service) resolveStation(ctx context.Context, ev Event, req ScanRequest, now time.Time) (Station, string, error) {
	p := qr.Parse(req.Payload)
	switch p.Kind {
	case qr.Manual:
		stations, err := s.Stations(ctx, ev.ID)
		if err != nil {
			return Station{}, "", err
		}
		w := qr.Window(now, qr.ManualPeriodSeconds)
		for _, st := range stations {
			for _, cand := range []int64{w, w - 1} {
				if qr.Equal(qr.ManualCode(st.secret, st.ID, cand), p.Code) {
					return st, "manual", nil
				}
			}
		}
		return Station{}, "", ErrQRInvalid
	case qr.Rotating, qr.Static:
		st, err := s.station(ctx, p.StationID)
		if err == ErrNotFound || (err == nil && st.eventID != ev.ID) {
			return Station{}, "", ErrQRInvalid
		}
		if err != nil {
			return Station{}, "", err
		}
		if p.Kind == qr.Static {
			if !qr.Equal(qr.StaticMAC(st.secret, st.ID), p.MAC) {
				return Station{}, "", ErrQRInvalid
			}
			return st, "static", nil
		}
		if !qr.Equal(qr.RotatingMAC(st.secret, st.ID, p.Window), p.MAC) {
			return Station{}, "", ErrQRInvalid
		}
		age := qr.Window(now, qr.PeriodSeconds) - p.Window
		maxAge := int64(1)
		if req.Queued {
			maxAge = int64(ev.queuedGrace / (qr.PeriodSeconds * time.Second))
		}
		if age > maxAge || age < -1 {
			return Station{}, "", ErrQRExpired
		}
		return st, "rotating", nil
	default:
		return Station{}, "", ErrQRInvalid
	}
}

// Scan claims a visit (spec §7.4): validates the code, the scan window and uniqueness.
func (s *Service) Scan(ctx context.Context, userID string, req ScanRequest) (ScanResult, error) {
	if len(req.Payload) > 512 {
		return ScanResult{}, ErrQRInvalid
	}
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	now := s.Now()
	st, trust, err := s.resolveStation(ctx, ev, req, now)
	if err != nil {
		return ScanResult{}, err
	}
	// A replayed offline scan counts at its capture time, but only within the grace period.
	at := now
	if req.Queued && req.ScannedAt != nil && req.ScannedAt.Before(now) && now.Sub(*req.ScannedAt) <= ev.queuedGrace {
		at = *req.ScannedAt
	}
	if open, _ := s.scanOpen(st, at); !open {
		return ScanResult{}, ErrWindowClosed
	}

	var teamID *string
	if st.Team != nil {
		var t string
		err := s.DB.QueryRow(ctx, `select team_id::text from team_members where user_id = $1 and event_id = $2`, userID, ev.ID).Scan(&t)
		if err == nil {
			teamID = &t
		} else if !isNoRows(err) {
			return ScanResult{}, wrap("team lookup", err)
		}
	}
	var scanID string
	err = s.DB.QueryRow(ctx, `insert into scans (event_id, user_id, station_id, scanned_at, queued, trust, team_id)
		values ($1, $2, $3, $4, $5, $6, $7) on conflict (user_id, station_id) do nothing returning id::text`,
		ev.ID, userID, st.ID, at, req.Queued, trust, teamID).Scan(&scanID)
	already := false
	if isNoRows(err) {
		already = true
		if err := s.DB.QueryRow(ctx, `select id::text from scans where user_id = $1 and station_id = $2`, userID, st.ID).Scan(&scanID); err != nil {
			return ScanResult{}, wrap("existing scan", err)
		}
	} else if err != nil {
		return ScanResult{}, wrap("insert scan", err)
	}
	if !already {
		if teamID != nil {
			s.checkTeamCompletion(ctx, ev, *teamID, st)
		}
		s.awardBadges(ctx, userID, ev)
	}
	sc, err := s.scanByID(ctx, userID, scanID)
	if err != nil {
		return ScanResult{}, err
	}
	return ScanResult{Scan: sc, AlreadyScanned: already}, nil
}

const scanSelect = `select sc.id::text, sc.station_id::text, st.name, sc.scanned_at, sc.rating, sc.team_id is not null, sc.team_complete,
	sc.deltas, st.team_min_members,
	(select count(*) from scans x where sc.team_id is not null and x.team_id = sc.team_id and x.station_id = sc.station_id)
	from scans sc join stations st on st.id = sc.station_id`

func scanRow(r pgx.Row) (Scan, error) {
	var sc Scan
	var inTeam, complete bool
	var deltas []byte
	var teamMin *int
	var teamCount int
	if err := r.Scan(&sc.ID, &sc.StationID, &sc.StationName, &sc.ScannedAt, &sc.Rating, &inTeam, &complete, &deltas, &teamMin, &teamCount); err != nil {
		return sc, err
	}
	if len(deltas) > 2 {
		_ = json.Unmarshal(deltas, &sc.Deltas)
	}
	sc.Team = complete
	sc.TeamStatus = "none"
	if inTeam {
		sc.TeamStatus = "pending"
		if complete {
			sc.TeamStatus = "complete"
		}
		if teamMin != nil {
			sc.TeamProgress = &TeamProgress{Scanned: teamCount, Required: *teamMin}
		}
	}
	sc.RateableUntil = sc.ScannedAt.Add(ratingLock)
	return sc, nil
}

func (s *Service) scanByID(ctx context.Context, userID, scanID string) (Scan, error) {
	sc, err := scanRow(s.DB.QueryRow(ctx, scanSelect+` where sc.id = $1 and sc.user_id = $2`, scanID, userID))
	if isNoRows(err) {
		return sc, ErrNotFound
	}
	return sc, wrap("scan", err)
}

func (s *Service) userScans(ctx context.Context, userID, eventID string) ([]Scan, error) {
	rows, err := s.DB.Query(ctx, scanSelect+` where sc.user_id = $1 and sc.event_id = $2 order by sc.scanned_at desc`, userID, eventID)
	if err != nil {
		return nil, wrap("scans", err)
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Scan, error) { return scanRow(r) })
}

// recomputeDeltas freezes the points of a scan from its rating, team status and station weights.
func (s *Service) recomputeDeltas(ctx context.Context, q pgx.Tx, scanID string, cfg game.Config) error {
	var rating *int
	var team bool
	var base int
	var weights []byte
	if err := q.QueryRow(ctx, `select sc.rating, sc.team_complete, st.base_points, st.weights from scans sc join stations st on st.id = sc.station_id where sc.id = $1`, scanID).Scan(&rating, &team, &base, &weights); err != nil {
		return err
	}
	var w game.Vector
	if err := json.Unmarshal(weights, &w); err != nil {
		return err
	}
	d := game.Deltas(game.Input{BasePoints: float64(base), Weights: w, Rating: rating, Team: team}, cfg)
	b, _ := json.Marshal(d)
	_, err := q.Exec(ctx, `update scans set deltas = $2, xp = $3 where id = $1`, scanID, b, game.XP(d))
	return err
}

// Rate records the 1–5 rating (changeable for 15 minutes after the scan).
func (s *Service) Rate(ctx context.Context, userID, scanID string, rating int) (RatingResult, error) {
	if rating < 1 || rating > 5 {
		return RatingResult{}, ErrInvalidRating
	}
	var eventID string
	var current *int
	var scannedAt time.Time
	err := s.DB.QueryRow(ctx, `select event_id::text, rating, scanned_at from scans where id = $1 and user_id = $2`, scanID, userID).Scan(&eventID, &current, &scannedAt)
	if isNoRows(err) {
		return RatingResult{}, ErrNotFound
	}
	if err != nil {
		return RatingResult{}, wrap("rate", err)
	}
	if current != nil && s.Now().Sub(scannedAt) > ratingLock {
		return RatingResult{}, ErrRatingLocked
	}
	ev, err := s.eventByID(ctx, eventID)
	if err != nil {
		return RatingResult{}, err
	}
	before, err := s.Progress(ctx, userID, ev)
	if err != nil {
		return RatingResult{}, err
	}
	err = s.tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `update scans set rating = $2, rated_at = now() where id = $1`, scanID, rating); err != nil {
			return err
		}
		return s.recomputeDeltas(ctx, tx, scanID, ev.Config)
	})
	if err != nil {
		return RatingResult{}, wrap("rate", err)
	}
	s.awardBadges(ctx, userID, ev)
	after, err := s.Progress(ctx, userID, ev)
	if err != nil {
		return RatingResult{}, err
	}
	if !before.CardUnlocked && after.CardUnlocked {
		s.Notify(ctx, userID, "card", "Karta postaci odblokowana!", "Zobacz, jaką ścieżką podążasz.", "/card")
	}
	sc, err := s.scanByID(ctx, userID, scanID)
	if err != nil {
		return RatingResult{}, err
	}
	s.touchTeamScore(ctx, userID, ev.ID)
	return RatingResult{Scan: sc, Progress: after}, nil
}

// Progress computes a player's stats, level, archetypes and badges in an event.
func (s *Service) Progress(ctx context.Context, userID string, ev Event) (Progress, error) {
	scans, err := s.userScans(ctx, userID, ev.ID)
	if err != nil {
		return Progress{}, err
	}
	deltas := make([]game.Stats, len(scans))
	rated := 0
	for i, sc := range scans {
		deltas[i] = sc.Deltas
		if sc.Rating != nil {
			rated++
		}
	}
	p := game.BuildProfile(deltas, rated, ev.Config)
	badges, err := s.badges(ctx, userID, ev.ID)
	if err != nil {
		return Progress{}, err
	}
	var title *string
	if p.TitleKey != "" {
		title = &p.TitleKey
	}
	if scans == nil {
		scans = []Scan{}
	}
	return Progress{
		EventID: ev.ID, Stats: p.Stats, XP: p.XP, Level: p.Level, LevelProgress: p.LevelProgress, RatedCount: rated,
		Archetypes: p.Archetypes, TitleKey: title, CardUnlocked: rated >= ev.Config.MinScansForCard,
		MinScansForCard: ev.Config.MinScansForCard, Scans: scans, Badges: badges,
	}, nil
}

func (s *Service) Character(ctx context.Context, userID string, ev Event) (Character, error) {
	p, err := s.Progress(ctx, userID, ev)
	if err != nil {
		return Character{}, err
	}
	return Character{
		Unlocked: p.CardUnlocked,
		Progress: p,
		Programs: game.Recommend(p.Stats, s.Catalog.ProgramItems(), 3),
		Clubs:    game.Recommend(p.Stats, s.Catalog.ClubItems(), 3),
	}, nil
}

// ---------- badges (spec §8.8; same rules as the demo backend) ----------

var badgeTitles = map[string]string{
	"first-scan":    "Pierwszy krok",
	"faculties-all": "Odkrywca wydziałów",
	"bookworm":      "Mól książkowy",
	"team-spirit":   "Duch drużyny",
	"early-bird":    "Ranny ptaszek",
	"card-unlocked": "Karta postaci",
}

func (s *Service) badges(ctx context.Context, userID, eventID string) ([]Badge, error) {
	rows, err := s.DB.Query(ctx, `select badge, awarded_at from badges_awarded where user_id = $1 and event_id = $2 order by awarded_at`, userID, eventID)
	if err != nil {
		return nil, wrap("badges", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Badge, error) {
		var b Badge
		err := r.Scan(&b.ID, &b.AwardedAt)
		return b, err
	})
	if out == nil {
		out = []Badge{}
	}
	return out, err
}

func (s *Service) awardBadges(ctx context.Context, userID string, ev Event) {
	loc, err := time.LoadLocation(ev.Timezone)
	if err != nil {
		loc = time.UTC
	}
	var scans, rated, teamDone, early, faculties, eventFaculties int
	var library bool
	err = s.DB.QueryRow(ctx, `select count(*), count(sc.rating), count(*) filter (where sc.team_complete),
			count(*) filter (where extract(hour from sc.scanned_at at time zone $3) < 11),
			count(distinct st.faculty), coalesce(bool_or(st.poi = 'cizt'), false),
			(select count(distinct faculty) from stations where event_id = $2 and faculty is not null)
		from scans sc join stations st on st.id = sc.station_id where sc.user_id = $1 and sc.event_id = $2`,
		userID, ev.ID, loc.String()).Scan(&scans, &rated, &teamDone, &early, &faculties, &library, &eventFaculties)
	if err != nil && !isNoRows(err) {
		s.Log.Warn("badge stats", "err", err)
		return
	}
	earned := map[string]bool{
		"first-scan":    scans > 0,
		"faculties-all": eventFaculties >= 2 && faculties >= eventFaculties,
		"bookworm":      library,
		"team-spirit":   teamDone >= 3,
		"early-bird":    early > 0,
		"card-unlocked": rated >= ev.Config.MinScansForCard,
	}
	for _, id := range []string{"first-scan", "faculties-all", "bookworm", "team-spirit", "early-bird", "card-unlocked"} {
		if !earned[id] {
			continue
		}
		tag, err := s.DB.Exec(ctx, `insert into badges_awarded (user_id, event_id, badge) values ($1, $2, $3) on conflict do nothing`, userID, ev.ID, id)
		if err == nil && tag.RowsAffected() == 1 {
			s.Notify(ctx, userID, "badge", "Nowa odznaka: "+badgeTitles[id], "Sprawdź ją w panelu bohatera.", "/hero")
		}
	}
}
