import archetypesJson from '#content/archetypes.json';
import type { ArchetypesContent, StatDef } from './types';
import type { GameConfig, StatCode } from '#lib/game/types';

/** Small, always-needed game content (stats, archetypes, titles). Kept separate from the catalogue. */
export const archetypes = archetypesJson as ArchetypesContent;
export const statByCode = Object.fromEntries(archetypes.stats.map((s) => [s.code, s])) as Record<StatCode, StatDef>;

export const defaultGameConfig: GameConfig = {
	ratingMultipliers: archetypes.defaults.ratingMultipliers,
	levelDivisor: archetypes.defaults.levelDivisor,
	teamBonus: archetypes.defaults.teamBonus,
	archetypeThreshold: archetypes.defaults.archetypeThreshold,
	maxArchetypes: archetypes.defaults.maxArchetypes,
	roundingStep: archetypes.defaults.roundingStep,
	minScansForCard: archetypes.defaults.minScansForCard
};

/** Human title for a profile's titleKey ("STR+KRE" → hybrid, "MAN" → single archetype). */
export function titleFor(key: string | null | undefined): { title: string; description: string } | null {
	if (!key) return null;
	const hybrid = archetypes.hybrids[key];
	if (hybrid) return hybrid;
	const stat = statByCode[key as StatCode];
	return stat ? { title: stat.archetype, description: stat.description } : null;
}
