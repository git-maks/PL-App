-- UniQuest schema (spec §11). Applied by `uniquest migrate` (embedded, ordered, transactional).
create extension if not exists citext;

create table users (
	id uuid primary key default gen_random_uuid(),
	kind text not null check (kind in ('guest', 'registered')),
	nickname text not null,
	email citext unique,
	email_verified_at timestamptz,
	role text not null default 'participant' check (role in ('participant', 'staff', 'admin')),
	avatar_png bytea,
	avatar_version int not null default 0,
	avatar_frame text,
	consents jsonb not null default '{"terms": false, "notifications": false}',
	created_at timestamptz not null default now(),
	last_active_at timestamptz not null default now()
);
create index users_last_active_idx on users (last_active_at) where kind = 'guest';

create table sessions (
	token_hash bytea primary key,
	user_id uuid not null references users on delete cascade,
	created_at timestamptz not null default now(),
	last_seen_at timestamptz not null default now(),
	expires_at timestamptz not null,
	user_agent text not null default ''
);
create index sessions_user_idx on sessions (user_id);

create table login_codes (
	id bigserial primary key,
	email citext not null,
	code_hash bytea not null,
	created_at timestamptz not null default now(),
	expires_at timestamptz not null,
	attempts int not null default 0,
	consumed_at timestamptz
);
create index login_codes_email_idx on login_codes (email, created_at desc);

create table events (
	id uuid primary key default gen_random_uuid(),
	slug text not null unique,
	name text not null,
	description text not null default '',
	timezone text not null default 'Europe/Warsaw',
	starts_at timestamptz not null,
	ends_at timestamptz not null,
	status text not null default 'draft' check (status in ('draft', 'active', 'finished')),
	config jsonb not null,
	team_min int not null default 2,
	team_max int not null default 5,
	queued_grace_seconds int not null default 600
);

create table stations (
	id uuid primary key default gen_random_uuid(),
	event_id uuid not null references events on delete cascade,
	slug text not null,
	name text not null,
	description text not null default '',
	kind text not null check (kind in ('workshop', 'presentation', 'show', 'place')),
	faculty text,
	poi text not null,
	base_points int not null default 100 check (base_points between 1 and 1000),
	weights jsonb not null,
	team_min_members int,
	qr_secret bytea not null,
	-- manual override of the schedule-based scan window
	manual_state text check (manual_state in ('open', 'closed')),
	manual_until timestamptz,
	unique (event_id, slug)
);

create table station_staff (
	station_id uuid not null references stations on delete cascade,
	user_id uuid not null references users on delete cascade,
	primary key (station_id, user_id)
);

create table station_sessions (
	id uuid primary key default gen_random_uuid(),
	station_id uuid not null references stations on delete cascade,
	room text not null default '',
	starts_at timestamptz not null,
	ends_at timestamptz not null,
	scan_opens_at timestamptz not null,
	scan_closes_at timestamptz not null
);
create index station_sessions_station_idx on station_sessions (station_id, starts_at);

create table teams (
	id uuid primary key default gen_random_uuid(),
	event_id uuid not null references events on delete cascade,
	name text not null,
	emblem text not null default 'shield',
	captain_id uuid not null references users on delete cascade,
	created_at timestamptz not null default now()
);

create table team_members (
	team_id uuid not null references teams on delete cascade,
	user_id uuid not null references users on delete cascade,
	event_id uuid not null references events on delete cascade,
	role text not null default 'member' check (role in ('captain', 'member')),
	joined_at timestamptz not null default now(),
	primary key (team_id, user_id),
	unique (event_id, user_id)
);

create table team_invites (
	code text primary key,
	team_id uuid not null references teams on delete cascade,
	created_by uuid not null references users on delete cascade,
	expires_at timestamptz not null,
	max_uses int not null,
	uses int not null default 0
);

create table scans (
	id uuid primary key default gen_random_uuid(),
	event_id uuid not null references events on delete cascade,
	user_id uuid not null references users on delete cascade,
	station_id uuid not null references stations on delete cascade,
	scanned_at timestamptz not null,
	received_at timestamptz not null default now(),
	queued boolean not null default false,
	trust text not null check (trust in ('rotating', 'static', 'manual')),
	rating smallint check (rating between 1 and 5),
	rated_at timestamptz,
	team_id uuid references teams on delete set null,
	team_complete boolean not null default false,
	-- points frozen at rating time so later weight changes do not rewrite history
	deltas jsonb not null default '{}',
	xp numeric(10, 2) not null default 0,
	unique (user_id, station_id)
);
create index scans_station_idx on scans (station_id);
create index scans_event_user_idx on scans (event_id, user_id);
create index scans_team_station_idx on scans (team_id, station_id) where team_id is not null;

create table badges_awarded (
	user_id uuid not null references users on delete cascade,
	event_id uuid not null references events on delete cascade,
	badge text not null,
	awarded_at timestamptz not null default now(),
	primary key (user_id, event_id, badge)
);

create table notifications (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references users on delete cascade,
	kind text not null check (kind in ('reminder', 'team', 'card', 'broadcast', 'badge')),
	title text not null,
	body text not null,
	url text,
	created_at timestamptz not null default now(),
	read_at timestamptz
);
create index notifications_user_idx on notifications (user_id, created_at desc);

create table push_subscriptions (
	id bigserial primary key,
	user_id uuid not null references users on delete cascade,
	endpoint text not null unique,
	p256dh text not null,
	auth text not null,
	created_at timestamptz not null default now(),
	last_ok_at timestamptz
);
create index push_subscriptions_user_idx on push_subscriptions (user_id);

create table session_reminders (
	user_id uuid not null references users on delete cascade,
	session_id uuid not null references station_sessions on delete cascade,
	notify_at timestamptz not null,
	sent_at timestamptz,
	primary key (user_id, session_id)
);
create index session_reminders_due_idx on session_reminders (notify_at) where sent_at is null;

create table audit_log (
	id bigserial primary key,
	actor_id uuid,
	action text not null,
	target text not null default '',
	meta jsonb not null default '{}',
	at timestamptz not null default now()
);
