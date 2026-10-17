package app

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/git-maks/pl-app/apps/api/internal/qr"
)

const inviteTTL = 15 * time.Minute

// Keep in sync with apps/web/src/lib/game/emblems.ts.
var allowedEmblems = map[string]bool{"shield": true, "star": true, "rocket_launch": true, "bolt": true, "science": true, "psychology": true, "memory": true, "smart_toy": true, "code": true, "satellite_alt": true, "lightbulb": true, "eco": true, "forest": true, "public": true, "pets": true, "owl": true, "palette": true, "music_note": true, "sports_soccer": true, "sports_esports": true, "casino": true, "chess_knight": true, "crown": true, "military_tech": true}

func (s *Service) teamDTO(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, teamID string, max int) (Team, error) {
	var t Team
	err := q.QueryRow(ctx, `select id::text, name, emblem, captain_id::text, created_at from teams where id = $1`, teamID).Scan(&t.ID, &t.Name, &t.Emblem, &t.CaptainID, &t.CreatedAt)
	if isNoRows(err) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	rows, err := q.Query(ctx, `select u.id::text, u.nickname, u.avatar_version, m.role, m.joined_at,
			coalesce((select sum(xp) from scans sc where sc.user_id = u.id and sc.event_id = m.event_id), 0)::float8
		from team_members m join users u on u.id = m.user_id where m.team_id = $1 order by m.joined_at`, teamID)
	if err != nil {
		return t, err
	}
	t.Members, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (TeamMember, error) {
		var m TeamMember
		var v int
		err := r.Scan(&m.UserID, &m.Nickname, &v, &m.Role, &m.JoinedAt, &m.XP)
		m.AvatarURL = s.avatarURL(m.UserID, v)
		return m, err
	})
	if err != nil {
		return t, err
	}
	for _, m := range t.Members {
		t.Score += m.XP
	}
	t.MaxMembers = max
	return t, nil
}

func (s *Service) memberIDs(ctx context.Context, teamID string) []string {
	rows, err := s.DB.Query(ctx, `select user_id::text from team_members where team_id = $1`, teamID)
	if err != nil {
		return nil
	}
	ids, _ := pgx.CollectRows(rows, pgx.RowTo[string])
	return ids
}

// MyTeam returns the caller's team in the active event (ErrNotFound when none).
func (s *Service) MyTeam(ctx context.Context, userID string) (Team, error) {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return Team{}, err
	}
	var teamID string
	err = s.DB.QueryRow(ctx, `select team_id::text from team_members where user_id = $1 and event_id = $2`, userID, ev.ID).Scan(&teamID)
	if isNoRows(err) {
		return Team{}, ErrNotFound
	}
	if err != nil {
		return Team{}, wrap("my team", err)
	}
	t, err := s.teamDTO(ctx, s.DB, teamID, ev.TeamMax)
	return t, wrap("my team", err)
}

