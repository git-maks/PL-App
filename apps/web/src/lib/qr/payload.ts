/*
 * QR payload formats (spec §7.4). The Go backend (internal/qr) implements the same formats;
 * shared vectors live in packages/contracts/fixtures/qr.
 *
 *   rotating station code : https://{host}/q/1.{stationHex}.{window}.{mac}
 *   static station poster : https://{host}/q/s.{stationHex}.{mac}
 *   team invite           : https://{host}/t/{code}
 *   manual station code   : 6 chars from MANUAL_ALPHABET (typed by hand)
 *
 * stationHex = station UUID without dashes; window = floor(unixSeconds / PERIOD);
 * mac = base64url(HMAC-SHA256(stationSecret, message))[0:12].
 */

export const PERIOD_SECONDS = 30;
export const MANUAL_PERIOD_SECONDS = 300;
export const MAC_LENGTH = 12;
export const MANUAL_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';

export type ParsedPayload =
	| { kind: 'rotating'; stationId: string; window: number; mac: string }
	| { kind: 'static'; stationId: string; mac: string }
	| { kind: 'team'; code: string }
	| { kind: 'manual'; code: string };

const HEX32 = /^[0-9a-f]{32}$/;
const MAC_RE = /^[A-Za-z0-9_-]{12}$/;
const TEAM_RE = /^[A-Za-z0-9]{6,16}$/;

export function hexToUuid(hex: string): string {
	return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
export function uuidToHex(uuid: string): string {
	return uuid.replaceAll('-', '').toLowerCase();
}

export function normalizeManualCode(input: string): string | null {
	const code = input.trim().toUpperCase().replace(/[\s-]/g, '');
	if (code.length !== 6) return null;
	for (const ch of code) if (!MANUAL_ALPHABET.includes(ch)) return null;
	return code;
}

/** Recognises our QR/URL formats; returns null for anything else (foreign QR codes). */
export function parsePayload(raw: string): ParsedPayload | null {
	const text = raw.trim();
	const manual = normalizeManualCode(text);
	if (manual && !text.includes('/')) return { kind: 'manual', code: manual };

	let path: string;
	try {
		path = new URL(text).pathname;
	} catch {
		return null;
	}
	const q = path.match(/\/q\/([^/]+)\/?$/);
	if (q?.[1]) {
		const parts = q[1].split('.');
		if (parts[0] === '1' && parts.length === 4) {
			const [, hex, w, mac] = parts as [string, string, string, string];
			const window = Number(w);
			if (HEX32.test(hex) && Number.isSafeInteger(window) && window > 0 && MAC_RE.test(mac)) {
				return { kind: 'rotating', stationId: hexToUuid(hex), window, mac };
			}
		}
		if (parts[0] === 's' && parts.length === 3) {
			const [, hex, mac] = parts as [string, string, string];
			if (HEX32.test(hex) && MAC_RE.test(mac)) return { kind: 'static', stationId: hexToUuid(hex), mac };
		}
		return null;
	}
	const t = path.match(/\/t\/([^/]+)\/?$/);
	if (t?.[1] && TEAM_RE.test(t[1])) return { kind: 'team', code: t[1].toUpperCase() };
	return null;
}

export function rotatingMessage(stationId: string, window: number): string {
	return `1|${uuidToHex(stationId)}|${window}`;
}
export function staticMessage(stationId: string): string {
	return `s|${uuidToHex(stationId)}`;
}
export function manualMessage(stationId: string, window: number): string {
	return `m|${uuidToHex(stationId)}|${window}`;
}

export function rotatingUrl(origin: string, stationId: string, window: number, mac: string): string {
	return `${origin}/q/1.${uuidToHex(stationId)}.${window}.${mac}`;
}
export function staticUrl(origin: string, stationId: string, mac: string): string {
	return `${origin}/q/s.${uuidToHex(stationId)}.${mac}`;
}
export function teamUrl(origin: string, code: string): string {
	return `${origin}/t/${code}`;
}

export function windowAt(unixSeconds: number, period = PERIOD_SECONDS): number {
	return Math.floor(unixSeconds / period);
}

// --- MAC computation (WebCrypto). Used by the demo backend; the real backend does this in Go. ---

function base64url(bytes: Uint8Array): string {
	let s = '';
	for (const b of bytes) s += String.fromCharCode(b);
	return btoa(s).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '');
}

async function hmac(secret: Uint8Array, message: string): Promise<Uint8Array> {
	const key = await crypto.subtle.importKey('raw', secret as BufferSource, { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
	return new Uint8Array(await crypto.subtle.sign('HMAC', key, new TextEncoder().encode(message)));
}

export async function macFor(secret: Uint8Array, message: string): Promise<string> {
	return base64url(await hmac(secret, message)).slice(0, MAC_LENGTH);
}

/** 6-character manual code: 30 bits of the HMAC mapped onto MANUAL_ALPHABET. */
export async function manualCodeFor(secret: Uint8Array, stationId: string, window: number): Promise<string> {
	const d = await hmac(secret, manualMessage(stationId, window));
	let bits = ((d[0]! << 24) | (d[1]! << 16) | (d[2]! << 8) | d[3]!) >>> 0;
	let out = '';
	for (let i = 0; i < 6; i++) {
		out += MANUAL_ALPHABET[bits & 31];
		bits >>>= 5;
	}
	return out;
}

/** Constant-time string comparison (both inputs are short ASCII). */
export function safeEqual(a: string, b: string): boolean {
	if (a.length !== b.length) return false;
	let diff = 0;
	for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
	return diff === 0;
}
