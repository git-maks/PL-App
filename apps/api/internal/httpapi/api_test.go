package httpapi_test

// Integration tests: real HTTP handlers against a real PostgreSQL (UQ_TEST_DATABASE_URL, default
// postgres://uniquest@127.0.0.1:5432/uniquest_test). Tables are truncated per test; packages that use
// the database must run sequentially (go test -p 1).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/git-maks/pl-app/apps/api/internal/app"
	"github.com/git-maks/pl-app/apps/api/internal/catalog"
	"github.com/git-maks/pl-app/apps/api/internal/config"
	"github.com/git-maks/pl-app/apps/api/internal/game"
	"github.com/git-maks/pl-app/apps/api/internal/httpapi"
	"github.com/git-maks/pl-app/apps/api/internal/qr"
	"github.com/git-maks/pl-app/apps/api/internal/realtime"
	"github.com/git-maks/pl-app/apps/api/internal/store"
)

type env struct {
	t   *testing.T
	srv *httptest.Server
	svc *app.Service
	db  *pgxpool.Pool
	now time.Time
	mu  sync.Mutex
}

func newEnv(t *testing.T, openAll bool) *env {
	t.Helper()
	url := os.Getenv("UQ_TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://uniquest@127.0.0.1:5432/uniquest_test?sslmode=disable"
	}
	ctx := context.Background()
	db, err := store.Open(ctx, url)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(db.Close)
	if _, err := store.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `truncate users, events, login_codes, audit_log cascade`); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{BaseURL: "http://uniquest.test", CookieSecure: false, RateLimits: true, SessionTTL: time.Hour, DevLoginCodes: true, OpenAllStations: openAll, Version: "test"}
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := realtime.NewHub(db, log)
	hctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	go hub.Run(hctx)
	svc := app.New(db, cfg, cat, hub, nil, nil, log)
	e := &env{t: t, svc: svc, db: db, now: time.Now()}
	svc.Now = func() time.Time { e.mu.Lock(); defer e.mu.Unlock(); return e.now }
	if _, err := svc.SeedEvent(ctx, cat.DemoEvent, app.SeedOptions{Date: time.Now(), Activate: true}); err != nil {
		t.Fatal(err)
	}
	api := httpapi.New(svc, hub, cfg, log, nil)
	e.srv = httptest.NewServer(api.Handler())
	t.Cleanup(func() { api.Close(); e.srv.Close() })
	time.Sleep(50 * time.Millisecond) // let the LISTEN connection start
	return e
}

func (e *env) advance(d time.Duration) { e.mu.Lock(); e.now = e.now.Add(d); e.mu.Unlock() }

type client struct {
	e *env
	h *http.Client
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, h: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
}

func (c *client) do(method, path string, body any, out any) (int, map[string]any) {
	c.e.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+"/api/v1"+path, r)
	req.Header.Set("Content-Type", "application/json")
	if method != http.MethodGet {
		req.Header.Set("X-Requested-With", "uniquest")
	}
	res, err := c.h.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var problem map[string]any
	if res.StatusCode >= 400 {
		_ = json.Unmarshal(raw, &problem)
		return res.StatusCode, problem
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			c.e.t.Fatalf("%s %s: %v: %s", method, path, err, raw)
		}
	}
	return res.StatusCode, nil
}

func (c *client) must(method, path string, body any, out any) {
	c.e.t.Helper()
	if st, p := c.do(method, path, body, out); st >= 400 {
		c.e.t.Fatalf("%s %s → %d %v", method, path, st, p)
	}
}

func (e *env) guest(nick string) (*client, app.Me) {
	c := e.client()
	var me app.Me
	c.must("POST", "/auth/guest", map[string]string{"nickname": nick}, &me)
	return c, me
}

type stationInfo struct {
	id     string
	secret []byte
}

func (e *env) station(slug string) stationInfo {
	var s stationInfo
	if err := e.db.QueryRow(context.Background(), `select id::text, qr_secret from stations where slug = $1`, slug).Scan(&s.id, &s.secret); err != nil {
		e.t.Fatal(err)
	}
	return s
}

func (e *env) rotating(slug string, offset int64) string {
	s := e.station(slug)
	w := qr.Window(e.svc.Now(), qr.PeriodSeconds) + offset
	return qr.RotatingURL("http://uniquest.test", s.id, w, qr.RotatingMAC(s.secret, s.id, w))
}