func (s *Service) CreateTeam(ctx context.Context, userID, name, emblem string) (Team, error) {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return Team{}, err
	}
	name = strings.Join(strings.Fields(name), " ")
	if l := utf8.RuneCountInString(name); l < 2 || l > 24 {
		return Team{}, &Error{Status: 400, Code: "bad_request", Title: "Nazwa drużyny musi mieć 2–24 znaki"}
	}
	if _, err := CleanNickname(name); err != nil {
		return Team{}, &Error{Status: 400, Code: "bad_request", Title: "Ta nazwa drużyny jest niedozwolona"}
	}
	if !allowedEmblems[emblem] {
		emblem = "shield"
	}
	var teamID string
	err = s.tx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from team_members where user_id = $1 and event_id = $2)`, userID, ev.ID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrAlreadyInTeam
		}
		if err := tx.QueryRow(ctx, `insert into teams (event_id, name, emblem, captain_id) values ($1, $2, $3, $4) returning id::text`, ev.ID, name, emblem, userID).Scan(&teamID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `insert into team_members (team_id, user_id, event_id, role) values ($1, $2, $3, 'captain')`, teamID, userID, ev.ID)
		return err
	})
	if err != nil {
		return Team{}, wrap("create team", err)
	}
	t, err := s.teamDTO(ctx, s.DB, teamID, ev.TeamMax)
	return t, wrap("create team", err)
}

func (s *Service) CreateInvite(ctx context.Context, userID, teamID string) (TeamInvite, error) {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return TeamInvite{}, err
	}
	var captain string
	var members int
	err = s.DB.QueryRow(ctx, `select t.captain_id::text, (select count(*) from team_members where team_id = t.id) from teams t where t.id = $1`, teamID).Scan(&captain, &members)
	if isNoRows(err) {
		return TeamInvite{}, ErrNotFound
	}
	if err != nil {
		return TeamInvite{}, wrap("invite", err)
	}
	if captain != userID {
		return TeamInvite{}, ErrNotCaptain
	}
	free := ev.TeamMax - members
	if free <= 0 {
		return TeamInvite{}, ErrTeamFull
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = alphabet[randInt(len(alphabet))]
	}
	code := string(b)
	exp := s.Now().Add(inviteTTL)
	if _, err := s.DB.Exec(ctx, `insert into team_invites (code, team_id, created_by, expires_at, max_uses) values ($1, $2, $3, $4, $5)`, code, teamID, userID, exp, free); err != nil {
		return TeamInvite{}, wrap("invite", err)
	}
	return TeamInvite{Code: code, URL: qr.TeamURL(s.Cfg.BaseURL, code), ExpiresAt: exp, UsesLeft: free}, nil
}

// JoinTeam accepts an invite code or URL.
func (s *Service) JoinTeam(ctx context.Context, userID, codeOrURL string) (Team, error) {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return Team{}, err
	}
	code := strings.ToUpper(strings.TrimSpace(codeOrURL))
	if p := qr.Parse(codeOrURL); p.Kind == qr.Team {
		code = p.Code
	}
	var teamID, nickname string
	err = s.tx(ctx, func(tx pgx.Tx) error {
		var exp time.Time
		var maxUses, uses int
		err := tx.QueryRow(ctx, `select i.team_id::text, i.expires_at, i.max_uses, i.uses from team_invites i join teams t on t.id = i.team_id
			where i.code = $1 and t.event_id = $2 for update of i`, code, ev.ID).Scan(&teamID, &exp, &maxUses, &uses)
		if isNoRows(err) {
			return ErrInviteInvalid
		}
		if err != nil {
			return err
		}
		if s.Now().After(exp) || uses >= maxUses {
			return ErrInviteInvalid
		}
		var inTeam bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from team_members where user_id = $1 and event_id = $2)`, userID, ev.ID).Scan(&inTeam); err != nil {
			return err
		}
		if inTeam {
			return ErrAlreadyInTeam
		}
		// Lock the team row so concurrent joins cannot exceed the limit.
		var members int
		if err := tx.QueryRow(ctx, `select (select count(*) from team_members where team_id = t.id) from teams t where t.id = $1 for update`, teamID).Scan(&members); err != nil {
			return err
		}
		if members >= ev.TeamMax {
			return ErrTeamFull
		}
		if _, err := tx.Exec(ctx, `insert into team_members (team_id, user_id, event_id) values ($1, $2, $3)`, teamID, userID, ev.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `update team_invites set uses = uses + 1 where code = $1`, code); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `select nickname from users where id = $1`, userID).Scan(&nickname)
	})
	if err != nil {
		return Team{}, wrap("join team", err)
	}
	t, err := s.teamDTO(ctx, s.DB, teamID, ev.TeamMax)
	if err != nil {
		return Team{}, wrap("join team", err)
	}
	var member TeamMember
	for _, m := range t.Members {
		if m.UserID == userID {
			member = m
		}
	}
	others := []string{}
	for _, m := range t.Members {
		if m.UserID != userID {
			others = append(others, m.UserID)
			s.Notify(ctx, m.UserID, "team", nickname+" dołącza do drużyny", "Drużyna „"+t.Name+"” ma teraz "+strconv.Itoa(len(t.Members))+" osoby.", "/team")
		}
	}
	s.Pub.Publish(ctx, others, "team.member_joined", map[string]any{"team": t, "member": member})
	return t, nil
}

