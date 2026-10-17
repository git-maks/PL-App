package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/git-maks/pl-app/apps/api/internal/catalog"
	"github.com/git-maks/pl-app/apps/api/internal/game"
)

const eventColumns = `id::text, slug, name, description, starts_at, ends_at, timezone, status, config, team_min, team_max, queued_grace_seconds`

func scanEvent(row pgx.Row) (Event, error) {
	var e Event
	var cfg []byte
	var grace int
	if err := row.Scan(&e.ID, &e.Slug, &e.Name, &e.Description, &e.StartsAt, &e.EndsAt, &e.Timezone, &e.Status, &cfg, &e.TeamMin, &e.TeamMax, &grace); err != nil {
		return e, err
	}
	e.queuedGrace = time.Duration(grace) * time.Second
	return e, json.Unmarshal(cfg, &e.Config)
}

// ActiveEvent returns the active event (the most recently started one if several are active).
func (s *Service) ActiveEvent(ctx context.Context) (Event, error) {
	e, err := scanEvent(s.DB.QueryRow(ctx, `select `+eventColumns+` from events where status = 'active' order by starts_at desc limit 1`))
	if isNoRows(err) {
		return e, ErrNoActiveEvent
	}
	return e, wrap("active event", err)
}

func (s *Service) eventByID(ctx context.Context, id string) (Event, error) {
	e, err := scanEvent(s.DB.QueryRow(ctx, `select `+eventColumns+` from events where id = $1`, id))
	if isNoRows(err) {
		return e, ErrNotFound
	}
	return e, wrap("event", err)
}

const stationColumns = `s.id::text, s.event_id::text, s.slug, s.name, s.description, s.kind, s.faculty, s.poi, s.base_points, s.weights, s.team_min_members, s.qr_secret, s.manual_state, s.manual_until`

func scanStation(row pgx.Row) (Station, error) {
	var st Station
	var weights []byte
	var teamMin *int
	err := row.Scan(&st.ID, &st.eventID, &st.Slug, &st.Name, &st.Description, &st.Kind, &st.Faculty, &st.Poi, &st.BasePoints, &weights, &teamMin, &st.secret, &st.manualState, &st.manualUntil)
	if err != nil {
		return st, err
	}
	if teamMin != nil {
		st.Team = &TeamRule{MinMembers: *teamMin}
	}
	st.Sessions = []Session{}
	return st, json.Unmarshal(weights, &st.Weights)
}

// Stations returns an event's stations with their sessions (agenda), ordered by first session.
func (s *Service) Stations(ctx context.Context, eventID string) ([]Station, error) {
	rows, err := s.DB.Query(ctx, `select `+stationColumns+` from stations s where s.event_id = $1 order by s.name`, eventID)
	if err != nil {
		return nil, wrap("stations", err)
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Station, error) { return scanStation(r) })
	if err != nil {
		return nil, wrap("stations", err)
	}
	byID := make(map[string]int, len(list))
	for i := range list {
		byID[list[i].ID] = i
	}
	srows, err := s.DB.Query(ctx, `select ss.id::text, ss.station_id::text, ss.room, ss.starts_at, ss.ends_at, ss.scan_opens_at, ss.scan_closes_at
		from station_sessions ss join stations s on s.id = ss.station_id where s.event_id = $1 order by ss.starts_at`, eventID)
	if err != nil {
		return nil, wrap("sessions", err)
	}
	sessions, err := pgx.CollectRows(srows, func(r pgx.CollectableRow) (Session, error) {
		var x Session
		err := r.Scan(&x.ID, &x.StationID, &x.Room, &x.StartsAt, &x.EndsAt, &x.ScanOpensAt, &x.ScanClosesAt)
		return x, err
	})
	if err != nil {
		return nil, wrap("sessions", err)
	}
	for _, x := range sessions {
		if i, ok := byID[x.StationID]; ok {
			list[i].Sessions = append(list[i].Sessions, x)
		}
	}
	return list, nil
}

func (s *Service) station(ctx context.Context, id string) (Station, error) {
	st, err := scanStation(s.DB.QueryRow(ctx, `select `+stationColumns+` from stations s where s.id = $1`, id))
	if isNoRows(err) {
		return st, ErrNotFound
	}
	if err != nil {
		return st, wrap("station", err)
	}
	all, err := s.DB.Query(ctx, `select id::text, station_id::text, room, starts_at, ends_at, scan_opens_at, scan_closes_at from station_sessions where station_id = $1 order by starts_at`, id)
	if err != nil {
		return st, wrap("station sessions", err)
	}
	st.Sessions, err = pgx.CollectRows(all, func(r pgx.CollectableRow) (Session, error) {
		var x Session
		err := r.Scan(&x.ID, &x.StationID, &x.Room, &x.StartsAt, &x.EndsAt, &x.ScanOpensAt, &x.ScanClosesAt)
		return x, err
	})
	return st, wrap("station sessions", err)
}

