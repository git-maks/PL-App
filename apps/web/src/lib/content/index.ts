/*
 * Knowledge-base catalogue (faculties, programmes, clubs, campus, …). Heavier than the game
 * content – import it from screens that need it, not from shared components.
 */
import facultiesJson from '#content/faculties.json';
import programsJson from '#content/programs.json';
import clubsJson from '#content/clubs.json';
import universityJson from '#content/university.json';
import studentLifeJson from '#content/student-life.json';
import recruitmentJson from '#content/recruitment.json';
import type {
	Club,
	Faculty,
	Program,
	RecruitmentContent,
	StudentLifeContent,
	UniversityContent
} from './types';

export { archetypes, statByCode, defaultGameConfig, titleFor } from './archetypes';
export { campus, poiById } from './campus';

export const faculties = facultiesJson as Faculty[];
export const programs = programsJson as Program[];
export const clubs = clubsJson as Club[];
export const university = universityJson as UniversityContent;
export const studentLife = studentLifeJson as StudentLifeContent;
export const recruitment = recruitmentJson as RecruitmentContent;

export const facultyByCode = Object.fromEntries(faculties.map((f) => [f.code, f])) as Record<string, Faculty>;
export const facultyBySlug = Object.fromEntries(faculties.map((f) => [f.slug, f])) as Record<string, Faculty>;
export const programBySlug = Object.fromEntries(programs.map((p) => [p.slug, p])) as Record<string, Program>;
export const clubBySlug = Object.fromEntries(clubs.map((c) => [c.slug, c])) as Record<string, Club>;
