import campusJson from '#content/campus.json';
import type { CampusContent, Poi } from './types';

export const campus = campusJson as CampusContent;
export const poiById = Object.fromEntries(campus.pois.map((p) => [p.id, p])) as Record<string, Poi>;

import facultiesJson from '#content/faculties.json';
/** Faculty code → knowledge-base slug (small lookup so the map does not pull the whole catalogue). */
export const facultySlugByCode: Record<string, string> = Object.fromEntries(
	(facultiesJson as { code: string; slug: string }[]).map((f) => [f.code, f.slug])
);