// scanOpen decides whether the station accepts scans at time t: a manual close wins, then a manual
// open window, then the session schedule (spec §7.4).
func (s *Service) scanOpen(st Station, t time.Time) (bool, *time.Time) {
	if s.Cfg.OpenAllStations {
		return true, nil
	}
	if st.manualState != nil && *st.manualState == "closed" {
		return false, nil
	}
	if st.manualState != nil && *st.manualState == "open" && st.manualUntil != nil && t.Before(*st.manualUntil) {
		return true, st.manualUntil
	}
	for _, x := range st.Sessions {
		if !t.Before(x.ScanOpensAt) && !t.After(x.ScanClosesAt) {
			until := x.ScanClosesAt
			return true, &until
		}
	}
	return false, nil
}

// SeedOptions controls SeedEvent.
type SeedOptions struct {
	Date     time.Time // day of the event (date part, in the event's time zone)
	Activate bool
}

// SeedEvent creates (or replaces) the demo event from content/demo-event.json on the given day.
func (s *Service) SeedEvent(ctx context.Context, ev catalog.DemoEvent, opt SeedOptions) (string, error) {
	loc, err := time.LoadLocation(ev.Timezone)
	if err != nil {
		return "", fmt.Errorf("timezone %q: %w", ev.Timezone, err)
	}
	day := time.Date(opt.Date.Year(), opt.Date.Month(), opt.Date.Day(), 0, 0, 0, 0, loc)
	at := func(hhmm string) (time.Time, error) {
		t, err := time.ParseInLocation("15:04", hhmm, loc)
		if err != nil {
			return time.Time{}, err
		}
		return day.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute), nil
	}
	start, err := at(ev.Day.Start)
	if err != nil {
		return "", err
	}
	end, err := at(ev.Day.End)
	if err != nil {
		return "", err
	}
	defaults := s.Catalog.Archetypes.Defaults
	cfg := defaults.Config
	if ev.MinScansForCard > 0 {
		cfg.MinScansForCard = ev.MinScansForCard
	}
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	cfgJSON, _ := json.Marshal(cfg)
	status := "draft"
	if opt.Activate {
		status = "active"
	}
	var eventID string
	err = s.tx(ctx, func(tx pgx.Tx) error {
		if opt.Activate {
			if _, err := tx.Exec(ctx, `update events set status = 'finished' where status = 'active' and slug <> $1`, ev.Slug); err != nil {
				return err
			}
		}
		if err := tx.QueryRow(ctx, `insert into events (slug, name, description, timezone, starts_at, ends_at, status, config)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
			on conflict (slug) do update set name = excluded.name, description = excluded.description, timezone = excluded.timezone,
				starts_at = excluded.starts_at, ends_at = excluded.ends_at, status = excluded.status, config = excluded.config
			returning id::text`, ev.Slug, ev.Name, ev.Description, ev.Timezone, start, end, status, cfgJSON).Scan(&eventID); err != nil {
			return err
		}
		for _, st := range ev.Stations {
			for code := range st.Weights {
				if !game.IsCode(code) {
					return fmt.Errorf("station %s: unknown stat %q", st.Slug, code)
				}
			}
			weights, _ := json.Marshal(st.Weights)
			var teamMin *int
			if st.Team != nil && st.Team.Mode {
				teamMin = &st.Team.MinMembers
			}
			secret := make([]byte, 32)
			if _, err := rand.Read(secret); err != nil {
				return err
			}
			var stationID string
			// Re-seeding keeps the station id and secret (printed posters stay valid).
			if err := tx.QueryRow(ctx, `insert into stations (event_id, slug, name, description, kind, faculty, poi, base_points, weights, team_min_members, qr_secret)
				values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
				on conflict (event_id, slug) do update set name = excluded.name, description = excluded.description, kind = excluded.kind,
					faculty = excluded.faculty, poi = excluded.poi, base_points = excluded.base_points, weights = excluded.weights,
					team_min_members = excluded.team_min_members
				returning id::text`, eventID, st.Slug, st.Name, st.Description, st.Kind, st.Faculty, st.Poi, defaults.BasePoints, weights, teamMin, secret).Scan(&stationID); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `delete from station_sessions where station_id = $1`, stationID); err != nil {
				return err
			}
			for _, x := range st.Sessions {
				a, err := at(x.Start)
				if err != nil {
					return err
				}
				b, err := at(x.End)
				if err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `insert into station_sessions (station_id, room, starts_at, ends_at, scan_opens_at, scan_closes_at)
					values ($1, $2, $3, $4, $5, $6)`, stationID, x.Room, a, b, b.Add(-10*time.Minute), b.Add(15*time.Minute)); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return eventID, wrap("seed event", err)
}

// EnsureAdmin creates (or promotes) a registered admin account with the given e-mail.
func (s *Service) EnsureAdmin(ctx context.Context, email string) error {
	e, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `insert into users (kind, nickname, email, email_verified_at, role, consents)
		values ('registered', 'Organizator', $1, now(), 'admin', '{"terms": true, "notifications": false}')
		on conflict (email) do update set role = 'admin', email_verified_at = coalesce(users.email_verified_at, now())`, e)
	return wrap("ensure admin", err)
}
