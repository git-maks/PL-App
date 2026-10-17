package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/git-maks/pl-app/apps/api/internal/game"
	"github.com/git-maks/pl-app/apps/api/internal/qr"
)

// canManage reports whether the user may operate the station (assigned staff or admin).
func (s *Service) canManage(ctx context.Context, user Me, stationID string) bool {
	if user.Role == "admin" {
		return true
	}
	if user.Role != "staff" {
		return false
	}
	var ok bool
	_ = s.DB.QueryRow(ctx, `select exists(select 1 from station_staff where station_id = $1 and user_id = $2)`, stationID, user.ID).Scan(&ok)
	return ok
}

func (s *Service) staffDTO(ctx context.Context, st Station) (StaffStation, error) {
	open, until := s.scanOpen(st, s.Now())
	out := StaffStation{Station: st, ScanOpen: open, ScanOpenUntil: until}
	err := s.DB.QueryRow(ctx, `select count(*), count(rating), avg(rating)::float8 from scans where station_id = $1`, st.ID).Scan(&out.ScansCount, &out.RatedCount, &out.AvgRating)
	return out, err
}

func (s *Service) StaffStations(ctx context.Context, user Me) ([]StaffStation, error) {
	if user.Role != "staff" && user.Role != "admin" {
		return nil, ErrForbidden
	}
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return nil, err
	}
	stations, err := s.Stations(ctx, ev.ID)
	if err != nil {
		return nil, err
	}
	out := []StaffStation{}
	for _, st := range stations {
		if !s.canManage(ctx, user, st.ID) {
			continue
		}
		dto, err := s.staffDTO(ctx, st)
		if err != nil {
			return nil, wrap("staff station", err)
		}
		out = append(out, dto)
	}
	return out, nil
}

// StationQR returns the current rotating payload for the station display (spec §7.4). The display
// never sees the station secret.
func (s *Service) StationQR(ctx context.Context, user Me, stationID string) (StationQR, error) {
	if !s.canManage(ctx, user, stationID) {
		return StationQR{}, ErrForbidden
	}
	st, err := s.station(ctx, stationID)
	if err != nil {
		return StationQR{}, err
	}
	now := s.Now()
	w := qr.Window(now, qr.PeriodSeconds)
	open, _ := s.scanOpen(st, now)
	var count int
	_ = s.DB.QueryRow(ctx, `select count(*) from scans where station_id = $1`, st.ID).Scan(&count)
	return StationQR{
		Payload:       qr.RotatingURL(s.Cfg.BaseURL, st.ID, w, qr.RotatingMAC(st.secret, st.ID, w)),
		ManualCode:    qr.ManualCode(st.secret, st.ID, qr.Window(now, qr.ManualPeriodSeconds)),
		ValidUntil:    time.Unix((w+1)*qr.PeriodSeconds, 0).UTC(),
		PeriodSeconds: qr.PeriodSeconds,
		ScanOpen:      open,
		ScansCount:    count,
	}, nil
}

// StaticPoster returns the printable static payload (fallback for stations without a screen).
func (s *Service) StaticPoster(ctx context.Context, user Me, stationID string) (string, error) {
	if !s.canManage(ctx, user, stationID) {
		return "", ErrForbidden
	}
	st, err := s.station(ctx, stationID)
	if err != nil {
		return "", err
	}
	return qr.StaticURL(s.Cfg.BaseURL, st.ID, qr.StaticMAC(st.secret, st.ID)), nil
}

func (s *Service) SetScanWindow(ctx context.Context, user Me, stationID string, open bool, minutes int) (StaffStation, error) {
	if !s.canManage(ctx, user, stationID) {
		return StaffStation{}, ErrForbidden
	}
	if minutes <= 0 || minutes > 120 {
		minutes = 15
	}
	var err error
	if open {
		_, err = s.DB.Exec(ctx, `update stations set manual_state = 'open', manual_until = $2 where id = $1`, stationID, s.Now().Add(time.Duration(minutes)*time.Minute))
	} else {
		_, err = s.DB.Exec(ctx, `update stations set manual_state = 'closed', manual_until = null where id = $1`, stationID)
	}
	if err != nil {
		return StaffStation{}, wrap("scan window", err)
	}
	s.audit(ctx, user.ID, "scan_window", stationID, map[string]any{"open": open, "minutes": minutes})
	st, err := s.station(ctx, stationID)
	if err != nil {
		return StaffStation{}, err
	}
	return s.staffDTO(ctx, st)
}

// ---------- admin ----------

