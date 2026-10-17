/*
 * DemoApi — the whole backend simulated in the browser (IndexedDB), used for the public
 * GitHub Pages demo. It follows the same contract and rules as the Go backend, including real
 * HMAC validation of rotating QR codes, so the demo exercises the genuine scanning flow.
 * Simulated extras: bot team-mates who join and scan, and synthetic leaderboard rivals.
 */
import { buildProfile, recommend, scanDeltas } from '#lib/game/scoring';
import { clubs, programs, faculties } from '#lib/content';
import { openKV } from '#lib/offline/idb';
import {
	MANUAL_PERIOD_SECONDS,
	PERIOD_SECONDS,
	macFor,
	manualCodeFor,
	parsePayload,
	rotatingMessage,
	rotatingUrl,
	safeEqual,
	staticMessage,
	teamUrl,
	windowAt
} from '#lib/qr/payload';
import {
	ApiError,
	type ApiClient,
	type Badge,
	type Character,
	type Me,
	type MePatch,
	type Notification,
	type Progress,
	type Scan,
	type ScanRequest,
	type ScanResult,
	type StaffStation,
	type Station,
	type StreamEvent,
	type Team,
	type TeamMember
} from '../types';
import { demoWorld, stationSecret, uuidFrom } from './model';

interface DemoScan {
	id: string;
	stationId: string;
	scannedAt: string;
	rating: number | null;
	ratedAt: string | null;
	teamId: string | null;
	teamScanned: number;
	teamComplete: boolean;
}
interface DemoState {
	me: (Me & { avatarData?: string | null }) | null;
	scans: DemoScan[];
	team: Team | null;
	notifications: Notification[];
	reminders: string[];
	badges: Badge[];
	closedStations: string[];
	pendingEmail: { email: string; code: string; expires: number } | null;
}

const RATING_LOCK_MS = 15 * 60_000;
const QUEUED_GRACE_WINDOWS = Math.ceil((10 * 60) / PERIOD_SECONDS);
const BOT_NAMES = ['Ola', 'Kuba', 'Zuzia', 'Antek'];
const RIVAL_TEAMS = ['Mechaniczne Wilki', 'Bity i Bajty', 'Zielona Energia', 'Architekci Chaosu', 'Startup Squad', 'Drużyna Rdzewiaka'];
const RIVAL_PLAYERS = ['Kasia', 'Michał', 'Natalia', 'Bartek', 'Ania', 'Paweł', 'Julia', 'Tomek'];

const empty = (): DemoState => ({
	me: null,
	scans: [],
	team: null,
	notifications: [],
	reminders: [],
	badges: [],
	closedStations: [],
	pendingEmail: null
});

function nowIso() {
	return new Date().toISOString();
}
function uuid() {
	return crypto.randomUUID();
}
function randomCode(n: number) {
	const a = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';
	const b = crypto.getRandomValues(new Uint8Array(n));
	return [...b].map((x) => a[x % a.length]).join('');
}
function hashNum(s: string) {
	let h = 2166136261;
	for (const ch of s) h = Math.imul(h ^ ch.charCodeAt(0), 16777619);
	return h >>> 0;
}

export class DemoApi implements ApiClient {
	readonly mode = 'demo' as const;
	private kv = openKV('uniquest-demo');
	private state: DemoState | null = null;
	private listeners = new Set<(e: StreamEvent) => void>();
	private timers: ReturnType<typeof setTimeout>[] = [];

	constructor(private readonly origin: string) {}

