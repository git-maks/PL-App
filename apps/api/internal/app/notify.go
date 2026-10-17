package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Notify stores an in-app notification and delivers it live (SSE) and as Web Push.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body, url string) {
	var n Notification
	var u *string
	if url != "" {
		full := s.Cfg.BasePath() + url
		u = &full
	}
	err := s.DB.QueryRow(ctx, `insert into notifications (user_id, kind, title, body, url) values ($1, $2, $3, $4, $5)
		returning id::text, kind, title, body, url, created_at, read_at`, userID, kind, title, body, url).
		Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.URL, &n.CreatedAt, &n.ReadAt)
	if err != nil {
		s.Log.Warn("notify", "err", err)
		return
	}
	n.URL = u
	s.Pub.Publish(ctx, []string{userID}, "notification", n)
	if s.Push != nil && s.Push.Enabled() {
		go s.Push.Send(context.WithoutCancel(ctx), userID, PushMessage{Title: title, Body: body, URL: derefOr(u, s.Cfg.BasePath()+"/hero"), Tag: kind})
	}
}

func derefOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

func (s *Service) Notifications(ctx context.Context, userID string) ([]Notification, error) {
	rows, err := s.DB.Query(ctx, `select id::text, kind, title, body, url, created_at, read_at from notifications where user_id = $1 order by created_at desc limit 50`, userID)
	if err != nil {
		return nil, wrap("notifications", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Notification, error) {
		var n Notification
		err := r.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.URL, &n.CreatedAt, &n.ReadAt)
		return n, err
	})
	if out == nil {
		out = []Notification{}
	}
	for i := range out {
		if out[i].URL != nil {
			full := s.Cfg.BasePath() + *out[i].URL
			out[i].URL = &full
		}
	}
	return out, wrap("notifications", err)
}

func (s *Service) MarkNotificationsRead(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(ctx, `update notifications set read_at = now() where user_id = $1 and read_at is null`, userID)
	return wrap("mark read", err)
}

// ---------- push subscriptions & reminders ----------

func (s *Service) SavePushSubscription(ctx context.Context, userID string, sub PushSubscription) error {
	if len(sub.Endpoint) > 1024 || !(len(sub.Endpoint) > 8 && sub.Endpoint[:8] == "https://") || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		return ErrBadRequest
	}
	_, err := s.DB.Exec(ctx, `insert into push_subscriptions (user_id, endpoint, p256dh, auth) values ($1, $2, $3, $4)
		on conflict (endpoint) do update set user_id = excluded.user_id, p256dh = excluded.p256dh, auth = excluded.auth`, userID, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth)
	return wrap("push subscribe", err)
}

func (s *Service) DeletePushSubscription(ctx context.Context, userID, endpoint string) error {
	_, err := s.DB.Exec(ctx, `delete from push_subscriptions where user_id = $1 and endpoint = $2`, userID, endpoint)
	return wrap("push unsubscribe", err)
}

const reminderLead = 10 * time.Minute

func (s *Service) SetReminder(ctx context.Context, userID, sessionID string, on bool) error {
	if !on {
		_, err := s.DB.Exec(ctx, `delete from session_reminders where user_id = $1 and session_id = $2`, userID, sessionID)
		return wrap("reminder", err)
	}
	tag, err := s.DB.Exec(ctx, `insert into session_reminders (user_id, session_id, notify_at)
		select $1, id, starts_at - $3::interval from station_sessions where id = $2 and starts_at > now()
		on conflict (user_id, session_id) do nothing`, userID, sessionID, reminderLead.String())
	if err != nil {
		return wrap("reminder", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		_ = s.DB.QueryRow(ctx, `select exists(select 1 from session_reminders where user_id = $1 and session_id = $2)`, userID, sessionID).Scan(&exists)
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Service) Reminders(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.DB.Query(ctx, `select session_id::text from session_reminders where user_id = $1`, userID)
	if err != nil {
		return nil, wrap("reminders", err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if out == nil {
		out = []string{}
	}
	return out, err
}

// SendDueReminders notifies players 10 minutes before sessions they asked to be reminded about.
// Safe to run on several instances: rows are claimed with SKIP LOCKED.
func (s *Service) SendDueReminders(ctx context.Context) (int, error) {
	type due struct {
		userID, station, room string
		startsAt              time.Time
	}
	var list []due
	err := s.tx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select r.user_id::text, st.name, ss.room, ss.starts_at from session_reminders r
			join station_sessions ss on ss.id = r.session_id join stations st on st.id = ss.station_id
			where r.sent_at is null and r.notify_at <= now() for update of r skip locked limit 200`)
		if err != nil {
			return err
		}
		list, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (due, error) {
			var d due
			err := r.Scan(&d.userID, &d.station, &d.room, &d.startsAt)
			return d, err
		})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `update session_reminders set sent_at = now() where sent_at is null and notify_at <= now()`)
		return err
	})
	if err != nil {
		return 0, err
	}
	for _, d := range list {
		where := d.station
		if d.room != "" {
			where += " · " + d.room
		}
		s.Notify(ctx, d.userID, "reminder", "Za 10 minut: "+d.station, "Startuje o "+d.startsAt.In(warsaw()).Format("15:04")+". "+where, "/quests")
	}
	return len(list), nil
}

func warsaw() *time.Location {
	if loc, err := time.LoadLocation("Europe/Warsaw"); err == nil {
		return loc
	}
	return time.UTC
}

// Broadcast sends an organiser announcement to everyone who played the active event.
func (s *Service) Broadcast(ctx context.Context, actorID, title, body string) error {
	if len(title) == 0 || len(title) > 80*4 || len(body) == 0 || len(body) > 280*4 {
		return ErrBadRequest
	}
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(ctx, `select distinct user_id::text from scans where event_id = $1
		union select user_id::text from team_members where event_id = $1`, ev.ID)
	if err != nil {
		return wrap("broadcast", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return wrap("broadcast", err)
	}
	s.audit(ctx, actorID, "broadcast", title, map[string]any{"recipients": len(ids)})
	go func() {
		bg := context.WithoutCancel(ctx)
		for _, id := range ids {
			s.Notify(bg, id, "broadcast", title, body, "")
		}
	}()
	return nil
}
