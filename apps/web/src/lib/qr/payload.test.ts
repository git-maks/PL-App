import { describe, expect, it } from 'vitest';
import vectors from '../../../../../packages/contracts/fixtures/qr/vectors.json';
import {
	macFor,
	manualCodeFor,
	normalizeManualCode,
	parsePayload,
	rotatingMessage,
	rotatingUrl,
	staticMessage,
	teamUrl
} from './payload';

const secret = Uint8Array.from(vectors.secretHex.match(/../g)!.map((h) => parseInt(h, 16)));

describe('QR vectors (shared with Go)', () => {
	it('rotating MACs', async () => {
		for (const c of vectors.cases) {
			expect(await macFor(secret, rotatingMessage(vectors.stationId, c.window))).toBe(c.mac);
		}
		const first = vectors.cases[0]!;
		expect(rotatingUrl('https://uniquest.example', vectors.stationId, first.window, first.mac)).toBe(first.url);
	});
	it('static MAC', async () => {
		expect(await macFor(secret, staticMessage(vectors.stationId))).toBe(vectors.static.mac);
	});
	it('manual codes', async () => {
		for (const m of vectors.manual) expect(await manualCodeFor(secret, vectors.stationId, m.window)).toBe(m.code);
	});
});

describe('parsePayload', () => {
	it('parses rotating URLs from any host', () => {
		const p = parsePayload(vectors.cases[0]!.url!);
		expect(p).toEqual({ kind: 'rotating', stationId: vectors.stationId, window: 58600000, mac: '2pNK4nIWuVSw' });
	});
	it('parses static and team URLs', () => {
		expect(parsePayload(vectors.static.url)?.kind).toBe('static');
		expect(parsePayload(teamUrl('https://x.pl/PL-App', 'ab12cd34'))).toEqual({ kind: 'team', code: 'AB12CD34' });
	});
	it('parses manual codes and rejects ambiguous characters', () => {
		expect(parsePayload(' ukz-kcj ')).toEqual({ kind: 'manual', code: 'UKZKCJ' });
		expect(normalizeManualCode('ABCDE0')).toBeNull();
	});
	it('rejects foreign codes', () => {
		expect(parsePayload('https://example.com/menu')).toBeNull();
		expect(parsePayload('WIFI:S:eduroam;;')).toBeNull();
		expect(parsePayload('https://x.pl/q/1.zz.1.abc')).toBeNull();
	});
});