	// ---------- persistence & events ----------
	private async load(): Promise<DemoState> {
		if (!this.state) this.state = { ...empty(), ...((await this.kv.get<DemoState>('state')) ?? {}) };
		return this.state;
	}
	private async save() {
		if (this.state) await this.kv.set('state', plain(this.state));
	}
	private emit(e: StreamEvent) {
		for (const l of this.listeners) l(e);
	}
	private later(ms: number, fn: () => void | Promise<void>) {
		this.timers.push(setTimeout(() => void fn(), ms));
	}
	private async requireMe(): Promise<Me & { avatarData?: string | null }> {
		const s = await this.load();
		if (!s.me) throw new ApiError(401, 'unauthorized', 'Brak sesji');
		return s.me;
	}
	private publicMe(me: DemoState['me']): Me {
		if (!me) throw new ApiError(401, 'unauthorized', 'Brak sesji');
		const { avatarData, ...rest } = me;
		return { ...rest, avatarUrl: avatarData ?? null };
	}
	private async notify(kind: Notification['kind'], title: string, body: string, url: string | null = null) {
		const s = await this.load();
		const n: Notification = { id: uuid(), kind, title, body, url, createdAt: nowIso(), readAt: null };
		s.notifications = [n, ...s.notifications].slice(0, 50);
		await this.save();
		this.emit({ type: 'notification', data: n });
	}