// leaveAllTeams removes the user from every team, handing captaincy to the longest-standing
// member and deleting teams that become empty.
func leaveAllTeams(ctx context.Context, tx pgx.Tx, userID string) error {
	rows, err := tx.Query(ctx, `select team_id::text from team_members where user_id = $1`, userID)
	if err != nil {
		return err
	}
	teams, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, teamID := range teams {
		if _, err := tx.Exec(ctx, `delete from team_members where team_id = $1 and user_id = $2`, teamID, userID); err != nil {
			return err
		}
		if err := reassignOrDelete(ctx, tx, teamID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) LeaveTeam(ctx context.Context, userID, teamID string) error {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return err
	}
	err = s.tx(ctx, func(tx pgx.Tx) error {
		var member bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from team_members where team_id = $1 and user_id = $2)`, teamID, userID).Scan(&member); err != nil {
			return err
		}
		if !member {
			return nil
		}
		if _, err := tx.Exec(ctx, `delete from team_members where team_id = $1 and user_id = $2`, teamID, userID); err != nil {
			return err
		}
		return reassignOrDelete(ctx, tx, teamID, userID)
	})
	if err != nil {
		return wrap("leave team", err)
	}
	s.broadcastTeam(ctx, ev, teamID, "team.member_left", userID)
	return nil
}

func reassignOrDelete(ctx context.Context, tx pgx.Tx, teamID, leaving string) error {
	var next string
	err := tx.QueryRow(ctx, `select user_id::text from team_members where team_id = $1 order by joined_at limit 1`, teamID).Scan(&next)
	if isNoRows(err) {
		_, err := tx.Exec(ctx, `delete from teams where id = $1`, teamID)
		return err
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update teams set captain_id = $2 where id = $1 and captain_id = $3`, teamID, next, leaving); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `update team_members set role = case when user_id = (select captain_id from teams where id = $1) then 'captain' else 'member' end where team_id = $1`, teamID)
	return err
}

func (s *Service) RemoveMember(ctx context.Context, userID, teamID, memberID string) error {
	ev, err := s.ActiveEvent(ctx)
	if err != nil {
		return err
	}
	var captain string
	err = s.DB.QueryRow(ctx, `select captain_id::text from teams where id = $1`, teamID).Scan(&captain)
	if isNoRows(err) {
		return ErrNotFound
	}
	if err != nil {
		return wrap("remove member", err)
	}
	if captain != userID {
		return ErrNotCaptain
	}
	if memberID == userID {
		return ErrBadRequest
	}
	if _, err := s.DB.Exec(ctx, `delete from team_members where team_id = $1 and user_id = $2`, teamID, memberID); err != nil {
		return wrap("remove member", err)
	}
	s.Pub.Publish(ctx, []string{memberID}, "team.member_left", map[string]any{"team": nil, "userId": memberID})
	s.broadcastTeam(ctx, ev, teamID, "team.updated", "")
	return nil
}

// broadcastTeam sends the current team state to its members.
func (s *Service) broadcastTeam(ctx context.Context, ev Event, teamID, eventType, userID string) {
	t, err := s.teamDTO(ctx, s.DB, teamID, ev.TeamMax)
	if err != nil {
		return
	}
	ids := make([]string, 0, len(t.Members))
	for _, m := range t.Members {
		ids = append(ids, m.UserID)
	}
	if eventType == "team.member_left" {
		s.Pub.Publish(ctx, ids, eventType, map[string]any{"team": t, "userId": userID})
		return
	}
	s.Pub.Publish(ctx, ids, eventType, t)
}

// touchTeamScore pushes the refreshed team score to team-mates after a rating.
func (s *Service) touchTeamScore(ctx context.Context, userID, eventID string) {
	var teamID string
	if err := s.DB.QueryRow(ctx, `select team_id::text from team_members where user_id = $1 and event_id = $2`, userID, eventID).Scan(&teamID); err != nil {
		return
	}
	ev, err := s.eventByID(ctx, eventID)
	if err != nil {
		return
	}
	s.broadcastTeam(ctx, ev, teamID, "team.updated", "")
	s.invalidateBoards(eventID)
}

