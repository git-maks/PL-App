// Package push sends Web Push notifications (RFC 8291/8292) with VAPID keys.
package push

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/git-maks/pl-app/apps/api/internal/app"
)

type Sender struct {
	db         *pgxpool.Pool
	log        *slog.Logger
	public     string
	private    string
	subscriber string
	client     *http.Client
}

func New(db *pgxpool.Pool, log *slog.Logger, public, private, subject string) *Sender {
	return &Sender{db: db, log: log, public: public, private: private, subscriber: subject, client: &http.Client{Timeout: 10 * time.Second}}
}

// SetHTTPClient replaces the HTTP client (tests, custom proxies).
func (s *Sender) SetHTTPClient(c *http.Client) { s.client = c }

func (s *Sender) Enabled() bool     { return s.public != "" && s.private != "" }
func (s *Sender) PublicKey() string { return s.public }

// Send delivers to all of the user's subscriptions and prunes expired ones (404/410).
func (s *Sender) Send(ctx context.Context, userID string, msg app.PushMessage) {
	if !s.Enabled() {
		return
	}
	rows, err := s.db.Query(ctx, `select id, endpoint, p256dh, auth from push_subscriptions where user_id = $1`, userID)
	if err != nil {
		s.log.Warn("push: load subscriptions", "err", err)
		return
	}
	type sub struct {
		id                     int64
		endpoint, p256dh, auth string
	}
	subs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (sub, error) {
		var x sub
		err := r.Scan(&x.id, &x.endpoint, &x.p256dh, &x.auth)
		return x, err
	})
	if err != nil {
		return
	}
	body, _ := json.Marshal(msg)
	for _, x := range subs {
		res, err := webpush.SendNotificationWithContext(ctx, body, &webpush.Subscription{
			Endpoint: x.endpoint,
			Keys:     webpush.Keys{P256dh: x.p256dh, Auth: x.auth},
		}, &webpush.Options{
			HTTPClient:      s.client,
			Subscriber:      s.subscriber,
			VAPIDPublicKey:  s.public,
			VAPIDPrivateKey: s.private,
			TTL:             3600,
			Urgency:         webpush.UrgencyNormal,
		})
		if err != nil {
			s.log.Warn("push: send", "err", err)
			continue
		}
		_ = res.Body.Close()
		switch {
		case res.StatusCode == http.StatusGone || res.StatusCode == http.StatusNotFound:
			_, _ = s.db.Exec(ctx, `delete from push_subscriptions where id = $1`, x.id)
		case res.StatusCode < 300:
			_, _ = s.db.Exec(ctx, `update push_subscriptions set last_ok_at = now() where id = $1`, x.id)
		default:
			s.log.Warn("push: rejected", "status", res.StatusCode)
		}
	}
}

// GenerateKeys returns a new VAPID key pair (base64url).
func GenerateKeys() (private, public string, err error) { return webpush.GenerateVAPIDKeys() }
