import { describe, expect, it } from 'vitest';
import fixtures from '../../../../../packages/contracts/fixtures/scoring/cases.json';
import { apportion, buildProfile, levelFor, recommend } from './scoring';
import type { GameConfig, ScoringInput, StatVector } from './types';
import { clubs, programs } from '#lib/content';

const cfg = fixtures.config as GameConfig;

describe('golden fixtures', () => {
	for (const c of fixtures.cases) {
		it(c.name, () => {
			const p = buildProfile(c.scans as ScoringInput[], cfg);
			expect(p.stats).toEqual(c.expected.stats);
			expect(p.xp).toBe(c.expected.xp);
			expect(p.level).toBe(c.expected.level);
			expect(p.archetypes).toEqual(c.expected.archetypes);
			expect(p.titleKey).toBe(c.expected.titleKey);
			const exp = c.expected as { programs?: string[]; clubs?: string[] };
			if (exp.programs) expect(recommend(p.stats, programs, 3).map((r) => r.slug)).toEqual(exp.programs);
			if (exp.clubs) expect(recommend(p.stats, clubs, 3).map((r) => r.slug)).toEqual(exp.clubs);
		});
	}
});

describe('apportion', () => {
	it('always sums to 100', () => {
		for (const v of [[1, 1, 1], [5, 3, 1], [0.7, 0.2], [185, 125, 105], [1]]) {
			expect(apportion(v, 5).reduce((a, b) => a + b, 0)).toBe(100);
		}
	});
});

describe('levels', () => {
	it('follows 1 + floor(sqrt(xp/25))', () => {
		expect(levelFor(0, 25)).toBe(1);
		expect(levelFor(24.99, 25)).toBe(1);
		expect(levelFor(25, 25)).toBe(2);
		expect(levelFor(100, 25)).toBe(3);
	});
});

describe('recommendations', () => {
	it('match a mechanical profile to mechanical programmes', () => {
		const p = buildProfile([{ basePoints: 100, weights: { STR: 0.8, MAN: 0.2 } as StatVector, rating: 5, team: false }], cfg);
		const top = recommend(p.stats, programs, 3).map((r) => r.slug);
		expect(top).toContain('mechanika-i-budowa-maszyn');
	});
	it('explains with the strongest contributing stats', () => {
		const p = buildProfile([{ basePoints: 100, weights: { INT: 0.5, MAN: 0.5 }, rating: 4, team: false }], cfg);
		const [first] = recommend(p.stats, programs, 1);
		expect(first?.slug).toBe('cyberbezpieczenstwo');
		expect(first?.because).toEqual(['MAN', 'INT']);
	});
});
