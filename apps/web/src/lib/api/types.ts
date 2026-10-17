import type { components } from './schema';

type S = components['schemas'];
export type Me = S['Me'];
export type MePatch = S['MePatch'];
export type Event = S['Event'];
export type Station = S['Station'];
export type Session = S['Session'];
export type Scan = S['Scan'];
export type ScanRequest = S['ScanRequest'];
export type ScanResult = S['ScanResult'];
export type RatingResult = S['RatingResult'];
export type Progress = S['Progress'];
export type Character = S['Character'];
export type Recommendation = S['Recommendation'];
export type Team = S['Team'];
export type TeamMember = S['TeamMember'];
export type TeamInvite = S['TeamInvite'];
export type Leaderboard = S['Leaderboard'];
export type Notification = S['Notification'];
export type Badge = S['Badge'];
export type StaffStation = S['StaffStation'];
export type StationQr = S['StationQr'];
export type Analytics = S['Analytics'];
export type EmailStartResponse = S['EmailStartResponse'];
export type Problem = S['Problem'];
export type ProblemCode = Problem['code'];
export type PushSubscriptionDto = S['PushSubscription'];
export type StationPatch = S['StationPatch'];

export type StreamEvent =
	| { type: 'notification'; data: Notification }
	| { type: 'team.updated'; data: Team }
	| { type: 'team.member_joined'; data: { team: Team; member: TeamMember } }
	| { type: 'team.member_left'; data: { team: Team | null; userId: string } }
	| { type: 'scan.confirmed'; data: Scan }
	| { type: 'progress.updated'; data: Progress };

export class ApiError extends Error {
	readonly status: number;
	readonly code: ProblemCode;
	readonly detail?: string;
	constructor(status: number, code: ProblemCode, title: string, detail?: string) {
		super(detail ?? title);
		this.status = status;
		this.code = code;
		this.detail = detail;
	}
}

/** Thrown when the request could not reach the server (offline, DNS, timeout). */
export class NetworkError extends Error {}

/**
 * Everything the UI needs from a backend. Two implementations:
 * HttpApi (Go backend) and DemoApi (in-browser simulation for the public GitHub Pages demo).
 */
export interface ApiClient {
	readonly mode: 'http' | 'demo';

	me(): Promise<Me | null>;
	createGuest(nickname?: string): Promise<Me>;
	emailStart(email: string, ageConfirmed: boolean): Promise<EmailStartResponse>;
	emailVerify(email: string, code: string): Promise<Me>;
	logout(): Promise<void>;
	updateMe(patch: MePatch): Promise<Me>;
	uploadAvatar(pngDataUrl: string): Promise<Me>;
	deleteAvatar(): Promise<Me>;
	deleteAccount(): Promise<void>;
	exportData(): Promise<unknown>;

	activeEvent(): Promise<Event>;
	stations(eventId: string): Promise<Station[]>;
	leaderboard(eventId: string, type: 'team' | 'solo'): Promise<Leaderboard>;

	progress(): Promise<Progress>;
	character(): Promise<Character>;
	scan(req: ScanRequest): Promise<ScanResult>;
	rate(scanId: string, rating: number): Promise<RatingResult>;

	myTeam(): Promise<Team | null>;
	createTeam(name: string, emblem: string): Promise<Team>;
	joinTeam(code: string): Promise<Team>;
	createInvite(teamId: string): Promise<TeamInvite>;
	leaveTeam(teamId: string): Promise<void>;
	removeMember(teamId: string, userId: string): Promise<void>;

	notifications(): Promise<Notification[]>;
	markNotificationsRead(): Promise<void>;
	reminders(): Promise<string[]>;
	setReminder(sessionId: string, on: boolean): Promise<void>;
	pushPublicKey(): Promise<string | null>;
	pushSubscribe(sub: PushSubscriptionDto): Promise<void>;
	pushUnsubscribe(endpoint: string): Promise<void>;
	/** Subscribe to server events; returns an unsubscribe function. */
	stream(onEvent: (e: StreamEvent) => void): () => void;

	staffStations(): Promise<StaffStation[]>;
	stationQr(stationId: string): Promise<StationQr>;
	setScanWindow(stationId: string, open: boolean, minutes?: number): Promise<StaffStation>;

	analytics(eventId: string): Promise<Analytics>;
	analyticsCsvUrl(eventId: string): string | null;
	patchStation(stationId: string, patch: StationPatch): Promise<Station>;
	setRole(email: string, role: 'participant' | 'staff' | 'admin'): Promise<void>;
	broadcast(title: string, body: string): Promise<void>;
}