func TestGuestScanRateProgress(t *testing.T) {
	e := newEnv(t, true)
	c, me := e.guest("Ada")
	if me.Kind != "guest" || me.Nickname != "Ada" {
		t.Fatalf("me = %+v", me)
	}
	var res app.ScanResult
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", 0)}, &res)
	if res.AlreadyScanned || res.Scan.Rating != nil {
		t.Fatalf("scan = %+v", res)
	}
	var rr app.RatingResult
	c.must("PUT", "/scans/"+res.Scan.ID+"/rating", map[string]int{"rating": 5}, &rr)
	// WM r=5, weights STR .7 INT .2 KRE .1 → STR 140, INT 40, KRE 20.
	want := game.Stats{140, 40, 0, 20, 0, 0, 0, 0}
	if rr.Progress.Stats != want || rr.Progress.XP != 200 || rr.Progress.Level != 3 {
		t.Fatalf("progress = %+v", rr.Progress)
	}
	if len(rr.Progress.Badges) == 0 {
		t.Error("expected the first-scan badge")
	}
	// Scanning the same station again is idempotent.
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", 0)}, &res)
	if !res.AlreadyScanned {
		t.Error("second scan should report alreadyScanned")
	}
	var notes []app.Notification
	c.must("GET", "/me/notifications", nil, &notes)
	if len(notes) == 0 || !strings.Contains(notes[len(notes)-1].Title, "odznaka") {
		t.Errorf("notifications = %+v", notes)
	}
}

func TestSpecExampleUnlocksCard(t *testing.T) {
	e := newEnv(t, true)
	c, _ := e.guest("Bolek")
	visits := []struct {
		slug   string
		rating int
	}{{"wm-druk-3d", 5}, {"weii-roboty-iot", 3}, {"sport-sztafeta", 4}, {"cizt-escape-room", 2}, {"wbia-makiety-vr", 4}}
	for _, v := range visits {
		var res app.ScanResult
		c.must("POST", "/scans", map[string]any{"payload": e.rotating(v.slug, 0)}, &res)
		c.must("PUT", "/scans/"+res.Scan.ID+"/rating", map[string]int{"rating": v.rating}, nil)
	}
	var ch app.Character
	c.must("GET", "/me/character", nil, &ch)
	if !ch.Unlocked || ch.Progress.TitleKey == nil || *ch.Progress.TitleKey != "STR+KRE" {
		t.Fatalf("character = %+v", ch.Progress)
	}
	want := []game.Share{{Code: "STR", Percent: 45}, {Code: "KRE", Percent: 30}, {Code: "VIT", Percent: 25}}
	if fmt.Sprint(ch.Progress.Archetypes) != fmt.Sprint(want) {
		t.Errorf("archetypes = %v", ch.Progress.Archetypes)
	}
	if len(ch.Programs) != 3 || ch.Programs[0].Slug != "budownictwo" {
		t.Errorf("programs = %+v", ch.Programs)
	}
}

func TestQRValidation(t *testing.T) {
	e := newEnv(t, true)
	c, _ := e.guest("Cezary")
	cases := map[string]string{
		e.rotating("wm-druk-3d", -3): "qr_expired",
		strings.Replace(e.rotating("wm-druk-3d", 0), ".", ".", 1)[:len(e.rotating("wm-druk-3d", 0))-2] + "zz": "qr_invalid",
		"https://example.com/menu": "qr_invalid",
		"ZZZZZZ":                   "qr_invalid",
	}
	for payload, code := range cases {
		st, p := c.do("POST", "/scans", map[string]any{"payload": payload}, nil)
		if st < 400 || p["code"] != code {
			t.Errorf("%s → %d %v, want %s", payload, st, p, code)
		}
	}
	// The previous 30-second window is still accepted.
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", -1)}, nil)
	// Queued (offline) scans are accepted within the grace period.
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("weii-roboty-iot", -10), "queued": true}, nil)
	// Manual code from the station display.
	s := e.station("wbia-makiety-vr")
	code := qr.ManualCode(s.secret, s.id, qr.Window(e.svc.Now(), qr.ManualPeriodSeconds))
	var res app.ScanResult
	c.must("POST", "/scans", map[string]any{"payload": strings.ToLower(code[:3]) + "-" + code[3:]}, &res)
	if res.Scan.StationID != s.id {
		t.Errorf("manual code matched %s", res.Scan.StationID)
	}
}

