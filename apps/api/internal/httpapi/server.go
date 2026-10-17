// Package httpapi exposes the UniQuest REST API (packages/contracts/openapi.yaml), the SSE stream
// and the embedded web app on a single origin.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/git-maks/pl-app/apps/api/internal/app"
	"github.com/git-maks/pl-app/apps/api/internal/config"
	"github.com/git-maks/pl-app/apps/api/internal/realtime"
)

type Server struct {
	svc    *app.Service
	hub    *realtime.Hub
	cfg    config.Config
	log    *slog.Logger
	static fs.FS // built web app (may be nil when the API runs alone)

	limiters sync.Map // key → *limiterEntry

	registered []string // "METHOD /path" of every API route, for the contract test

	closing   chan struct{} // closed on shutdown so long-lived SSE streams end promptly
	closeOnce sync.Once
}

// Close ends open SSE streams (call before http.Server.Shutdown).
func (s *Server) Close() { s.closeOnce.Do(func() { close(s.closing) }) }

type limiterEntry struct {
	l    *rate.Limiter
	seen time.Time
	mu   sync.Mutex
}

func New(svc *app.Service, hub *realtime.Hub, cfg config.Config, log *slog.Logger, static fs.FS) *Server {
	return &Server{svc: svc, hub: hub, cfg: cfg, log: log, static: static, closing: make(chan struct{})}
}

type ctxKey int

const userKey ctxKey = 1

func userID(r *http.Request) string {
	id, _ := r.Context().Value(userKey).(string)
	return id
}

// ---------- responses ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) problem(w http.ResponseWriter, r *http.Request, err error) {
	var de *app.Error
	if !errors.As(err, &de) {
		if errors.Is(err, context.Canceled) {
			return
		}
		s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		de = &app.Error{Status: http.StatusInternalServerError, Code: "internal", Title: "Błąd serwera"}
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(de.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": de.Title, "status": de.Status, "code": de.Code, "detail": de.Detail})
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return &app.Error{Status: http.StatusBadRequest, Code: "bad_request", Title: "Niepoprawne dane", Detail: err.Error()}
	}
	return nil
}

// ---------- middleware ----------

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allow applies a token bucket per key (e.g. "guest:<ip>").
func (s *Server) allow(key string, perMinute float64, burst int) bool {
	if !s.cfg.RateLimits {
		return true
	}
	v, _ := s.limiters.LoadOrStore(key, &limiterEntry{l: rate.NewLimiter(rate.Limit(perMinute/60), burst)})
	e := v.(*limiterEntry)
	e.mu.Lock()
	e.seen = time.Now()
	e.mu.Unlock()
	return e.l.Allow()
}

// SweepLimiters drops idle limiter entries (run periodically).
func (s *Server) SweepLimiters() {
	cut := time.Now().Add(-15 * time.Minute)
	s.limiters.Range(func(k, v any) bool {
		e := v.(*limiterEntry)
		e.mu.Lock()
		old := e.seen.Before(cut)
		e.mu.Unlock()
		if old {
			s.limiters.Delete(k)
		}
		return true
	})
}

func (s *Server) limited(w http.ResponseWriter, r *http.Request, key string, perMinute float64, burst int) bool {
	if s.allow(key, perMinute, burst) {
		return false
	}
	w.Header().Set("Retry-After", "30")
	s.problem(w, r, app.ErrRateLimited)
	return true
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "err", rec, "path", r.URL.Path)
				s.problem(sw, r, errors.New("panic"))
			}
			if strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasSuffix(r.URL.Path, "/stream") {
				s.log.Info("http", "method", r.Method, "route", r.Pattern, "status", sw.status, "ms", time.Since(start).Milliseconds())
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	csp := strings.Join([]string{
		"default-src 'self'",
		// Inline scripts are additionally restricted by the hash-based CSP SvelteKit emits in each page.
		"script-src 'self' 'wasm-unsafe-eval' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"font-src 'self'",
		"connect-src 'self'",
		"worker-src 'self'",
		"manifest-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; ")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if s.cfg.CookieSecure {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// csrf rejects state-changing API requests without our custom header or from a foreign origin.
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			if r.Header.Get("X-Requested-With") != "uniquest" {
				s.problem(w, r, &app.Error{Status: http.StatusForbidden, Code: "forbidden", Title: "Brak nagłówka X-Requested-With"})
				return
			}
			if o := r.Header.Get("Origin"); o != "" && o != s.cfg.Origin() {
				s.problem(w, r, &app.Error{Status: http.StatusForbidden, Code: "forbidden", Title: "Niedozwolone źródło żądania"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if c, err := r.Cookie(s.cfg.SessionCookie()); err == nil {
				if id, ok := s.svc.SessionUser(r.Context(), c.Value); ok {
					r = r.WithContext(context.WithValue(r.Context(), userKey, id))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cfg.SessionCookie(), Value: token, Path: "/", Expires: exp,
		HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: s.cfg.SessionCookie(), Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
}

// Handler assembles routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	if s.static != nil {
		mux.Handle("/", s.staticHandler())
	}
	return s.logRequests(s.securityHeaders(s.csrf(s.session(mux))))
}
