package push_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/git-maks/pl-app/apps/api/internal/app"
	"github.com/git-maks/pl-app/apps/api/internal/push"
	"github.com/git-maks/pl-app/apps/api/internal/store"
)

// A fake push service receives the encrypted message; a 410 response removes the subscription.
func TestSendAndPruneExpired(t *testing.T) {
	url := os.Getenv("UQ_TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://uniquest@127.0.0.1:5432/uniquest_test?sslmode=disable"
	}
	ctx := context.Background()
	db, err := store.Open(ctx, url)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	defer db.Close()
	if _, err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var userID string
	if err := db.QueryRow(ctx, `insert into users (kind, nickname) values ('guest', 'Push Test') returning id::text`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, `delete from users where id = $1`, userID)

	var hits atomic.Int32
	var gone atomic.Bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Content-Encoding") != "aes128gcm" || !strings.HasPrefix(r.Header.Get("Authorization"), "vapid ") || len(body) == 0 {
			t.Errorf("unexpected push request: %v", r.Header)
		}
		if gone.Load() {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	// A real browser key pair for the subscription (P-256 + 16-byte auth secret).
	key, _ := ecdh.P256().GenerateKey(rand.Reader)
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	enc := base64.RawURLEncoding.EncodeToString
	if _, err := db.Exec(ctx, `insert into push_subscriptions (user_id, endpoint, p256dh, auth) values ($1, $2, $3, $4)`,
		userID, srv.URL+"/sub/1", enc(key.PublicKey().Bytes()), enc(auth)); err != nil {
		t.Fatal(err)
	}
	priv, pub, err := push.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	s := push.New(db, slog.New(slog.NewTextHandler(io.Discard, nil)), pub, priv, "mailto:test@example.com")
	s.SetHTTPClient(srv.Client())

	s.Send(ctx, userID, app.PushMessage{Title: "Za 10 minut", Body: "Laboratorium OZE"})
	if hits.Load() != 1 {
		t.Fatalf("push service hits = %d", hits.Load())
	}
	gone.Store(true)
	s.Send(ctx, userID, app.PushMessage{Title: "x"})
	var n int
	_ = db.QueryRow(ctx, `select count(*) from push_subscriptions where user_id = $1`, userID).Scan(&n)
	if n != 0 {
		t.Fatalf("expired subscription not pruned (%d left)", n)
	}
}
