// Package app holds UniQuest's use cases (domain logic + persistence). HTTP handlers in
// internal/httpapi are thin adapters over this package; DTO shapes follow
// packages/contracts/openapi.yaml.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/git-maks/pl-app/apps/api/internal/catalog"
	"github.com/git-maks/pl-app/apps/api/internal/config"
)

// Error is a domain error rendered as application/problem+json.
type Error struct {
	Status int
	Code   string
	Title  string
	Detail string
}

func (e *Error) Error() string { return e.Code + ": " + e.Title }

func newErr(status int, code, title string) *Error {
	return &Error{Status: status, Code: code, Title: title}
}

var (
	ErrUnauthorized  = newErr(http.StatusUnauthorized, "unauthorized", "Brak sesji")
	ErrForbidden     = newErr(http.StatusForbidden, "forbidden", "Brak uprawnień")
	ErrNotFound      = newErr(http.StatusNotFound, "not_found", "Nie znaleziono")
	ErrNoActiveEvent = newErr(http.StatusNotFound, "no_active_event", "Brak aktywnego wydarzenia")
	ErrQRInvalid     = newErr(http.StatusBadRequest, "qr_invalid", "To nie jest kod stacji UniQuest")
	ErrQRExpired     = newErr(http.StatusUnprocessableEntity, "qr_expired", "Kod wygasł")
	ErrWindowClosed  = newErr(http.StatusUnprocessableEntity, "scan_window_closed", "Skanowanie na tej stacji jest zamknięte")
	ErrRatingLocked  = newErr(http.StatusConflict, "rating_locked", "Ocenę można zmienić tylko przez 15 minut")
	ErrInvalidRating = newErr(http.StatusBadRequest, "invalid_rating", "Ocena musi być w skali 1–5")
	ErrTeamFull      = newErr(http.StatusConflict, "team_full", "Drużyna jest pełna")
	ErrAlreadyInTeam = newErr(http.StatusConflict, "already_in_team", "Należysz już do drużyny")
	ErrInviteInvalid = newErr(http.StatusNotFound, "invite_invalid", "Zaproszenie jest nieważne")
	ErrNotCaptain    = newErr(http.StatusForbidden, "not_captain", "Tylko kapitan może to zrobić")
	ErrInvalidEmail  = newErr(http.StatusBadRequest, "invalid_email", "Niepoprawny adres e-mail")
	ErrInvalidCode   = newErr(http.StatusBadRequest, "invalid_code", "Kod jest niepoprawny")
	ErrCodeExpired   = newErr(http.StatusBadRequest, "code_expired", "Kod wygasł")
	ErrInvalidNick   = newErr(http.StatusBadRequest, "invalid_nickname", "Pseudonim musi mieć 2–24 znaki")
	ErrInvalidAvatar = newErr(http.StatusBadRequest, "invalid_avatar", "Niepoprawny awatar")
	ErrRateLimited   = newErr(http.StatusTooManyRequests, "rate_limited", "Za dużo prób")
	ErrBadRequest    = newErr(http.StatusBadRequest, "bad_request", "Niepoprawne żądanie")
)

// Publisher delivers server-sent events to users (implemented by internal/realtime).
type Publisher interface {
	Publish(ctx context.Context, userIDs []string, eventType string, data any)
}

// Pusher sends Web Push notifications (implemented by internal/push; may be disabled).
type Pusher interface {
	Enabled() bool
	PublicKey() string
	Send(ctx context.Context, userID string, msg PushMessage)
}

type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// Mailer sends transactional e-mail (login codes).
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

type Service struct {
	DB      *pgxpool.Pool
	Cfg     config.Config
	Catalog *catalog.Catalog
	Pub     Publisher
	Push    Pusher
	Mail    Mailer
	Log     *slog.Logger
	Now     func() time.Time

	boardMu    sync.Mutex
	boardCache map[string]cachedBoard
}

type cachedBoard struct {
	at    time.Time
	board Leaderboard
}

func New(db *pgxpool.Pool, cfg config.Config, cat *catalog.Catalog, pub Publisher, push Pusher, mail Mailer, log *slog.Logger) *Service {
	return &Service{DB: db, Cfg: cfg, Catalog: cat, Pub: pub, Push: push, Mail: mail, Log: log, Now: time.Now, boardCache: map[string]cachedBoard{}}
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// tx runs fn in a transaction.
func (s *Service) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.DB, fn)
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	var de *Error
	if errors.As(err, &de) {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}

func (s *Service) avatarURL(userID string, version int) *string {
	if version == 0 {
		return nil
	}
	u := fmt.Sprintf("%s/api/v1/users/%s/avatar?v=%d", s.Cfg.BasePath(), userID, version)
	return &u
}
