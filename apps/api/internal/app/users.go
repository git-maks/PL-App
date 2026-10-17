package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"math/big"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const userColumns = `id::text, kind, nickname, email::text, role, avatar_version, avatar_frame, consents, created_at`

func (s *Service) scanMe(row pgx.Row) (Me, error) {
	var m Me
	var version int
	var consents []byte
	if err := row.Scan(&m.ID, &m.Kind, &m.Nickname, &m.Email, &m.Role, &version, &m.AvatarFrame, &consents, &m.CreatedAt); err != nil {
		return m, err
	}
	_ = json.Unmarshal(consents, &m.Consents)
	m.AvatarURL = s.avatarURL(m.ID, version)
	return m, nil
}

// GetMe loads a user by id.
func (s *Service) GetMe(ctx context.Context, userID string) (Me, error) {
	m, err := s.scanMe(s.DB.QueryRow(ctx, `select `+userColumns+` from users where id = $1`, userID))
	if isNoRows(err) {
		return m, ErrUnauthorized
	}
	return m, wrap("get me", err)
}

var nickAdjectives = []string{"Dzielny", "Bystry", "Ciekawski", "Zwinny", "Pomysłowy", "Odważny", "Sprytny", "Wytrwały"}
var nickNouns = []string{"Konstruktor", "Odkrywca", "Technomag", "Strateg", "Badacz", "Architekt", "Analityk", "Atleta"}

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

func randomNickname() string {
	return nickAdjectives[randInt(len(nickAdjectives))] + " " + nickNouns[randInt(len(nickNouns))] + " " + string(rune('0'+1+randInt(9))) + string(rune('0'+randInt(10))) + string(rune('0'+randInt(10))) + string(rune('0'+randInt(10)))
}

// A short blocklist; organisers can rename offensive nicknames from the admin side as well.
var blockedWords = []string{"kurw", "chuj", "jeb", "pierdol", "pizd", "cwel", "fuck", "nigg", "hitler"}

// CleanNickname trims, collapses whitespace, strips control characters and validates length/content.
func CleanNickname(in string) (string, error) {
	var b strings.Builder
	space := false
	for _, r := range strings.TrimSpace(in) {
		switch {
		case unicode.IsControl(r) || r == '​':
			continue
		case unicode.IsSpace(r):
			if !space {
				b.WriteRune(' ')
			}
			space = true
		default:
			b.WriteRune(r)
			space = false
		}
	}
	n := b.String()
	if l := utf8.RuneCountInString(n); l < 2 || l > 24 {
		return "", ErrInvalidNick
	}
	low := strings.ToLower(n)
	for _, w := range blockedWords {
		if strings.Contains(low, w) {
			return "", ErrInvalidNick
		}
	}
	return n, nil
}

// CreateGuest creates an anonymous player (spec §7.3) and returns it.
func (s *Service) CreateGuest(ctx context.Context, nickname string) (Me, error) {
	nick := randomNickname()
	if strings.TrimSpace(nickname) != "" {
		n, err := CleanNickname(nickname)
		if err != nil {
			return Me{}, err
		}
		nick = n
	}
	m, err := s.scanMe(s.DB.QueryRow(ctx, `insert into users (kind, nickname) values ('guest', $1) returning `+userColumns, nick))
	return m, wrap("create guest", err)
}

func (s *Service) UpdateMe(ctx context.Context, userID string, p MePatch) (Me, error) {
	if p.Nickname != nil {
		n, err := CleanNickname(*p.Nickname)
		if err != nil {
			return Me{}, err
		}
		if _, err := s.DB.Exec(ctx, `update users set nickname = $2 where id = $1`, userID, n); err != nil {
			return Me{}, wrap("update nickname", err)
		}
	}
	if p.Consents != nil {
		b, _ := json.Marshal(p.Consents)
		if _, err := s.DB.Exec(ctx, `update users set consents = $2 where id = $1`, userID, b); err != nil {
			return Me{}, wrap("update consents", err)
		}
	}
	if p.AvatarFrame != nil {
		if _, err := s.DB.Exec(ctx, `update users set avatar_frame = nullif($2, '') where id = $1`, userID, *p.AvatarFrame); err != nil {
			return Me{}, wrap("update frame", err)
		}
	}
	return s.GetMe(ctx, userID)
}

const maxAvatarBytes = 16 * 1024

