import {
	ApiError,
	NetworkError,
	type ApiClient,
	type Me,
	type Problem,
	type StreamEvent
} from './types';

/** HTTP implementation talking to the Go backend (same origin, cookie session). */
export class HttpApi implements ApiClient {
	readonly mode = 'http' as const;
	constructor(private readonly root: string) {}

	private async req<T>(method: string, path: string, body?: unknown): Promise<T> {
		let res: Response;
		try {
			res = await fetch(this.root + path, {
				method,
				credentials: 'same-origin',
				headers: {
					Accept: 'application/json',
					...(method !== 'GET' ? { 'X-Requested-With': 'uniquest' } : {}),
					...(body !== undefined ? { 'Content-Type': 'application/json' } : {})
				},
				body: body !== undefined ? JSON.stringify(body) : undefined
			});
		} catch (e) {
			throw new NetworkError(e instanceof Error ? e.message : 'network error');
		}
		if (res.status === 204) return undefined as T;
		const isJson = (res.headers.get('content-type') ?? '').includes('json');
		const data: unknown = isJson ? await res.json() : await res.text();
		if (!res.ok) {
			const p = (isJson ? data : {}) as Partial<Problem>;
			throw new ApiError(res.status, p.code ?? 'internal', p.title ?? res.statusText, p.detail);
		}
		return data as T;
	}

	async me() {
		try {
			return await this.req<Me>('GET', '/me');
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) return null;
			throw e;
		}
	}
	createGuest(nickname?: string) {
		return this.req<Me>('POST', '/auth/guest', nickname ? { nickname } : {});
	}
	emailStart(email: string, ageConfirmed: boolean) {
		return this.req<{ sent: boolean; devCode?: string }>('POST', '/auth/email/start', { email, ageConfirmed });
	}
	emailVerify(email: string, code: string) {
		return this.req<Me>('POST', '/auth/email/verify', { email, code });
	}
	logout() {
		return this.req<void>('POST', '/auth/logout');
	}
	updateMe(patch: Parameters<ApiClient['updateMe']>[0]) {
		return this.req<Me>('PATCH', '/me', patch);
	}
	uploadAvatar(png: string) {
		return this.req<Me>('PUT', '/me/avatar', { png });
	}
	deleteAvatar() {
		return this.req<Me>('DELETE', '/me/avatar');
	}
	deleteAccount() {
		return this.req<void>('DELETE', '/me');
	}
	exportData() {
		return this.req<unknown>('GET', '/me/export');
	}

	activeEvent() {
		return this.req<Awaited<ReturnType<ApiClient['activeEvent']>>>('GET', '/events/active');
	}
	stations(eventId: string) {
		return this.req<Awaited<ReturnType<ApiClient['stations']>>>('GET', `/events/${eventId}/stations`);
	}
	leaderboard(eventId: string, type: 'team' | 'solo') {
		return this.req<Awaited<ReturnType<ApiClient['leaderboard']>>>('GET', `/events/${eventId}/leaderboard?type=${type}`);
	}

	progress() {
		return this.req<Awaited<ReturnType<ApiClient['progress']>>>('GET', '/me/progress');
	}
	character() {
		return this.req<Awaited<ReturnType<ApiClient['character']>>>('GET', '/me/character');
	}
	scan(r: Parameters<ApiClient['scan']>[0]) {
		return this.req<Awaited<ReturnType<ApiClient['scan']>>>('POST', '/scans', r);
	}
	rate(scanId: string, rating: number) {
		return this.req<Awaited<ReturnType<ApiClient['rate']>>>('PUT', `/scans/${scanId}/rating`, { rating });
	}

	async myTeam() {
		try {
			return await this.req<Awaited<ReturnType<ApiClient['createTeam']>>>('GET', '/teams/mine');
		} catch (e) {
			if (e instanceof ApiError && e.status === 404) return null;
			throw e;
		}
	}
	createTeam(name: string, emblem: string) {
		return this.req<Awaited<ReturnType<ApiClient['createTeam']>>>('POST', '/teams', { name, emblem });
	}
	joinTeam(code: string) {
		return this.req<Awaited<ReturnType<ApiClient['joinTeam']>>>('POST', '/teams/join', { code });
	}
	createInvite(teamId: string) {
		return this.req<Awaited<ReturnType<ApiClient['createInvite']>>>('POST', `/teams/${teamId}/invites`);
	}
	leaveTeam(teamId: string) {
		return this.req<void>('POST', `/teams/${teamId}/leave`);
	}
	removeMember(teamId: string, userId: string) {
		return this.req<void>('DELETE', `/teams/${teamId}/members/${userId}`);
	}

	notifications() {
		return this.req<Awaited<ReturnType<ApiClient['notifications']>>>('GET', '/me/notifications');
	}
	markNotificationsRead() {
		return this.req<void>('POST', '/me/notifications/read');
	}
	reminders() {
		return this.req<string[]>('GET', '/me/reminders');
	}
	setReminder(sessionId: string, on: boolean) {
		return this.req<void>(on ? 'PUT' : 'DELETE', `/sessions/${sessionId}/reminder`);
	}
	async pushPublicKey() {
		try {
			return (await this.req<{ key: string }>('GET', '/push/vapid-public-key')).key || null;
		} catch {
			return null;
		}
	}
	pushSubscribe(sub: Parameters<ApiClient['pushSubscribe']>[0]) {
		return this.req<void>('POST', '/push/subscriptions', sub);
	}
	pushUnsubscribe(endpoint: string) {
		return this.req<void>('DELETE', '/push/subscriptions', { endpoint });
	}

	stream(onEvent: (e: StreamEvent) => void): () => void {
		if (typeof EventSource === 'undefined') return () => {};
		const es = new EventSource(this.root + '/stream');
		const types: StreamEvent['type'][] = [
			'notification',
			'team.updated',
			'team.member_joined',
			'team.member_left',
			'scan.confirmed',
			'progress.updated'
		];
		for (const type of types) {
			es.addEventListener(type, (m) => {
				try {
					onEvent({ type, data: JSON.parse((m as MessageEvent<string>).data) } as StreamEvent);
				} catch {
					/* ignore malformed event */
				}
			});
		}
		return () => es.close();
	}

	staffStations() {
		return this.req<Awaited<ReturnType<ApiClient['staffStations']>>>('GET', '/staff/stations');
	}
	stationQr(stationId: string) {
		return this.req<Awaited<ReturnType<ApiClient['stationQr']>>>('GET', `/staff/stations/${stationId}/qr`);
	}
	setScanWindow(stationId: string, open: boolean, minutes = 15) {
		return this.req<Awaited<ReturnType<ApiClient['setScanWindow']>>>('POST', `/staff/stations/${stationId}/window`, { open, minutes });
	}

	analytics(eventId: string) {
		return this.req<Awaited<ReturnType<ApiClient['analytics']>>>('GET', `/admin/events/${eventId}/analytics`);
	}
	analyticsCsvUrl(eventId: string) {
		return `${this.root}/admin/events/${eventId}/analytics.csv`;
	}
	patchStation(stationId: string, patch: Parameters<ApiClient['patchStation']>[1]) {
		return this.req<Awaited<ReturnType<ApiClient['patchStation']>>>('PATCH', `/admin/stations/${stationId}`, patch);
	}
	setRole(email: string, role: 'participant' | 'staff' | 'admin') {
		return this.req<void>('POST', '/admin/users/role', { email, role });
	}
	broadcast(title: string, body: string) {
		return this.req<void>('POST', '/admin/broadcast', { title, body });
	}
}