func TestScanWindowAndRatingLock(t *testing.T) {
	e := newEnv(t, false)
	c, _ := e.guest("Danka")
	admin := e.admin()
	// Move the clock to an hour with no session running (sessions are seeded for today 10:00–16:00).
	e.mu.Lock()
	e.now = time.Date(e.now.Year(), e.now.Month(), e.now.Day(), 6, 0, 0, 0, time.Local)
	e.mu.Unlock()
	st, p := c.do("POST", "/scans", map[string]any{"payload": e.rotating("rektorat-qa", 0)}, nil)
	if st != 422 || p["code"] != "scan_window_closed" {
		t.Fatalf("closed window → %d %v", st, p)
	}
	// Staff opens the window manually.
	var stations []app.StaffStation
	admin.must("GET", "/staff/stations", nil, &stations)
	id := e.station("rektorat-qa").id
	admin.must("POST", "/staff/stations/"+id+"/window", map[string]any{"open": true, "minutes": 10}, nil)
	var res app.ScanResult
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("rektorat-qa", 0)}, &res)
	c.must("PUT", "/scans/"+res.Scan.ID+"/rating", map[string]int{"rating": 2}, nil)
	e.advance(16 * time.Minute)
	if st, p := c.do("PUT", "/scans/"+res.Scan.ID+"/rating", map[string]int{"rating": 5}, nil); st != 409 || p["code"] != "rating_locked" {
		t.Fatalf("late re-rating → %d %v", st, p)
	}
	// The staff display exposes a valid rotating payload.
	var q app.StationQR
	admin.must("GET", "/staff/stations/"+id+"/qr", nil, &q)
	if p := qr.Parse(q.Payload); p.Kind != qr.Rotating || p.StationID != id {
		t.Errorf("qr payload = %s", q.Payload)
	}
	// Participants cannot see the staff panel.
	if st, _ := c.do("GET", "/staff/stations", nil, nil); st != 403 {
		t.Errorf("participant staff access → %d", st)
	}
}

func (e *env) admin() *client {
	if err := e.svc.EnsureAdmin(context.Background(), "admin@pollub.test"); err != nil {
		e.t.Fatal(err)
	}
	c := e.client()
	var out map[string]any
	c.must("POST", "/auth/email/start", map[string]any{"email": "admin@pollub.test", "ageConfirmed": true}, &out)
	c.must("POST", "/auth/email/verify", map[string]any{"email": "admin@pollub.test", "code": out["devCode"]}, nil)
	return c
}

func TestEmailUpgradeKeepsProgress(t *testing.T) {
	e := newEnv(t, true)
	c, me := e.guest("Ewa")
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", 0)}, nil)
	var start map[string]any
	if st, p := c.do("POST", "/auth/email/start", map[string]any{"email": "ewa@example.com"}, nil); st != 400 {
		t.Fatalf("missing age confirmation → %d %v", st, p)
	}
	c.must("POST", "/auth/email/start", map[string]any{"email": "Ewa@Example.com", "ageConfirmed": true}, &start)
	if st, p := c.do("POST", "/auth/email/verify", map[string]any{"email": "ewa@example.com", "code": "000000"}, nil); st != 400 || p["code"] != "invalid_code" {
		t.Fatalf("wrong code → %d %v", st, p)
	}
	var after app.Me
	c.must("POST", "/auth/email/verify", map[string]any{"email": "ewa@example.com", "code": start["devCode"]}, &after)
	if after.ID != me.ID || after.Kind != "registered" || after.Email == nil || *after.Email != "ewa@example.com" {
		t.Fatalf("upgraded = %+v", after)
	}
	var p app.Progress
	c.must("GET", "/me/progress", nil, &p)
	if len(p.Scans) != 1 {
		t.Fatalf("progress lost: %+v", p.Scans)
	}
	// Logging in on a second device with a guest that has its own visit merges both.
	d, _ := e.guest("Ewa2")
	d.must("POST", "/scans", map[string]any{"payload": e.rotating("weii-roboty-iot", 0)}, nil)
	d.must("POST", "/auth/email/start", map[string]any{"email": "ewa@example.com", "ageConfirmed": true}, &start)
	d.must("POST", "/auth/email/verify", map[string]any{"email": "ewa@example.com", "code": start["devCode"]}, &after)
	d.must("GET", "/me/progress", nil, &p)
	if after.ID != me.ID || len(p.Scans) != 2 {
		t.Fatalf("merge: me=%s scans=%d", after.ID, len(p.Scans))
	}
}

