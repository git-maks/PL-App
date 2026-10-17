package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/git-maks/pl-app/apps/api/internal/app"
)

// authed wraps handlers that need a logged-in user.
func (s *Server) authed(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := userID(r)
		if id == "" {
			s.problem(w, r, app.ErrUnauthorized)
			return
		}
		h(w, r, id)
	}
}

// withMe additionally loads the user (for role checks).
func (s *Server) withMe(h func(http.ResponseWriter, *http.Request, app.Me)) http.HandlerFunc {
	return s.authed(func(w http.ResponseWriter, r *http.Request, id string) {
		me, err := s.svc.GetMe(r.Context(), id)
		if err != nil {
			s.problem(w, r, err)
			return
		}
		h(w, r, me)
	})
}

// timeout bounds ordinary API handlers (the SSE stream is exempt).
func timeout(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		h(w, r.WithContext(ctx))
	}
}

func (s *Server) routes(mux *http.ServeMux) {
	const p = "/api/v1"
	handle := func(pattern string, h http.HandlerFunc) {
		method, path, _ := cut(pattern)
		s.registered = append(s.registered, method+" "+path)
		mux.HandleFunc(method+" "+p+path, timeout(h))
	}

	handle("GET /healthz", s.health)
	handle("GET /readyz", s.ready)

	// auth
	handle("POST /auth/guest", s.createGuest)
	handle("POST /auth/email/start", s.emailStart)
	handle("POST /auth/email/verify", s.emailVerify)
	handle("POST /auth/logout", s.logout)
	handle("POST /auth/logout-all", s.authed(s.logoutAll))

	// me
	handle("GET /me", s.authed(s.getMe))
	handle("PATCH /me", s.authed(s.patchMe))
	handle("DELETE /me", s.authed(s.deleteMe))
	handle("PUT /me/avatar", s.authed(s.putAvatar))
	handle("DELETE /me/avatar", s.authed(s.deleteAvatar))
	handle("GET /me/export", s.authed(s.export))
	handle("GET /me/progress", s.authed(s.progress))
	handle("GET /me/character", s.authed(s.character))
	handle("GET /me/notifications", s.authed(s.notifications))
	handle("POST /me/notifications/read", s.authed(s.notificationsRead))
	handle("GET /me/reminders", s.authed(s.reminders))
	handle("GET /users/{id}/avatar", s.avatar)

	// events & game
	handle("GET /events/active", s.activeEvent)
	handle("GET /events/{eventId}/stations", s.stations)
	handle("GET /events/{eventId}/leaderboard", s.authed(s.leaderboard))
	handle("POST /scans", s.authed(s.scan))
	handle("PUT /scans/{scanId}/rating", s.authed(s.rate))

	// teams
	handle("POST /teams", s.authed(s.createTeam))
	handle("GET /teams/mine", s.authed(s.myTeam))
	handle("POST /teams/join", s.authed(s.joinTeam))
	handle("POST /teams/{teamId}/invites", s.authed(s.invite))
	handle("POST /teams/{teamId}/leave", s.authed(s.leaveTeam))
	handle("DELETE /teams/{teamId}/members/{userId}", s.authed(s.removeMember))

	// realtime & push (the stream is long-lived: no timeout wrapper)
	s.registered = append(s.registered, "GET /stream")
	mux.HandleFunc("GET "+p+"/stream", s.authed(s.stream))
	handle("GET /push/vapid-public-key", s.vapidKey)
	handle("POST /push/subscriptions", s.authed(s.pushSubscribe))
	handle("DELETE /push/subscriptions", s.authed(s.pushUnsubscribe))
	handle("PUT /sessions/{sessionId}/reminder", s.authed(s.reminderOn))
	handle("DELETE /sessions/{sessionId}/reminder", s.authed(s.reminderOff))

	// staff & admin
	handle("GET /staff/stations", s.withMe(s.staffStations))
	handle("GET /staff/stations/{stationId}/qr", s.withMe(s.stationQR))
	handle("GET /staff/stations/{stationId}/poster", s.withMe(s.stationPoster))
	handle("POST /staff/stations/{stationId}/window", s.withMe(s.scanWindow))
	handle("GET /admin/events/{eventId}/analytics", s.withMe(s.analytics))
	handle("GET /admin/events/{eventId}/analytics.csv", s.withMe(s.analyticsCSV))
	handle("PATCH /admin/stations/{stationId}", s.withMe(s.patchStation))
	handle("POST /admin/users/role", s.withMe(s.setRole))
	handle("POST /admin/broadcast", s.withMe(s.broadcast))

	mux.HandleFunc(p+"/", func(w http.ResponseWriter, r *http.Request) {
		s.problem(w, r, app.ErrNotFound)
	})
}

// Routes lists the registered API routes as "METHOD /path" (without the /api/v1 prefix).
func (s *Server) Routes() []string {
	if s.registered == nil {
		s.routes(http.NewServeMux())
	}
	return s.registered
}

func cut(pattern string) (method, path string, ok bool) {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ' ' {
			return pattern[:i], pattern[i+1:], true
		}
	}
	return "", pattern, false
}

