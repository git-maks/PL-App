package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------- sessions (server-side, spec §7.3) ----------

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// CreateSession issues a random 256-bit token; only its SHA-256 is stored.
func (s *Service) CreateSession(ctx context.Context, userID, userAgent string) (string, time.Time, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	exp := s.Now().Add(s.Cfg.SessionTTL)
	if len(userAgent) > 200 {
		userAgent = userAgent[:200]
	}
	_, err := s.DB.Exec(ctx, `insert into sessions (token_hash, user_id, expires_at, user_agent) values ($1, $2, $3, $4)`, hashToken(token), userID, exp, userAgent)
	return token, exp, wrap("create session", err)
}

// SessionUser resolves a session token to a user id, sliding the expiry (written at most hourly).
func (s *Service) SessionUser(ctx context.Context, token string) (string, bool) {
	if token == "" || len(token) > 100 {
		return "", false
	}
	var userID string
	var lastSeen, exp time.Time
	err := s.DB.QueryRow(ctx, `select user_id::text, last_seen_at, expires_at from sessions where token_hash = $1`, hashToken(token)).Scan(&userID, &lastSeen, &exp)
	if err != nil || s.Now().After(exp) {
		return "", false
	}
	if s.Now().Sub(lastSeen) > time.Hour {
		_, _ = s.DB.Exec(ctx, `update sessions set last_seen_at = now(), expires_at = $2 where token_hash = $1`, hashToken(token), s.Now().Add(s.Cfg.SessionTTL))
		_, _ = s.DB.Exec(ctx, `update users set last_active_at = now() where id = $1`, userID)
	}
	return userID, true
}

func (s *Service) DeleteSession(ctx context.Context, token string) error {
	_, err := s.DB.Exec(ctx, `delete from sessions where token_hash = $1`, hashToken(token))
	return wrap("logout", err)
}

func (s *Service) DeleteAllSessions(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(ctx, `delete from sessions where user_id = $1`, userID)
	return wrap("logout all", err)
}

// ---------- e-mail login codes ----------

const (
	codeTTL         = 10 * time.Minute
	codeMaxAttempts = 5
	codesPerHour    = 10
)

func normalizeEmail(in string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(in))
	if len(e) > 254 {
		return "", ErrInvalidEmail
	}
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e || !strings.Contains(e[strings.LastIndex(e, "@"):], ".") {
		return "", ErrInvalidEmail
	}
	return e, nil
}

func codeHash(email, code string) []byte {
	h := sha256.Sum256([]byte(email + "|" + code))
	return h[:]
}

// StartEmailLogin sends a 6-digit code. Returns the code itself only when DevLoginCodes is on.
func (s *Service) StartEmailLogin(ctx context.Context, rawEmail string, ageConfirmed bool) (string, error) {
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return "", err
	}
	if !ageConfirmed {
		return "", &Error{Status: 400, Code: "bad_request", Title: "Wymagane potwierdzenie wieku lub zgody opiekuna"}
	}
	var recent int
	if err := s.DB.QueryRow(ctx, `select count(*) from login_codes where email = $1 and created_at > now() - interval '1 hour'`, email).Scan(&recent); err != nil {
		return "", wrap("count codes", err)
	}
	if recent >= codesPerHour {
		return "", ErrRateLimited
	}
	code := fmt.Sprintf("%06d", randInt(1_000_000))
	if _, err := s.DB.Exec(ctx, `insert into login_codes (email, code_hash, expires_at) values ($1, $2, $3)`, email, codeHash(email, code), s.Now().Add(codeTTL)); err != nil {
		return "", wrap("store code", err)
	}
	if s.Cfg.DevLoginCodes {
		s.Log.Info("dev login code", "email_domain", email[strings.LastIndex(email, "@"):], "code", code)
		return code, nil
	}
	body := "Twój kod logowania do UniQuest: " + code + "\n\nKod jest ważny 10 minut. Jeśli to nie Ty, zignoruj tę wiadomość.\n\nUniQuest · Politechnika Lubelska"
	if err := s.Mail.Send(ctx, email, "Kod logowania UniQuest: "+code, body); err != nil {
		return "", wrap("send code", err)
	}
	return "", nil
}