func (s *Service) PatchStation(ctx context.Context, user Me, stationID string, p StationPatch) (Station, error) {
	if user.Role != "admin" {
		return Station{}, ErrForbidden
	}
	err := s.tx(ctx, func(tx pgx.Tx) error {
		if p.BasePoints != nil {
			if *p.BasePoints < 1 || *p.BasePoints > 1000 {
				return ErrBadRequest
			}
			if _, err := tx.Exec(ctx, `update stations set base_points = $2 where id = $1`, stationID, *p.BasePoints); err != nil {
				return err
			}
		}
		if p.Weights != nil {
			sum := 0.0
			for k, v := range p.Weights {
				if !game.IsCode(k) || v < 0 {
					return ErrBadRequest
				}
				sum += v
			}
			if sum <= 0 {
				return ErrBadRequest
			}
			norm := game.Vector{}
			for k, v := range p.Weights {
				if v > 0 {
					norm[k] = v / sum
				}
			}
			b, _ := json.Marshal(norm)
			if _, err := tx.Exec(ctx, `update stations set weights = $2 where id = $1`, stationID, b); err != nil {
				return err
			}
		}
		if p.TeamMinMembers != nil {
			var v *int
			if *p.TeamMinMembers > 0 {
				v = p.TeamMinMembers
			}
			if _, err := tx.Exec(ctx, `update stations set team_min_members = $2 where id = $1`, stationID, v); err != nil {
				return err
			}
		}
		if p.StaffEmails != nil {
			if _, err := tx.Exec(ctx, `delete from station_staff where station_id = $1`, stationID); err != nil {
				return err
			}
			for _, e := range p.StaffEmails {
				if _, err := tx.Exec(ctx, `insert into station_staff (station_id, user_id) select $1, id from users where email = $2 and email_verified_at is not null on conflict do nothing`, stationID, strings.ToLower(strings.TrimSpace(e))); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `update users set role = 'staff' where email = $1 and role = 'participant'`, strings.ToLower(strings.TrimSpace(e))); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return Station{}, wrap("patch station", err)
	}
	s.audit(ctx, user.ID, "patch_station", stationID, map[string]any{"patch": p})
	return s.station(ctx, stationID)
}

// Analytics returns aggregated, anonymous statistics for the organisers.
func (s *Service) Analytics(ctx context.Context, user Me, eventID string) (Analytics, error) {
	if user.Role != "admin" {
		return Analytics{}, ErrForbidden
	}
	ev, err := s.eventByID(ctx, eventID)
	if err != nil {
		return Analytics{}, err
	}
	var a Analytics
	if err := s.DB.QueryRow(ctx, `select count(distinct user_id), count(*), count(rating) from scans where event_id = $1`, eventID).Scan(&a.Participants, &a.Scans, &a.Rated); err != nil {
		return a, wrap("analytics", err)
	}
	rows, err := s.DB.Query(ctx, `select st.id::text, st.name, count(sc.id), count(sc.rating), avg(sc.rating)::float8
		from stations st left join scans sc on sc.station_id = st.id where st.event_id = $1 group by st.id order by st.name`, eventID)
	if err != nil {
		return a, wrap("analytics", err)
	}
	a.Stations, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (AnalyticsStation, error) {
		var x AnalyticsStation
		err := r.Scan(&x.StationID, &x.Name, &x.Visits, &x.Rated, &x.AvgRating)
		return x, err
	})
	if err != nil {
		return a, wrap("analytics", err)
	}
	// Dominant archetype per player, computed from frozen per-visit points.
	urows, err := s.DB.Query(ctx, `select user_id::text, deltas from scans where event_id = $1 and rating is not null`, eventID)
	if err != nil {
		return a, wrap("analytics", err)
	}
	perUser := map[string][]game.Stats{}
	for urows.Next() {
		var id string
		var d game.Stats
		var raw []byte
		if err := urows.Scan(&id, &raw); err != nil {
			urows.Close()
			return a, err
		}
		_ = json.Unmarshal(raw, &d)
		perUser[id] = append(perUser[id], d)
	}
	urows.Close()
	counts := map[string]int{}
	for _, ds := range perUser {
		p := game.BuildProfile(ds, len(ds), ev.Config)
		if len(p.Archetypes) > 0 {
			counts[p.Archetypes[0].Code]++
		}
	}
	for _, c := range game.Codes {
		a.Archetypes = append(a.Archetypes, AnalyticsArchetype{Code: c, Users: counts[c]})
	}
	return a, nil
}

// AnalyticsCSV writes the station aggregates as CSV (UTF-8 with BOM for spreadsheet apps).
func (s *Service) AnalyticsCSV(ctx context.Context, user Me, eventID string, w io.Writer) error {
	a, err := s.Analytics(ctx, user, eventID)
	if err != nil {
		return err
	}
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.Write([]string{"stacja", "wizyty", "ocenione", "srednia_ocena"})
	for _, st := range a.Stations {
		avg := ""
		if st.AvgRating != nil {
			avg = strconv.FormatFloat(*st.AvgRating, 'f', 2, 64)
		}
		_ = cw.Write([]string{st.Name, strconv.Itoa(st.Visits), strconv.Itoa(st.Rated), avg})
	}
	_ = cw.Write(nil)
	_ = cw.Write([]string{"archetyp_dominujacy", "liczba_graczy"})
	for _, ar := range a.Archetypes {
		_ = cw.Write([]string{ar.Code, strconv.Itoa(ar.Users)})
	}
	cw.Flush()
	return cw.Error()
}
