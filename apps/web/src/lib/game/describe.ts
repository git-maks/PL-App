import { statByCode } from '#lib/content/archetypes';
import type { ArchetypeShare } from './types';

/** "Twoja ścieżka to w 70% Konstruktor, w 30% Analityk" (spec §8.4). */
export function describePath(archetypes: ArchetypeShare[]): string {
	if (archetypes.length === 0) return 'Twoja ścieżka dopiero się kształtuje.';
	const parts = archetypes.map((a) => `w ${a.percent}% ${statByCode[a.code].archetype}`);
	const list = parts.length > 1 ? parts.slice(0, -1).join(', ') + ' i ' + parts.at(-1) : parts[0];
	return `Twoja ścieżka to ${list}.`;
}
