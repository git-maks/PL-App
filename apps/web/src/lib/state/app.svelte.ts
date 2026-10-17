/*
 * Application state (Svelte 5 runes). One instance per tab; screens read from it and call its
 * actions. The server stays authoritative — this class caches its answers, keeps the offline
 * scan queue and applies live updates from the event stream.
 */
import { browser } from '$app/env';
import {
	api,
	ApiError,
	NetworkError,
	type ApiClient,
	type Event,
	type Me,
	type Notification,
	type Progress,
	type ScanResult,
	type Station,
	type StreamEvent,
	type Team
} from '#lib/api';
import { enqueue, listQueue, removeQueued, updateQueued, type QueuedScan } from '#lib/offline/queue';
import type { Stats } from '#lib/game/types';
import { toast } from './toast.svelte';

export type ScanOutcome =
	| { kind: 'ok'; result: ScanResult }
	| { kind: 'queued'; item: QueuedScan }
	| { kind: 'error'; error: ApiError };

export const ERROR_MESSAGES: Partial<Record<ApiError['code'], string>> = {
	qr_invalid: 'Ten kod nie należy do UniQuest albo jest uszkodzony.',
	qr_expired: 'Kod już się zmienił — zeskanuj ten, który jest teraz na ekranie.',
	scan_window_closed: 'Skanowanie na tej stacji jest teraz zamknięte. Kod pojawia się pod koniec zajęć.',
	no_active_event: 'Teraz nie trwa żadne wydarzenie UniQuest.',
	rating_locked: 'Ocenę można zmienić tylko przez 15 minut od skanu.',
	team_full: 'Ta drużyna jest już pełna.',
	already_in_team: 'Należysz już do drużyny.',
	invite_invalid: 'Zaproszenie jest nieważne lub wygasło.',
	not_captain: 'Tylko kapitan może to zrobić.',
	rate_limited: 'Za dużo prób — odczekaj chwilę.',
	invalid_code: 'Kod jest niepoprawny.',
	code_expired: 'Kod wygasł — wyślij nowy.',
	invalid_email: 'Sprawdź adres e-mail.',
	invalid_nickname: 'Pseudonim musi mieć 2–24 znaki.'
};

export function errorMessage(e: unknown): string {
	if (e instanceof ApiError) return ERROR_MESSAGES[e.code] ?? e.message;
	if (e instanceof NetworkError) return 'Brak połączenia z serwerem.';
	return 'Coś poszło nie tak. Spróbuj ponownie.';
}

class AppState {
	ready = $state(false);
	bootError = $state<string | null>(null);
	me = $state<Me | null>(null);
	event = $state<Event | null>(null);
	stations = $state<Station[]>([]);
	progress = $state<Progress | null>(null);
	team = $state<Team | null>(null);
	notifications = $state<Notification[]>([]);
	reminders = $state<string[]>([]);
	queue = $state<QueuedScan[]>([]);
	online = $state(true);
	/** Stat gains from the last rating — consumed by the hero screen animation. */
	lastDeltas = $state<Stats | null>(null);

	unread = $derived(this.notifications.filter((n) => !n.readAt).length);
	pendingRatings = $derived(this.progress?.scans.filter((s) => s.rating === null).length ?? 0);
	stationById = $derived(new Map(this.stations.map((s) => [s.id, s])));

	private client: ApiClient | null = null;
	private unsubscribe: (() => void) | null = null;
	private flushing = false;

	get api(): ApiClient {
		if (!this.client) throw new Error('AppState not initialised');
		return this.client;
	}

	async init() {
		if (!browser || this.client) return;
		this.client = await api();
		this.online = navigator.onLine;
		addEventListener('online', () => {
			this.online = true;
			void this.flushQueue();
			void this.refresh();
		});
		addEventListener('offline', () => (this.online = false));
		document.addEventListener('visibilitychange', () => {
			if (document.visibilityState === 'visible') void this.refresh();
		});
		try {
			this.queue = await listQueue();
			this.me = await this.client.me();
			await this.loadEvent();
			if (this.me) await this.afterLogin();
		} catch (e) {
			this.bootError = errorMessage(e);
		} finally {
			this.ready = true;
		}
		setInterval(() => void this.flushQueue(), 30_000);
	}

	private async loadEvent() {
		try {
			this.event = await this.api.activeEvent();
			this.stations = await this.api.stations(this.event.id);
		} catch (e) {
			if (!(e instanceof ApiError && e.code === 'no_active_event')) throw e;
			this.event = null;
			this.stations = [];
		}
	}

	private async afterLogin() {
		this.unsubscribe?.();
		this.unsubscribe = this.api.stream((e) => this.onStream(e));
		await Promise.all([this.refresh(), this.flushQueue()]);
	}

