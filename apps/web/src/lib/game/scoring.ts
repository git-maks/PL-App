/*
 * UniQuest scoring engine (spec §8). Pure functions only.
 * The Go backend implements the same rules (internal/game); both are verified against the shared
 * golden fixtures in packages/contracts/fixtures/scoring. Change the rules in both places.
 */
import {
	STAT_CODES,
	type ArchetypeShare,
	type GameConfig,
	type Profile,
	type Recommendation,
	type ScoringInput,
	type StatCode,
	type Stats,
	type StatVector
} from './types';

const ORDER: Record<StatCode, number> = Object.fromEntries(STAT_CODES.map((c, i) => [c, i])) as Record<StatCode, number>;

export function round2(x: number): number {
	return Math.round(x * 100) / 100;
}

export function emptyStats(): Stats {
	return { STR: 0, INT: 0, MAN: 0, KRE: 0, ENE: 0, CHA: 0, WIS: 0, VIT: 0 };
}

export function ratingMultiplier(rating: number, cfg: GameConfig): number {
	const m = cfg.ratingMultipliers[rating - 1];
	if (m === undefined) throw new RangeError(`rating out of range: ${rating}`);
	return m;
}

/** Points a single rated visit adds to each stat: base × weight × f(rating) × teamBonus. */
export function scanDeltas(scan: ScoringInput, cfg: GameConfig): Stats {
	const out = emptyStats();
	if (scan.rating === null) return out;
	const f = ratingMultiplier(scan.rating, cfg) * (scan.team ? cfg.teamBonus : 1);
	for (const code of STAT_CODES) {
		const w = scan.weights[code] ?? 0;
		if (w > 0) out[code] = round2(scan.basePoints * w * f);
	}
	return out;
}

export function sumStats(deltas: Stats[]): Stats {
	const out = emptyStats();
	for (const d of deltas) for (const c of STAT_CODES) out[c] = round2(out[c] + d[c]);
	return out;
}

export function totalXp(stats: Stats): number {
	return round2(STAT_CODES.reduce((s, c) => s + stats[c], 0));
}

/** level = 1 + floor(sqrt(xp / divisor)); XP needed for level L is divisor × (L-1)². */
export function levelFor(xp: number, divisor: number): number {
	return 1 + Math.floor(Math.sqrt(Math.max(0, xp) / divisor));
}

export function levelProgress(xp: number, divisor: number) {
	const level = levelFor(xp, divisor);
	const current = divisor * (level - 1) ** 2;
	const next = divisor * level ** 2;
	return { current, next, ratio: (xp - current) / (next - current) };
}

/** Stats sorted by value desc; ties broken by canonical stat order so Go and TS agree. */
export function rankStats(stats: Stats): StatCode[] {
	return [...STAT_CODES].sort((a, b) => stats[b] - stats[a] || ORDER[a] - ORDER[b]);
}

/**
 * Largest-remainder rounding to `step` (e.g. 5%) so the parts always sum to 100.
 * Input values need not be normalised.
 */
export function apportion(values: number[], step: number): number[] {
	const units = Math.round(100 / step);
	const total = values.reduce((a, b) => a + b, 0);
	if (total <= 0) return values.map(() => 0);
	const quotas = values.map((v) => (v / total) * units);
	const floors = quotas.map((q) => Math.floor(q + 1e-9));
	let left = units - floors.reduce((a, b) => a + b, 0);
	const byRemainder = quotas
		.map((q, i) => ({ i, r: q - Math.floor(q + 1e-9) }))
		.sort((a, b) => b.r - a.r || a.i - b.i);
	for (const { i } of byRemainder) {
		if (left <= 0) break;
		floors[i] = (floors[i] ?? 0) + 1;
		left--;
	}
	return floors.map((f) => f * step);
}

export function archetypeShares(stats: Stats, cfg: GameConfig): ArchetypeShare[] {
	const xp = totalXp(stats);
	if (xp <= 0) return [];
	const ranked = rankStats(stats);
	let picked = ranked.filter((c) => stats[c] / xp >= cfg.archetypeThreshold).slice(0, cfg.maxArchetypes);
	if (picked.length === 0) picked = ranked.slice(0, 1);
	const percents = apportion(
		picked.map((c) => stats[c]),
		cfg.roundingStep
	);
	return picked.map((code, i) => ({ code, percent: percents[i] ?? 0 })).filter((a) => a.percent > 0);
}

/** Key into archetypes.json `hybrids` (pair in canonical order), or the single stat code. */
export function titleKey(archetypes: ArchetypeShare[]): string | null {
	const [a, b] = archetypes;
	if (!a) return null;
	if (!b) return a.code;
	return [a.code, b.code].sort((x, y) => ORDER[x] - ORDER[y]).join('+');
}

export function buildProfile(scans: ScoringInput[], cfg: GameConfig): Profile {
	const stats = sumStats(scans.map((s) => scanDeltas(s, cfg)));
	const xp = totalXp(stats);
	const archetypes = archetypeShares(stats, cfg);
	return {
		stats,
		xp,
		level: levelFor(xp, cfg.levelDivisor),
		levelProgress: levelProgress(xp, cfg.levelDivisor),
		ratedCount: scans.filter((s) => s.rating !== null).length,
		shares: rankStats(stats).map((code) => ({ code, share: xp > 0 ? stats[code] / xp : 0 })),
		archetypes,
		titleKey: titleKey(archetypes)
	};
}

function norm(v: StatVector | Stats): number {
	return Math.sqrt(STAT_CODES.reduce((s, c) => s + (v[c] ?? 0) ** 2, 0));
}

/** Cosine similarity between the player's stats and each item's attribute vector (spec §8.5). */
export function recommend<T extends { slug: string; vector: StatVector }>(
	stats: Stats,
	items: T[],
	limit: number
): Recommendation[] {
	const sn = norm(stats);
	if (sn === 0) return [];
	return items
		.map((item) => {
			const vn = norm(item.vector);
			const contrib = STAT_CODES.map((c) => ({ c, v: stats[c] * (item.vector[c] ?? 0) }));
			const dot = contrib.reduce((s, x) => s + x.v, 0);
			const because = contrib
				.filter((x) => x.v > 0)
				.sort((a, b) => b.v - a.v || ORDER[a.c] - ORDER[b.c])
				.slice(0, 2)
				.map((x) => x.c);
			return { slug: item.slug, score: vn === 0 ? 0 : Math.round((dot / (sn * vn)) * 100), because };
		})
		.sort((a, b) => b.score - a.score || (a.slug < b.slug ? -1 : 1))
		.slice(0, limit);
}