// VerifyEmailLogin checks the code and returns the user to start a session for:
//   - e-mail unknown + current guest → the guest is upgraded (progress kept);
//   - e-mail unknown + no session    → a new registered user is created;
//   - e-mail known                   → that account; a current guest's progress is merged into it.
func (s *Service) VerifyEmailLogin(ctx context.Context, currentUserID, rawEmail, code string) (string, error) {
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return "", err
	}
	code = strings.TrimSpace(code)
	var userID string
	// Failed checks must still commit the attempt counter, so they are reported via checkErr.
	var checkErr error
	err = s.tx(ctx, func(tx pgx.Tx) error {
		var id int64
		var hash []byte
		var exp time.Time
		var attempts int
		err := tx.QueryRow(ctx, `select id, code_hash, expires_at, attempts from login_codes
			where email = $1 and consumed_at is null order by created_at desc limit 1 for update`, email).Scan(&id, &hash, &exp, &attempts)
		if isNoRows(err) {
			checkErr = ErrInvalidCode
			return nil
		}
		if err != nil {
			return err
		}
		if s.Now().After(exp) {
			checkErr = ErrCodeExpired
			return nil
		}
		if attempts >= codeMaxAttempts {
			checkErr = ErrRateLimited
			return nil
		}
		if subtle.ConstantTimeCompare(hash, codeHash(email, code)) != 1 {
			checkErr = ErrInvalidCode
			_, err := tx.Exec(ctx, `update login_codes set attempts = attempts + 1 where id = $1`, id)
			return err
		}
		if _, err := tx.Exec(ctx, `update login_codes set consumed_at = now() where id = $1`, id); err != nil {
			return err
		}

		var existing string
		err = tx.QueryRow(ctx, `select id::text from users where email = $1`, email).Scan(&existing)
		switch {
		case err == nil:
			userID = existing
			if currentUserID != "" && currentUserID != existing {
				return mergeGuest(ctx, tx, currentUserID, existing)
			}
			return nil
		case !isNoRows(err):
			return err
		}
		if currentUserID != "" {
			userID = currentUserID
			_, err := tx.Exec(ctx, `update users set kind = 'registered', email = $2, email_verified_at = now() where id = $1`, currentUserID, email)
			return err
		}
		return tx.QueryRow(ctx, `insert into users (kind, nickname, email, email_verified_at, consents)
			values ('registered', $1, $2, now(), '{"terms": true, "notifications": false}') returning id::text`, randomNickname(), email).Scan(&userID)
	})
	if err != nil {
		return "", wrap("verify code", err)
	}
	if checkErr != nil {
		return "", checkErr
	}
	return userID, nil
}

// mergeGuest moves a guest's visits (for stations the target has not visited) and team into the
// target account, then deletes the guest.
func mergeGuest(ctx context.Context, tx pgx.Tx, guestID, targetID string) error {
	var kind string
	if err := tx.QueryRow(ctx, `select kind from users where id = $1`, guestID).Scan(&kind); err != nil {
		return err
	}
	if kind != "guest" {
		return nil // never merge two registered accounts
	}
	if _, err := tx.Exec(ctx, `update scans set user_id = $2 where user_id = $1
		and station_id not in (select station_id from scans where user_id = $2)`, guestID, targetID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update team_members set user_id = $2 where user_id = $1
		and event_id not in (select event_id from team_members where user_id = $2)`, guestID, targetID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update teams set captain_id = $2 where captain_id = $1
		and id in (select team_id from team_members where user_id = $2)`, guestID, targetID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update badges_awarded set user_id = $2 where user_id = $1
		and (event_id, badge) not in (select event_id, badge from badges_awarded where user_id = $2)`, guestID, targetID); err != nil {
		return err
	}
	if err := leaveAllTeams(ctx, tx, guestID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `delete from users where id = $1`, guestID)
	return err
}