func TestTeamsAndLiveEvents(t *testing.T) {
	e := newEnv(t, true)
	a, meA := e.guest("Kapitan")
	b, meB := e.guest("Majtek")
	var team app.Team
	a.must("POST", "/teams", map[string]string{"name": "Bity i Śruby", "emblem": "star"}, &team)
	var inv app.TeamInvite
	a.must("POST", "/teams/"+team.ID+"/invites", nil, &inv)
	if !strings.HasSuffix(inv.URL, "/t/"+inv.Code) {
		t.Fatalf("invite = %+v", inv)
	}
	events := e.stream(a)
	b.must("POST", "/teams/join", map[string]string{"code": inv.URL}, &team)
	if len(team.Members) != 2 || team.CaptainID != meA.ID {
		t.Fatalf("team = %+v", team)
	}
	if ev := events.next(t, "team.member_joined"); !strings.Contains(ev, meB.ID) {
		t.Errorf("member_joined event = %s", ev)
	}
	// Team station (min 2): bonus only once both scanned.
	var ra, rb app.ScanResult
	a.must("POST", "/scans", map[string]any{"payload": e.rotating("wz-startup", 0)}, &ra)
	a.must("PUT", "/scans/"+ra.Scan.ID+"/rating", map[string]int{"rating": 3}, nil)
	if ra.Scan.TeamStatus != "pending" {
		t.Fatalf("first team scan = %+v", ra.Scan)
	}
	b.must("POST", "/scans", map[string]any{"payload": e.rotating("wz-startup", 0)}, &rb)
	if rb.Scan.TeamStatus != "complete" {
		t.Fatalf("second team scan = %+v", rb.Scan)
	}
	events.next(t, "scan.confirmed")
	var p app.Progress
	a.must("GET", "/me/progress", nil, &p)
	// CHA .6 INT .4 × 100 × 1 × 1.2 → CHA 72, INT 48.
	if p.Stats.Get("CHA") != 72 || p.Stats.Get("INT") != 48 {
		t.Fatalf("team bonus not applied: %+v", p.Stats)
	}
	var board app.Leaderboard
	a.must("GET", "/events/"+p.EventID+"/leaderboard?type=team", nil, &board)
	if len(board.Entries) != 1 || !board.Entries[0].IsMe || board.Entries[0].Score != 120 {
		t.Errorf("leaderboard = %+v", board)
	}
	// A third player joins with the same invite; joining twice is refused.
	c, _ := e.guest("Gapa")
	c.must("POST", "/teams/join", map[string]string{"code": inv.Code}, nil)
	if st, p := c.do("POST", "/teams/join", map[string]string{"code": inv.Code}, nil); st != 409 || p["code"] != "already_in_team" {
		t.Errorf("double join → %d %v", st, p)
	}
	if st, p := c.do("POST", "/teams/join", map[string]string{"code": "NOPE1234"}, nil); st != 404 || p["code"] != "invite_invalid" {
		t.Errorf("unknown invite → %d %v", st, p)
	}
	// Only the captain may invite.
	if st, p := b.do("POST", "/teams/"+team.ID+"/invites", nil, nil); st != 403 || p["code"] != "not_captain" {
		t.Errorf("non-captain invite → %d %v", st, p)
	}
	// Captain leaves: captaincy passes to the longest-standing member.
	a.must("POST", "/teams/"+team.ID+"/leave", nil, nil)
	b.must("GET", "/teams/mine", nil, &team)
	if team.CaptainID != meB.ID || len(team.Members) != 2 {
		t.Errorf("after leave = %+v", team)
	}
}

type sse struct {
	ch chan string
}