	// ---------- auth & profile ----------
	async me() {
		const s = await this.load();
		return s.me ? this.publicMe(s.me) : null;
	}
	async createGuest(nickname?: string) {
		const s = await this.load();
		const adjectives = ['Dzielny', 'Bystry', 'Ciekawski', 'Zwinny', 'Pomysłowy', 'Odważny'];
		const nouns = ['Konstruktor', 'Odkrywca', 'Technomag', 'Strateg', 'Badacz', 'Architekt'];
		const r = crypto.getRandomValues(new Uint16Array(3));
		s.me = {
			id: uuid(),
			kind: 'guest',
			nickname: nickname?.trim() || `${adjectives[r[0]! % 6]} ${nouns[r[1]! % 6]} ${1000 + (r[2]! % 9000)}`,
			email: null,
			role: 'participant',
			avatarUrl: null,
			avatarFrame: null,
			avatarData: null,
			consents: { terms: false, notifications: false },
			createdAt: nowIso()
		};
		await this.save();
		return this.publicMe(s.me);
	}
	async emailStart(email: string) {
		const s = await this.load();
		if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) throw new ApiError(400, 'invalid_email', 'Niepoprawny e-mail');
		const code = String(100000 + (crypto.getRandomValues(new Uint32Array(1))[0]! % 900000));
		s.pendingEmail = { email: email.toLowerCase(), code, expires: Date.now() + 10 * 60_000 };
		await this.save();
		return { sent: true, devCode: code };
	}
	async emailVerify(email: string, code: string) {
		const s = await this.load();
		const me = await this.requireMe();
		const p = s.pendingEmail;
		if (!p || p.email !== email.toLowerCase()) throw new ApiError(400, 'invalid_code', 'Kod nie pasuje');
		if (Date.now() > p.expires) throw new ApiError(400, 'code_expired', 'Kod wygasł');
		if (p.code !== code) throw new ApiError(400, 'invalid_code', 'Kod nie pasuje');
		me.kind = 'registered';
		me.email = p.email;
		s.pendingEmail = null;
		await this.save();
		return this.publicMe(me);
	}
	async logout() {
		this.timers.forEach(clearTimeout);
		this.state = empty();
		await this.kv.clear();
	}
	async updateMe(patch: MePatch) {
		const me = await this.requireMe();
		if (patch.nickname !== undefined) {
			const n = patch.nickname.trim();
			if (n.length < 2 || n.length > 24) throw new ApiError(400, 'invalid_nickname', 'Pseudonim musi mieć 2–24 znaki');
			me.nickname = n;
		}
		if (patch.consents) me.consents = { ...me.consents, ...patch.consents };
		if (patch.avatarFrame !== undefined) me.avatarFrame = patch.avatarFrame;
		await this.save();
		return this.publicMe(me);
	}
	async uploadAvatar(png: string) {
		const me = await this.requireMe();
		if (!png.startsWith('data:image/png;base64,') || png.length > 24_000) throw new ApiError(400, 'invalid_avatar', 'Niepoprawny awatar');
		me.avatarData = png;
		await this.save();
		return this.publicMe(me);
	}
	async deleteAvatar() {
		const me = await this.requireMe();
		me.avatarData = null;
		await this.save();
		return this.publicMe(me);
	}
	async deleteAccount() {
		await this.logout();
	}
	async exportData() {
		const s = await this.load();
		return { exportedAt: nowIso(), mode: 'demo', ...s };
	}

	// ---------- events ----------
	async activeEvent() {
		return (await demoWorld()).event;
	}
	async stations() {
		return (await demoWorld()).stations;
	}

	// ---------- game ----------
	private async scanDto(s: DemoScan): Promise<Scan> {
		const w = await demoWorld();
		const st = w.stationById.get(s.stationId)!;
		const team = s.teamComplete;
		return {
			id: s.id,
			stationId: s.stationId,
			stationName: st.name,
			scannedAt: s.scannedAt,
			rating: s.rating,
			team,
			teamStatus: s.teamId ? (s.teamComplete ? 'complete' : 'pending') : 'none',
			teamProgress: s.teamId && st.team ? { scanned: s.teamScanned, required: st.team.minMembers } : null,
			deltas: scanDeltas({ basePoints: st.basePoints, weights: st.weights, rating: s.rating, team }, w.event.config),
			rateableUntil: new Date(new Date(s.scannedAt).getTime() + RATING_LOCK_MS).toISOString()
		};
	}

	async progress(): Promise<Progress> {
		await this.requireMe();
		const s = await this.load();
		const w = await demoWorld();
		const inputs = s.scans.map((x) => {
			const st = w.stationById.get(x.stationId)!;
			return { basePoints: st.basePoints, weights: st.weights, rating: x.rating, team: x.teamComplete };
		});
		const p = buildProfile(inputs, w.event.config);
		return {
			eventId: w.event.id,
			stats: p.stats,
			xp: p.xp,
			level: p.level,
			levelProgress: p.levelProgress,
			ratedCount: p.ratedCount,
			archetypes: p.archetypes,
			titleKey: p.titleKey,
			cardUnlocked: p.ratedCount >= w.event.config.minScansForCard,
			minScansForCard: w.event.config.minScansForCard,
			scans: await Promise.all([...s.scans].reverse().map((x) => this.scanDto(x))),
			badges: s.badges
		};
	}

	async character(): Promise<Character> {
		const progress = await this.progress();
		return {
			unlocked: progress.cardUnlocked,
			progress,
			programs: recommend(progress.stats, programs, 3),
			clubs: recommend(progress.stats, clubs, 3)
		};
	}

	private async resolveStation(raw: string, queued: boolean): Promise<Station> {
		const w = await demoWorld();
		const p = parsePayload(raw);
		if (!p || p.kind === 'team') throw new ApiError(400, 'qr_invalid', 'To nie jest kod stacji UniQuest');
		const now = Math.floor(Date.now() / 1000);
		if (p.kind === 'manual') {
			const win = windowAt(now, MANUAL_PERIOD_SECONDS);
			for (const st of w.stations) {
				const secret = await stationSecret(st.slug);
				for (const cand of [win, win - 1]) {
					if (safeEqual(await manualCodeFor(secret, st.id, cand), p.code)) return st;
				}
			}
			throw new ApiError(400, 'qr_invalid', 'Nieznany kod');
		}
		const st = w.stationById.get(p.stationId);
		if (!st) throw new ApiError(400, 'qr_invalid', 'Nieznana stacja');
		const secret = await stationSecret(st.slug);
		if (p.kind === 'static') {
			if (!safeEqual(await macFor(secret, staticMessage(st.id)), p.mac)) throw new ApiError(400, 'qr_invalid', 'Kod nie jest autentyczny');
			return st;
		}
		if (!safeEqual(await macFor(secret, rotatingMessage(st.id, p.window)), p.mac)) throw new ApiError(400, 'qr_invalid', 'Kod nie jest autentyczny');
		const age = windowAt(now) - p.window;
		const maxAge = queued ? QUEUED_GRACE_WINDOWS : 1;
		if (age > maxAge || age < -1) throw new ApiError(422, 'qr_expired', 'Kod wygasł — zeskanuj aktualny');
		return st;
	}

	async scan(req: ScanRequest): Promise<ScanResult> {
		await this.requireMe();
		const s = await this.load();
		const st = await this.resolveStation(req.payload, !!req.queued);
		if (s.closedStations.includes(st.id)) throw new ApiError(422, 'scan_window_closed', 'Skanowanie na tej stacji jest zamknięte');
		const existing = s.scans.find((x) => x.stationId === st.id);
		if (existing) return { scan: await this.scanDto(existing), alreadyScanned: true };

		const teamId = st.team && s.team ? s.team.id : null;
		const scan: DemoScan = {
			id: uuid(),
			stationId: st.id,
			scannedAt: req.queued && req.scannedAt ? req.scannedAt : nowIso(),
			rating: null,
			ratedAt: null,
			teamId,
			teamScanned: teamId ? 1 : 0,
			teamComplete: false
		};
		s.scans.push(scan);
		await this.awardBadges();
		await this.save();
		if (teamId && st.team) this.simulateTeamScans(scan.id, st.team.minMembers);
		return { scan: await this.scanDto(scan), alreadyScanned: false };
	}

	/** Bots in the player's team "arrive" and scan the same team station. */
	private simulateTeamScans(scanId: string, required: number) {
		// Each attempt lets one more team-mate scan, as long as the team has someone left to arrive.
		for (let i = 1; i <= 4; i++) {
			this.later(2500 * i, async () => {
				const s = await this.load();
				const sc = s.scans.find((x) => x.id === scanId);
				if (!sc || sc.teamComplete || !s.team || sc.teamScanned >= s.team.members.length) return;
				sc.teamScanned++;
				if (sc.teamScanned >= required) {
					sc.teamComplete = true;
					await this.awardBadges();
					await this.notify('team', 'Zadanie drużynowe zaliczone!', 'Cała drużyna dotarła — dostajecie bonus ×1,2.', '/hero');
				}
				await this.save();
				this.emit({ type: 'scan.confirmed', data: await this.scanDto(sc) });
				this.emit({ type: 'progress.updated', data: await this.progress() });
			});
		}
	}

	async rate(scanId: string, rating: number) {
		await this.requireMe();
		if (!Number.isInteger(rating) || rating < 1 || rating > 5) throw new ApiError(400, 'invalid_rating', 'Ocena 1–5');
		const s = await this.load();
		const sc = s.scans.find((x) => x.id === scanId);
		if (!sc) throw new ApiError(404, 'not_found', 'Nie ma takiej wizyty');
		if (sc.rating !== null && Date.now() - new Date(sc.scannedAt).getTime() > RATING_LOCK_MS) {
			throw new ApiError(409, 'rating_locked', 'Ocenę można zmienić tylko przez 15 minut');
		}
		const before = (await this.progress()).cardUnlocked;
		sc.rating = rating;
		sc.ratedAt = nowIso();
		await this.awardBadges();
		await this.save();
		const progress = await this.progress();
		if (!before && progress.cardUnlocked) {
			await this.notify('card', 'Karta postaci odblokowana!', 'Zobacz, jaką ścieżką podążasz.', '/card');
		}
		return { scan: await this.scanDto(sc), progress };
	}

	private async awardBadges() {
		const s = await this.load();
		const w = await demoWorld();
		const have = new Set(s.badges.map((b) => b.id));
		const add = async (id: Badge['id'], title: string) => {
			if (have.has(id)) return;
			have.add(id);
			s.badges.push({ id, awardedAt: nowIso() });
			await this.notify('badge', 'Nowa odznaka: ' + title, 'Sprawdź ją w panelu bohatera.', '/hero');
		};
		const visited = s.scans.map((x) => w.stationById.get(x.stationId)!);
		if (visited.length > 0) await add('first-scan', 'Pierwszy krok');
		if (faculties.every((f) => visited.some((v) => v.faculty === f.code))) await add('faculties-all', 'Odkrywca wydziałów');
		if (visited.some((v) => v.poi === 'cizt')) await add('bookworm', 'Mól książkowy');
		if (s.scans.filter((x) => x.teamComplete).length >= 3) await add('team-spirit', 'Duch drużyny');
		if (s.scans.some((x) => new Date(x.scannedAt).getHours() < 11)) await add('early-bird', 'Ranny ptaszek');
		if (s.scans.filter((x) => x.rating !== null).length >= w.event.config.minScansForCard) await add('card-unlocked', 'Karta postaci');
	}

	// ---------- teams ----------
	async myTeam() {
		await this.requireMe();
		return (await this.load()).team;
	}
	private async member(userId: string, nickname: string, role: TeamMember['role'], xp: number): Promise<TeamMember> {
		return { userId, nickname, avatarUrl: null, role, xp, joinedAt: nowIso() };
	}
	private async refreshTeamScore(team: Team) {
		const myXp = (await this.progress()).xp;
		const me = this.state!.me!;
		for (const m of team.members) if (m.userId === me.id) m.xp = myXp;
		team.score = Math.round(team.members.reduce((a, m) => a + m.xp, 0));
	}
	async createTeam(name: string, emblem: string) {
		const me = await this.requireMe();
		const s = await this.load();
		if (s.team) throw new ApiError(409, 'already_in_team', 'Masz już drużynę');
		const n = name.trim();
		if (n.length < 2 || n.length > 24) throw new ApiError(400, 'bad_request', 'Nazwa musi mieć 2–24 znaki');
		const team: Team = {
			id: uuid(),
			name: n,
			emblem: emblem || 'shield',
			captainId: me.id,
			members: [await this.member(me.id, me.nickname, 'captain', 0)],
			score: 0,
			maxMembers: 5,
			createdAt: nowIso()
		};
		await this.refreshTeamScore(team);
		s.team = team;
		await this.save();
		this.scheduleBots(team.id, 2);
		return team;
	}
	/** Simulated friends accept the invite a few seconds later (demonstrates live updates). */
	private scheduleBots(teamId: string, count: number) {
		for (let i = 0; i < count; i++) {
			this.later(4000 + i * 3500, async () => {
				const s = await this.load();
				if (!s.team || s.team.id !== teamId || s.team.members.length >= s.team.maxMembers) return;
				const name = BOT_NAMES.find((b) => !s.team!.members.some((m) => m.nickname === b)) ?? 'Gość';
				const m = await this.member(await uuidFrom('bot|' + teamId + '|' + name), name, 'member', 120 + (hashNum(name) % 300));
				s.team.members.push(m);
				await this.refreshTeamScore(s.team);
				await this.save();
				this.emit({ type: 'team.member_joined', data: { team: s.team, member: m } });
				await this.notify('team', `${name} dołącza do drużyny`, `Drużyna „${s.team.name}” ma już ${s.team.members.length} osoby.`, '/team');
			});
		}
	}
	async joinTeam(codeOrUrl: string) {
		const me = await this.requireMe();
		const s = await this.load();
		if (s.team) throw new ApiError(409, 'already_in_team', 'Masz już drużynę');
		const parsed = parsePayload(codeOrUrl);
		const code = parsed?.kind === 'team' ? parsed.code : codeOrUrl.trim().toUpperCase();
		if (!/^[A-Z0-9]{6,16}$/.test(code)) throw new ApiError(404, 'invite_invalid', 'Nieprawidłowe zaproszenie');
		// Demo: every valid-looking invite leads to a simulated team of two friends.
		const captainId = await uuidFrom('captain|' + code);
		const team: Team = {
			id: await uuidFrom('team|' + code),
			name: 'Drużyna ' + code.slice(0, 4),
			emblem: 'star',
			captainId,
			members: [
				await this.member(captainId, BOT_NAMES[hashNum(code) % 4]!, 'captain', 260),
				await this.member(await uuidFrom('m2|' + code), BOT_NAMES[(hashNum(code) + 1) % 4]!, 'member', 180),
				await this.member(me.id, me.nickname, 'member', 0)
			],
			score: 0,
			maxMembers: 5,
			createdAt: nowIso()
		};
		await this.refreshTeamScore(team);
		s.team = team;
		await this.save();
		return team;
	}
	async createInvite(teamId: string) {
		const me = await this.requireMe();
		const s = await this.load();
		if (!s.team || s.team.id !== teamId) throw new ApiError(404, 'not_found', 'Brak drużyny');
		if (s.team.captainId !== me.id) throw new ApiError(403, 'not_captain', 'Tylko kapitan może zapraszać');
		const code = randomCode(8);
		return {
			code,
			url: teamUrl(this.origin, code),
			expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
			usesLeft: s.team.maxMembers - s.team.members.length
		};
	}
	async leaveTeam(teamId: string) {
		const s = await this.load();
		if (s.team?.id === teamId) {
			s.team = null;
			await this.save();
			this.emit({ type: 'team.member_left', data: { team: null, userId: s.me?.id ?? '' } });
		}
	}
	async removeMember(teamId: string, userId: string) {
		const me = await this.requireMe();
		const s = await this.load();
		if (!s.team || s.team.id !== teamId) throw new ApiError(404, 'not_found', 'Brak drużyny');
		if (s.team.captainId !== me.id) throw new ApiError(403, 'not_captain', 'Tylko kapitan');
		s.team.members = s.team.members.filter((m) => m.userId !== userId);
		await this.refreshTeamScore(s.team);
		await this.save();
		this.emit({ type: 'team.updated', data: s.team });
	}

	async leaderboard(_eventId: string, type: 'team' | 'solo') {
		const s = await this.load();
		const myXp = s.me ? (await this.progress()).xp : 0;
		const hour = new Date().getHours();
		if (type === 'team') {
			if (s.team) await this.refreshTeamScore(s.team);
			const rows = RIVAL_TEAMS.map((name) => ({ id: name, name, score: 300 + (hashNum(name + hour) % 1400), members: 3 + (hashNum(name) % 3), isMe: false }));
			if (s.team) rows.push({ id: s.team.id, name: s.team.name, score: s.team.score, members: s.team.members.length, isMe: true });
			rows.sort((a, b) => b.score - a.score);
			return { type, entries: rows.map((r, i) => ({ rank: i + 1, ...r })) };
		}
		const rows = RIVAL_PLAYERS.map((name) => ({ id: name, name, score: 100 + (hashNum(name + hour) % 700), isMe: false }));
		if (s.me) rows.push({ id: s.me.id, name: s.me.nickname, score: myXp, isMe: true });
		rows.sort((a, b) => b.score - a.score);
		return { type, entries: rows.map((r, i) => ({ rank: i + 1, ...r })) };
	}

	// ---------- notifications & push ----------
	async notifications() {
		return (await this.load()).notifications;
	}
	async markNotificationsRead() {
		const s = await this.load();
		const t = nowIso();
		s.notifications = s.notifications.map((n) => (n.readAt ? n : { ...n, readAt: t }));
		await this.save();
	}
	async reminders() {
		return (await this.load()).reminders;
	}
	async setReminder(sessionId: string, on: boolean) {
		const s = await this.load();
		s.reminders = on ? [...new Set([...s.reminders, sessionId])] : s.reminders.filter((r) => r !== sessionId);
		await this.save();
	}
	async pushPublicKey() {
		return null; // the static demo has no push server; reminders are shown in-app instead
	}
	async pushSubscribe() {}
	async pushUnsubscribe() {}
	stream(onEvent: (e: StreamEvent) => void) {
		this.listeners.add(onEvent);
		return () => this.listeners.delete(onEvent);
	}

	// ---------- staff & admin (role switch available in demo settings) ----------
	private async staffDto(st: Station): Promise<StaffStation> {
		const s = await this.load();
		const mine = s.scans.filter((x) => x.stationId === st.id);
		const rated = mine.filter((x) => x.rating !== null);
		return {
			station: st,
			scanOpen: !s.closedStations.includes(st.id),
			scanOpenUntil: null,
			scansCount: mine.length,
			ratedCount: rated.length,
			avgRating: rated.length ? rated.reduce((a, x) => a + (x.rating ?? 0), 0) / rated.length : null
		};
	}
	async staffStations() {
		return Promise.all((await demoWorld()).stations.map((st) => this.staffDto(st)));
	}
	async stationQr(stationId: string) {
		const w = await demoWorld();
		const st = w.stationById.get(stationId);
		if (!st) throw new ApiError(404, 'not_found', 'Brak stacji');
		const now = Math.floor(Date.now() / 1000);
		const win = windowAt(now);
		const secret = await stationSecret(st.slug);
		const s = await this.load();
		return {
			payload: rotatingUrl(this.origin, st.id, win, await macFor(secret, rotatingMessage(st.id, win))),
			manualCode: await manualCodeFor(secret, st.id, windowAt(now, MANUAL_PERIOD_SECONDS)),
			validUntil: new Date((win + 1) * PERIOD_SECONDS * 1000).toISOString(),
			periodSeconds: PERIOD_SECONDS,
			scanOpen: !s.closedStations.includes(st.id),
			scansCount: s.scans.filter((x) => x.stationId === st.id).length
		};
	}
	/** Static (poster) payload — used by the demo's printable test sheet. */
	async staticPayload(stationId: string) {
		const w = await demoWorld();
		const st = w.stationById.get(stationId)!;
		const secret = await stationSecret(st.slug);
		return `${this.origin}/q/s.${st.id.replaceAll('-', '')}.${await macFor(secret, staticMessage(st.id))}`;
	}
	async stationPoster(stationId: string) {
		return this.staticPayload(stationId);
	}
	async setScanWindow(stationId: string, open: boolean) {
		const s = await this.load();
		s.closedStations = open ? s.closedStations.filter((x) => x !== stationId) : [...new Set([...s.closedStations, stationId])];
		await this.save();
		return this.staffDto((await demoWorld()).stationById.get(stationId)!);
	}
	async analytics() {
		const w = await demoWorld();
		const s = await this.load();
		const stations = w.stations.map((st) => {
			const base = 40 + (hashNum(st.slug) % 160);
			const mine = s.scans.filter((x) => x.stationId === st.id);
			const rated = Math.round(base * 0.85) + mine.filter((x) => x.rating !== null).length;
			return { stationId: st.id, name: st.name, visits: base + mine.length, rated, avgRating: 2.6 + (hashNum(st.slug + 'r') % 230) / 100 };
		});
		const codes = ['STR', 'INT', 'MAN', 'KRE', 'ENE', 'CHA', 'WIS', 'VIT'] as const;
		return {
			participants: 412 + (s.me ? 1 : 0),
			scans: stations.reduce((a, x) => a + x.visits, 0),
			rated: stations.reduce((a, x) => a + x.rated, 0),
			stations,
			archetypes: codes.map((code) => ({ code, users: 20 + (hashNum(code) % 90) }))
		};
	}
	analyticsCsvUrl() {
		return null;
	}
	async patchStation(stationId: string) {
		const st = (await demoWorld()).stationById.get(stationId);
		if (!st) throw new ApiError(404, 'not_found', 'Brak stacji');
		return st;
	}
	async setRole(_email: string, role: 'participant' | 'staff' | 'admin') {
		// Demo: switches the role of the current user (lets anyone explore staff/admin screens).
		const me = await this.requireMe();
		me.role = role;
		await this.save();
	}
	async broadcast(title: string, body: string) {
		await this.notify('broadcast', title, body);
	}
}

/** Plain-object deep copy that survives Svelte proxies and IndexedDB structured clone. */
function plain<T>(v: T): T {
	return JSON.parse(JSON.stringify(v)) as T;
}
