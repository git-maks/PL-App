/*
 * Demo world: the demo event from content/demo-event.json turned into API objects
 * with deterministic ids (so printed QR sheets keep working across reloads and devices).
 */
import { archetypes } from '#lib/content/archetypes';
import demoEventJson from '#content/demo-event.json';
import type { DemoEventContent } from '#lib/content/types';
import type { Event, Session, Station } from '../types';

const demoEvent = demoEventJson as DemoEventContent;

async function sha256(text: string): Promise<Uint8Array> {
	return new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text)));
}

/** Deterministic RFC 4122-shaped UUID (version nibble 5) from a name. */
export async function uuidFrom(name: string): Promise<string> {
	const b = (await sha256('uniquest-demo|' + name)).slice(0, 16);
	b[6] = (b[6]! & 0x0f) | 0x50;
	b[8] = (b[8]! & 0x3f) | 0x80;
	const h = [...b].map((x) => x.toString(16).padStart(2, '0')).join('');
	return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
}

/** Per-station secret for the demo. Public by design — the demo has no secrets to protect. */
export function stationSecret(slug: string): Promise<Uint8Array> {
	return sha256('uniquest-demo-secret|' + slug);
}

function todayAt(hhmm: string, base = new Date()): Date {
	const [h, m] = hhmm.split(':').map(Number) as [number, number];
	const d = new Date(base);
	d.setHours(h, m, 0, 0);
	return d;
}

export interface DemoWorld {
	event: Event;
	stations: Station[];
	stationById: Map<string, Station>;
	slugById: Map<string, string>;
}

let worldPromise: Promise<DemoWorld> | null = null;

export function demoWorld(): Promise<DemoWorld> {
	return (worldPromise ??= build());
}

async function build(): Promise<DemoWorld> {
	const d = archetypes.defaults;
	const eventId = await uuidFrom('event|' + demoEvent.slug);
	const event: Event = {
		id: eventId,
		slug: demoEvent.slug,
		name: demoEvent.name,
		description: demoEvent.description,
		startsAt: todayAt(demoEvent.day.start).toISOString(),
		endsAt: todayAt(demoEvent.day.end).toISOString(),
		timezone: demoEvent.timezone,
		status: 'active',
		teamMin: 2,
		teamMax: 5,
		config: {
			ratingMultipliers: d.ratingMultipliers,
			levelDivisor: d.levelDivisor,
			teamBonus: d.teamBonus,
			archetypeThreshold: d.archetypeThreshold,
			maxArchetypes: d.maxArchetypes,
			roundingStep: d.roundingStep,
			minScansForCard: demoEvent.minScansForCard
		}
	};
	const stations: Station[] = [];
	for (const s of demoEvent.stations) {
		const id = await uuidFrom('station|' + s.slug);
		const sessions: Session[] = [];
		for (const [i, sess] of s.sessions.entries()) {
			const start = todayAt(sess.start);
			const end = todayAt(sess.end);
			sessions.push({
				id: await uuidFrom(`session|${s.slug}|${i}`),
				stationId: id,
				room: sess.room,
				startsAt: start.toISOString(),
				endsAt: end.toISOString(),
				scanOpensAt: new Date(end.getTime() - 10 * 60_000).toISOString(),
				scanClosesAt: new Date(end.getTime() + 15 * 60_000).toISOString()
			});
		}
		stations.push({
			id,
			slug: s.slug,
			name: s.name,
			description: s.description,
			kind: s.kind,
			faculty: s.faculty ?? null,
			poi: s.poi,
			basePoints: d.basePoints,
			weights: s.weights,
			team: s.team?.mode ? { minMembers: s.team.minMembers } : null,
			sessions
		});
	}
	return {
		event,
		stations,
		stationById: new Map(stations.map((s) => [s.id, s])),
		slugById: new Map(stations.map((s) => [s.id, s.slug]))
	};
}