func (e *env) stream(c *client) *sse {
	req, _ := http.NewRequest("GET", e.srv.URL+"/api/v1/stream", nil)
	h := *c.h
	h.Timeout = 0
	res, err := h.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	s := &sse{ch: make(chan string, 16)}
	go func() {
		defer res.Body.Close()
		sc := bufio.NewScanner(res.Body)
		var typ string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "event: ") {
				typ = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				s.ch <- typ + " " + strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	return s
}

func (s *sse) next(t *testing.T, typ string) string {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case m := <-s.ch:
			if strings.HasPrefix(m, typ+" ") {
				return m
			}
		case <-deadline:
			t.Fatalf("no %s event", typ)
			return ""
		}
	}
}

func TestSecurityAndPrivacy(t *testing.T) {
	e := newEnv(t, true)
	c, me := e.guest("Fryderyk")
	// CSRF: mutations need the custom header and a same-origin Origin.
	req, _ := http.NewRequest("PATCH", e.srv.URL+"/api/v1/me", strings.NewReader(`{"nickname":"Hacker"}`))
	if res, _ := c.h.Do(req); res.StatusCode != 403 {
		t.Errorf("missing X-Requested-With → %d", res.StatusCode)
	}
	req, _ = http.NewRequest("PATCH", e.srv.URL+"/api/v1/me", strings.NewReader(`{"nickname":"Hacker"}`))
	req.Header.Set("X-Requested-With", "uniquest")
	req.Header.Set("Origin", "https://evil.example")
	if res, _ := c.h.Do(req); res.StatusCode != 403 {
		t.Errorf("foreign origin → %d", res.StatusCode)
	}
	if st, p := c.do("PATCH", "/me", map[string]string{"nickname": "x"}, nil); st != 400 || p["code"] != "invalid_nickname" {
		t.Errorf("short nickname → %d %v", st, p)
	}
	// Avatar must be a real small PNG.
	if st, _ := c.do("PUT", "/me/avatar", map[string]string{"png": "data:image/png;base64,AAAA"}, nil); st != 400 {
		t.Errorf("bogus avatar → %d", st)
	}
	// Unauthenticated access.
	anon := e.client()
	if st, p := anon.do("GET", "/me/progress", nil, nil); st != 401 || p["code"] != "unauthorized" {
		t.Errorf("anon progress → %d %v", st, p)
	}
	// Export and delete.
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", 0)}, nil)
	var export map[string]any
	c.must("GET", "/me/export", nil, &export)
	if visits, _ := export["visits"].([]any); len(visits) != 1 {
		t.Errorf("export = %v", export)
	}
	c.must("DELETE", "/me", nil, nil)
	var n int
	_ = e.db.QueryRow(context.Background(), `select count(*) from scans where user_id = $1`, me.ID).Scan(&n)
	if n != 0 {
		t.Errorf("scans survived account deletion: %d", n)
	}
	if st, _ := c.do("GET", "/me", nil, nil); st != 401 {
		t.Errorf("session survived deletion → %d", st)
	}
}

func TestAdminAnalytics(t *testing.T) {
	e := newEnv(t, true)
	c, _ := e.guest("Gracz")
	var res app.ScanResult
	c.must("POST", "/scans", map[string]any{"payload": e.rotating("wm-druk-3d", 0)}, &res)
	c.must("PUT", "/scans/"+res.Scan.ID+"/rating", map[string]int{"rating": 4}, nil)
	var ev app.Event
	c.must("GET", "/events/active", nil, &ev)
	if st, _ := c.do("GET", "/admin/events/"+ev.ID+"/analytics", nil, nil); st != 403 {
		t.Errorf("participant analytics → %d", st)
	}
	admin := e.admin()
	var a app.Analytics
	admin.must("GET", "/admin/events/"+ev.ID+"/analytics", nil, &a)
	if a.Participants != 1 || a.Scans != 1 || a.Rated != 1 {
		t.Errorf("analytics = %+v", a)
	}
	var top int
	for _, x := range a.Archetypes {
		if x.Code == "STR" {
			top = x.Users
		}
	}
	if top != 1 {
		t.Errorf("archetypes = %+v", a.Archetypes)
	}
	// Weight editing normalises to 1.
	var st app.Station
	admin.must("PATCH", "/admin/stations/"+res.Scan.StationID, map[string]any{"weights": map[string]float64{"STR": 3, "MAN": 1}}, &st)
	if st.Weights["STR"] != 0.75 || st.Weights["MAN"] != 0.25 {
		t.Errorf("weights = %v", st.Weights)
	}
}