// ---------- system ----------

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": s.cfg.Version})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DB.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database unavailable", "version": s.cfg.Version})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": s.cfg.Version, "sseClients": s.hub.Connections()})
}

// ---------- auth ----------

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, uid string) (app.Me, error) {
	token, exp, err := s.svc.CreateSession(r.Context(), uid, r.UserAgent())
	if err != nil {
		return app.Me{}, err
	}
	s.setSessionCookie(w, token, exp)
	return s.svc.GetMe(r.Context(), uid)
}

func (s *Server) createGuest(w http.ResponseWriter, r *http.Request) {
	if s.limited(w, r, "guest:"+s.clientIP(r), 10, 10) {
		return
	}
	var body struct {
		Nickname string `json:"nickname"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	me, err := s.svc.CreateGuest(r.Context(), body.Nickname)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	me, err = s.startSession(w, r, me.ID)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, me)
}

func (s *Server) emailStart(w http.ResponseWriter, r *http.Request) {
	if s.limited(w, r, "email:"+s.clientIP(r), 3, 3) {
		return
	}
	var body struct {
		Email        string `json:"email"`
		AgeConfirmed bool   `json:"ageConfirmed"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	code, err := s.svc.StartEmailLogin(r.Context(), body.Email, body.AgeConfirmed)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	resp := map[string]any{"sent": true}
	if code != "" {
		resp["devCode"] = code
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) emailVerify(w http.ResponseWriter, r *http.Request) {
	if s.limited(w, r, "verify:"+s.clientIP(r), 10, 10) {
		return
	}
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	uid, err := s.svc.VerifyEmailLogin(r.Context(), userID(r), body.Email, body.Code)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	if c, err := r.Cookie(s.cfg.SessionCookie()); err == nil {
		_ = s.svc.DeleteSession(r.Context(), c.Value)
	}
	me, err := s.startSession(w, r, uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.cfg.SessionCookie()); err == nil {
		_ = s.svc.DeleteSession(r.Context(), c.Value)
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logoutAll(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.DeleteAllSessions(r.Context(), uid); err != nil {
		s.problem(w, r, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- me ----------

func (s *Server) getMe(w http.ResponseWriter, r *http.Request, uid string) {
	me, err := s.svc.GetMe(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

func (s *Server) patchMe(w http.ResponseWriter, r *http.Request, uid string) {
	var p app.MePatch
	if err := decode(r, &p); err != nil {
		s.problem(w, r, err)
		return
	}
	me, err := s.svc.UpdateMe(r.Context(), uid, p)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.DeleteAccount(r.Context(), uid); err != nil {
		s.problem(w, r, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) putAvatar(w http.ResponseWriter, r *http.Request, uid string) {
	var body struct {
		PNG string `json:"png"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	me, err := s.svc.SetAvatar(r.Context(), uid, body.PNG)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

func (s *Server) deleteAvatar(w http.ResponseWriter, r *http.Request, uid string) {
	me, err := s.svc.DeleteAvatar(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, me)
}

func (s *Server) avatar(w http.ResponseWriter, r *http.Request) {
	b, err := s.svc.Avatar(r.Context(), r.PathValue("id"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	// The URL carries ?v=<version>, so a changed avatar gets a new URL.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(b)
}

func (s *Server) export(w http.ResponseWriter, r *http.Request, uid string) {
	data, err := s.svc.Export(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="uniquest-moje-dane.json"`)
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) progress(w http.ResponseWriter, r *http.Request, uid string) {
	ev, err := s.svc.ActiveEvent(r.Context())
	if err != nil {
		s.problem(w, r, err)
		return
	}
	p, err := s.svc.Progress(r.Context(), uid, ev)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) character(w http.ResponseWriter, r *http.Request, uid string) {
	ev, err := s.svc.ActiveEvent(r.Context())
	if err != nil {
		s.problem(w, r, err)
		return
	}
	c, err := s.svc.Character(r.Context(), uid, ev)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request, uid string) {
	n, err := s.svc.Notifications(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) notificationsRead(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.MarkNotificationsRead(r.Context(), uid); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reminders(w http.ResponseWriter, r *http.Request, uid string) {
	ids, err := s.svc.Reminders(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ids)
}

// ---------- events & game ----------

func (s *Server) activeEvent(w http.ResponseWriter, r *http.Request) {
	ev, err := s.svc.ActiveEvent(r.Context())
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) stations(w http.ResponseWriter, r *http.Request) {
	list, err := s.svc.Stations(r.Context(), r.PathValue("eventId"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) leaderboard(w http.ResponseWriter, r *http.Request, uid string) {
	b, err := s.svc.Leaderboard(r.Context(), r.PathValue("eventId"), r.URL.Query().Get("type"), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) scan(w http.ResponseWriter, r *http.Request, uid string) {
	if s.limited(w, r, "scan:"+uid, 20, 10) {
		return
	}
	var req app.ScanRequest
	if err := decode(r, &req); err != nil {
		s.problem(w, r, err)
		return
	}
	res, err := s.svc.Scan(r.Context(), uid, req)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) rate(w http.ResponseWriter, r *http.Request, uid string) {
	var body struct {
		Rating int `json:"rating"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	res, err := s.svc.Rate(r.Context(), uid, r.PathValue("scanId"), body.Rating)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---------- teams ----------

func (s *Server) createTeam(w http.ResponseWriter, r *http.Request, uid string) {
	var body struct {
		Name   string `json:"name"`
		Emblem string `json:"emblem"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	t, err := s.svc.CreateTeam(r.Context(), uid, body.Name, body.Emblem)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) myTeam(w http.ResponseWriter, r *http.Request, uid string) {
	t, err := s.svc.MyTeam(r.Context(), uid)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) joinTeam(w http.ResponseWriter, r *http.Request, uid string) {
	if s.limited(w, r, "join:"+uid, 10, 5) {
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	t, err := s.svc.JoinTeam(r.Context(), uid, body.Code)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) invite(w http.ResponseWriter, r *http.Request, uid string) {
	inv, err := s.svc.CreateInvite(r.Context(), uid, r.PathValue("teamId"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (s *Server) leaveTeam(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.LeaveTeam(r.Context(), uid, r.PathValue("teamId")); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.RemoveMember(r.Context(), uid, r.PathValue("teamId"), r.PathValue("userId")); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- realtime & push ----------

// stream is the per-user SSE channel (spec §7.6).
func (s *Server) stream(w http.ResponseWriter, r *http.Request, uid string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.problem(w, r, fmt.Errorf("streaming unsupported"))
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "retry: 5000\n\n")
	flusher.Flush()

	msgs, unsubscribe := s.hub.Subscribe(uid)
	defer unsubscribe()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.closing:
			return
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case m := <-msgs:
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m.Type, m.Data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) vapidKey(w http.ResponseWriter, r *http.Request) {
	key := ""
	if s.svc.Push != nil && s.svc.Push.Enabled() {
		key = s.svc.Push.PublicKey()
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

func (s *Server) pushSubscribe(w http.ResponseWriter, r *http.Request, uid string) {
	var sub app.PushSubscription
	if err := decode(r, &sub); err != nil {
		s.problem(w, r, err)
		return
	}
	if err := s.svc.SavePushSubscription(r.Context(), uid, sub); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) pushUnsubscribe(w http.ResponseWriter, r *http.Request, uid string) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	if err := s.svc.DeletePushSubscription(r.Context(), uid, body.Endpoint); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reminderOn(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.SetReminder(r.Context(), uid, r.PathValue("sessionId"), true); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reminderOff(w http.ResponseWriter, r *http.Request, uid string) {
	if err := s.svc.SetReminder(r.Context(), uid, r.PathValue("sessionId"), false); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- staff & admin ----------

func (s *Server) staffStations(w http.ResponseWriter, r *http.Request, me app.Me) {
	list, err := s.svc.StaffStations(r.Context(), me)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) stationQR(w http.ResponseWriter, r *http.Request, me app.Me) {
	q, err := s.svc.StationQR(r.Context(), me, r.PathValue("stationId"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) stationPoster(w http.ResponseWriter, r *http.Request, me app.Me) {
	payload, err := s.svc.StaticPoster(r.Context(), me, r.PathValue("stationId"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"payload": payload})
}

func (s *Server) scanWindow(w http.ResponseWriter, r *http.Request, me app.Me) {
	var body struct {
		Open    bool `json:"open"`
		Minutes int  `json:"minutes"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	st, err := s.svc.SetScanWindow(r.Context(), me, r.PathValue("stationId"), body.Open, body.Minutes)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) analytics(w http.ResponseWriter, r *http.Request, me app.Me) {
	a, err := s.svc.Analytics(r.Context(), me, r.PathValue("eventId"))
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) analyticsCSV(w http.ResponseWriter, r *http.Request, me app.Me) {
	if me.Role != "admin" {
		s.problem(w, r, app.ErrForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="uniquest-analityka-`+strconv.FormatInt(time.Now().Unix(), 10)+`.csv"`)
	if err := s.svc.AnalyticsCSV(r.Context(), me, r.PathValue("eventId"), w); err != nil {
		s.log.Error("csv export", "err", err)
	}
}

func (s *Server) patchStation(w http.ResponseWriter, r *http.Request, me app.Me) {
	var p app.StationPatch
	if err := decode(r, &p); err != nil {
		s.problem(w, r, err)
		return
	}
	st, err := s.svc.PatchStation(r.Context(), me, r.PathValue("stationId"), p)
	if err != nil {
		s.problem(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) setRole(w http.ResponseWriter, r *http.Request, me app.Me) {
	if me.Role != "admin" {
		s.problem(w, r, app.ErrForbidden)
		return
	}
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	if err := s.svc.SetRole(r.Context(), me.ID, body.Email, body.Role); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) broadcast(w http.ResponseWriter, r *http.Request, me app.Me) {
	if me.Role != "admin" {
		s.problem(w, r, app.ErrForbidden)
		return
	}
	var body struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := decode(r, &body); err != nil {
		s.problem(w, r, err)
		return
	}
	if err := s.svc.Broadcast(r.Context(), me.ID, body.Title, body.Body); err != nil {
		s.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