	async refresh() {
		if (!this.client || !this.me) return;
		try {
			const [progress, team, notifications, reminders] = await Promise.all([
				this.event ? this.api.progress() : Promise.resolve(null),
				this.event ? this.api.myTeam() : Promise.resolve(null),
				this.api.notifications(),
				this.api.reminders()
			]);
			this.progress = progress;
			this.team = team;
			this.notifications = notifications;
			this.reminders = reminders;
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				this.me = null;
				return;
			}
			if (!(e instanceof NetworkError)) console.warn('refresh failed', e);
		}
	}

	private onStream(e: StreamEvent) {
		switch (e.type) {
			case 'notification':
				this.notifications = [e.data, ...this.notifications.filter((n) => n.id !== e.data.id)];
				toast(e.data.title);
				break;
			case 'team.updated':
				this.team = e.data;
				break;
			case 'team.member_joined':
				this.team = e.data.team;
				break;
			case 'team.member_left':
				this.team = e.data.team;
				break;
			case 'progress.updated':
				this.progress = e.data;
				break;
			case 'scan.confirmed':
				if (this.progress) {
					this.progress.scans = this.progress.scans.map((s) => (s.id === e.data.id ? e.data : s));
				}
				break;
		}
	}

	// ---------- account ----------
	async startAsGuest(nickname?: string) {
		this.me = await this.api.createGuest(nickname);
		await this.afterLogin();
	}

	async setMe(me: Me) {
		const wasLoggedIn = !!this.me;
		this.me = me;
		if (!wasLoggedIn) await this.afterLogin();
	}

	async logout() {
		await this.api.logout();
		this.unsubscribe?.();
		this.me = null;
		this.progress = null;
		this.team = null;
		this.notifications = [];
	}

	// ---------- scanning ----------
	async submitScan(payload: string): Promise<ScanOutcome> {
		if (!this.online) return { kind: 'queued', item: await this.queueScan(payload) };
		try {
			const result = await this.api.scan({ payload });
			await this.refresh();
			return { kind: 'ok', result };
		} catch (e) {
			if (e instanceof NetworkError) return { kind: 'queued', item: await this.queueScan(payload) };
			if (e instanceof ApiError) return { kind: 'error', error: e };
			throw e;
		}
	}

	private async queueScan(payload: string) {
		const item = await enqueue(payload);
		this.queue = await listQueue();
		return item;
	}

	async rate(scanId: string, rating: number) {
		const res = await this.api.rate(scanId, rating);
		this.lastDeltas = res.scan.deltas;
		this.progress = res.progress;
		return res;
	}

	async rateQueued(id: string, rating: number) {
		const item = this.queue.find((q) => q.id === id);
		if (!item) return;
		await updateQueued({ ...$state.snapshot(item), rating });
		this.queue = await listQueue();
	}

	/** Replays offline scans in order; stops at the first network failure. */
	async flushQueue() {
		if (this.flushing || !this.client || !this.me || !this.online) return;
		const items = await listQueue();
		if (items.length === 0) return;
		this.flushing = true;
		let synced = 0;
		try {
			for (const item of items) {
				try {
					const res = await this.api.scan({ payload: item.payload, scannedAt: item.scannedAt, queued: true });
					if (item.rating !== null && res.scan.rating === null) await this.api.rate(res.scan.id, item.rating);
					await removeQueued(item.id);
					synced++;
				} catch (e) {
					if (e instanceof NetworkError) break;
					await removeQueued(item.id);
					toast('Nie udało się zapisać skanu z kolejki: ' + errorMessage(e));
				}
			}
		} finally {
			this.flushing = false;
			this.queue = await listQueue();
		}
		if (synced > 0) {
			toast(synced === 1 ? 'Zsynchronizowano zapisany skan.' : `Zsynchronizowano ${synced} zapisane skany.`);
			await this.refresh();
		}
	}

	// ---------- teams ----------
	async createTeam(name: string, emblem: string) {
		this.team = await this.api.createTeam(name, emblem);
	}
	async joinTeam(code: string) {
		this.team = await this.api.joinTeam(code);
	}
	async leaveTeam() {
		if (!this.team) return;
		await this.api.leaveTeam(this.team.id);
		this.team = null;
	}

	// ---------- reminders & notifications ----------
	async toggleReminder(sessionId: string) {
		const on = !this.reminders.includes(sessionId);
		await this.api.setReminder(sessionId, on);
		this.reminders = on ? [...this.reminders, sessionId] : this.reminders.filter((r) => r !== sessionId);
		return on;
	}
	async markAllRead() {
		await this.api.markNotificationsRead();
		const t = new Date().toISOString();
		this.notifications = this.notifications.map((n) => (n.readAt ? n : { ...n, readAt: t }));
	}
}

export const app = new AppState();