// SetAvatar stores the pixel-art avatar after validating it really is a small PNG.
func (s *Service) SetAvatar(ctx context.Context, userID, dataURL string) (Me, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return Me{}, ErrInvalidAvatar
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL[len(prefix):])
	if err != nil || len(raw) > maxAvatarBytes {
		return Me{}, ErrInvalidAvatar
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width > 128 || cfg.Height > 128 || cfg.Width < 8 || cfg.Height < 8 {
		return Me{}, ErrInvalidAvatar
	}
	if _, err := s.DB.Exec(ctx, `update users set avatar_png = $2, avatar_version = avatar_version + 1 where id = $1`, userID, raw); err != nil {
		return Me{}, wrap("set avatar", err)
	}
	return s.GetMe(ctx, userID)
}

func (s *Service) DeleteAvatar(ctx context.Context, userID string) (Me, error) {
	if _, err := s.DB.Exec(ctx, `update users set avatar_png = null, avatar_version = 0 where id = $1`, userID); err != nil {
		return Me{}, wrap("delete avatar", err)
	}
	return s.GetMe(ctx, userID)
}

func (s *Service) Avatar(ctx context.Context, userID string) ([]byte, error) {
	var b []byte
	err := s.DB.QueryRow(ctx, `select avatar_png from users where id = $1 and avatar_png is not null`, userID).Scan(&b)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	return b, wrap("avatar", err)
}

// DeleteAccount removes the user and everything that cascades from it (GDPR art. 17).
func (s *Service) DeleteAccount(ctx context.Context, userID string) error {
	return s.tx(ctx, func(tx pgx.Tx) error {
		// A captain leaving hands the team over first so team-mates keep their team.
		if err := leaveAllTeams(ctx, tx, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `delete from users where id = $1`, userID)
		return err
	})
}

// Export returns all personal data of the user (GDPR art. 20).
func (s *Service) Export(ctx context.Context, userID string) (map[string]any, error) {
	me, err := s.GetMe(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"exportedAt": s.Now().UTC(), "user": me}
	rows, err := s.DB.Query(ctx, `select st.name, sc.scanned_at, sc.rating, sc.team_complete, sc.xp::float8
		from scans sc join stations st on st.id = sc.station_id where sc.user_id = $1 order by sc.scanned_at`, userID)
	if err != nil {
		return nil, wrap("export scans", err)
	}
	type visit struct {
		Station   string    `json:"station"`
		ScannedAt time.Time `json:"scannedAt"`
		Rating    *int      `json:"rating"`
		Team      bool      `json:"teamBonus"`
		XP        float64   `json:"xp"`
	}
	visits, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (visit, error) {
		var v visit
		err := r.Scan(&v.Station, &v.ScannedAt, &v.Rating, &v.Team, &v.XP)
		return v, err
	})
	if err != nil {
		return nil, wrap("export scans", err)
	}
	out["visits"] = visits
	var teams []string
	trows, err := s.DB.Query(ctx, `select t.name from team_members m join teams t on t.id = m.team_id where m.user_id = $1`, userID)
	if err == nil {
		teams, _ = pgx.CollectRows(trows, pgx.RowTo[string])
	}
	out["teams"] = teams
	notes, _ := s.Notifications(ctx, userID)
	out["notifications"] = notes
	var subs int
	_ = s.DB.QueryRow(ctx, `select count(*) from push_subscriptions where user_id = $1`, userID).Scan(&subs)
	out["pushSubscriptions"] = subs
	return out, nil
}

// SetRole grants or revokes staff/admin by verified e-mail.
func (s *Service) SetRole(ctx context.Context, actorID, email, role string) error {
	switch role {
	case "participant", "staff", "admin":
	default:
		return ErrBadRequest
	}
	tag, err := s.DB.Exec(ctx, `update users set role = $2 where email = $1 and email_verified_at is not null`, strings.ToLower(strings.TrimSpace(email)), role)
	if err != nil {
		return wrap("set role", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	s.audit(ctx, actorID, "set_role", email, map[string]any{"role": role})
	return nil
}

func (s *Service) audit(ctx context.Context, actorID, action, target string, meta map[string]any) {
	b, _ := json.Marshal(meta)
	if _, err := s.DB.Exec(ctx, `insert into audit_log (actor_id, action, target, meta) values (nullif($1, '')::uuid, $2, $3, $4)`, actorID, action, target, b); err != nil {
		s.Log.Warn("audit log failed", "err", err)
	}
}

// PurgeInactiveGuests deletes guest accounts inactive for longer than `after` (retention, spec §13).
func (s *Service) PurgeInactiveGuests(ctx context.Context, after time.Duration) (int64, error) {
	tag, err := s.DB.Exec(ctx, `delete from users where kind = 'guest' and last_active_at < $1`, s.Now().Add(-after))
	if err != nil {
		return 0, err
	}
	_, _ = s.DB.Exec(ctx, `delete from sessions where expires_at < now()`)
	_, _ = s.DB.Exec(ctx, `delete from login_codes where expires_at < now() - interval '1 day'`)
	return tag.RowsAffected(), nil
}