// checkTeamCompletion completes a team station once enough team-mates scanned it: every
// member's visit gets the team bonus and their points are recomputed (spec §8.7).
func (s *Service) checkTeamCompletion(ctx context.Context, ev Event, teamID string, st Station) {
	if st.Team == nil {
		return
	}
	var completed []struct{ scanID, userID string }
	err := s.tx(ctx, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, `select count(*) from scans where team_id = $1 and station_id = $2`, teamID, st.ID).Scan(&n); err != nil {
			return err
		}
		if n < st.Team.MinMembers {
			return nil
		}
		rows, err := tx.Query(ctx, `update scans set team_complete = true where team_id = $1 and station_id = $2 and not team_complete returning id::text, user_id::text`, teamID, st.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var x struct{ scanID, userID string }
			if err := rows.Scan(&x.scanID, &x.userID); err != nil {
				rows.Close()
				return err
			}
			completed = append(completed, x)
		}
		rows.Close()
		for _, x := range completed {
			if err := s.recomputeDeltas(ctx, tx, x.scanID, ev.Config); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.Log.Warn("team completion", "err", err)
		return
	}
	for _, x := range completed {
		s.Notify(ctx, x.userID, "team", "Zadanie drużynowe zaliczone!", st.Name+" – dostajecie bonus ×1,2.", "/hero")
		if sc, err := s.scanByID(ctx, x.userID, x.scanID); err == nil {
			s.Pub.Publish(ctx, []string{x.userID}, "scan.confirmed", sc)
		}
		if p, err := s.Progress(ctx, x.userID, ev); err == nil {
			s.Pub.Publish(ctx, []string{x.userID}, "progress.updated", p)
		}
		s.awardBadges(ctx, x.userID, ev)
	}
}

// ---------- leaderboard (cached a few seconds; spec §7.5) ----------

func (s *Service) invalidateBoards(eventID string) {
	s.boardMu.Lock()
	defer s.boardMu.Unlock()
	delete(s.boardCache, eventID+"|team")
	delete(s.boardCache, eventID+"|solo")
}

func (s *Service) Leaderboard(ctx context.Context, eventID, kind, userID string) (Leaderboard, error) {
	if kind != "solo" {
		kind = "team"
	}
	key := eventID + "|" + kind
	s.boardMu.Lock()
	c, ok := s.boardCache[key]
	s.boardMu.Unlock()
	if !ok || s.Now().Sub(c.at) > 5*time.Second {
		b, err := s.computeBoard(ctx, eventID, kind)
		if err != nil {
			return Leaderboard{}, err
		}
		c = cachedBoard{at: s.Now(), board: b}
		s.boardMu.Lock()
		s.boardCache[key] = c
		s.boardMu.Unlock()
	}
	out := Leaderboard{Type: kind, Entries: make([]LeaderboardEntry, len(c.board.Entries))}
	copy(out.Entries, c.board.Entries)
	var myTeam string
	if kind == "team" {
		_ = s.DB.QueryRow(ctx, `select team_id::text from team_members where user_id = $1 and event_id = $2`, userID, eventID).Scan(&myTeam)
	}
	for i := range out.Entries {
		out.Entries[i].IsMe = out.Entries[i].ID == userID || (myTeam != "" && out.Entries[i].ID == myTeam)
	}
	return out, nil
}

func (s *Service) computeBoard(ctx context.Context, eventID, kind string) (Leaderboard, error) {
	var q string
	if kind == "team" {
		q = `select t.id::text, t.name, coalesce(sum(sc.xp), 0)::float8, (select count(*) from team_members m2 where m2.team_id = t.id)::int
			from teams t join team_members m on m.team_id = t.id left join scans sc on sc.user_id = m.user_id and sc.event_id = t.event_id
			where t.event_id = $1 group by t.id order by 3 desc, t.created_at limit 50`
	} else {
		q = `select u.id::text, u.nickname, sum(sc.xp)::float8, 0 from scans sc join users u on u.id = sc.user_id
			where sc.event_id = $1 group by u.id having sum(sc.xp) > 0 order by 3 desc, min(sc.scanned_at) limit 50`
	}
	rows, err := s.DB.Query(ctx, q, eventID)
	if err != nil {
		return Leaderboard{}, wrap("leaderboard", err)
	}
	entries, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (LeaderboardEntry, error) {
		var e LeaderboardEntry
		var members int
		err := r.Scan(&e.ID, &e.Name, &e.Score, &members)
		if kind == "team" {
			e.Members = &members
		}
		return e, err
	})
	if err != nil {
		return Leaderboard{}, wrap("leaderboard", err)
	}
	sort.SliceStable(entries, func(a, b int) bool { return entries[a].Score > entries[b].Score })
	for i := range entries {
		entries[i].Rank = i + 1
	}
	if entries == nil {
		entries = []LeaderboardEntry{}
	}
	return Leaderboard{Type: kind, Entries: entries}, nil
}
