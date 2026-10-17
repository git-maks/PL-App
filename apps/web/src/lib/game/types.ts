export const STAT_CODES = ['STR', 'INT', 'MAN', 'KRE', 'ENE', 'CHA', 'WIS', 'VIT'] as const;
export type StatCode = (typeof STAT_CODES)[number];
export type StatVector = Partial<Record<StatCode, number>>;
export type Stats = Record<StatCode, number>;

export interface GameConfig {
	ratingMultipliers: number[];
	levelDivisor: number;
	teamBonus: number;
	archetypeThreshold: number;
	maxArchetypes: number;
	roundingStep: number;
	minScansForCard: number;
}

/** A rated (or not yet rated) visit, as the scoring engine sees it. */
export interface ScoringInput {
	basePoints: number;
	weights: StatVector;
	rating: number | null;
	team: boolean;
}

export interface ArchetypeShare {
	code: StatCode;
	percent: number;
}

export interface Profile {
	stats: Stats;
	xp: number;
	level: number;
	levelProgress: { current: number; next: number; ratio: number };
	ratedCount: number;
	shares: { code: StatCode; share: number }[];
	archetypes: ArchetypeShare[];
	titleKey: string | null;
}

export interface Recommendation {
	slug: string;
	score: number;
	because: StatCode[];
}
