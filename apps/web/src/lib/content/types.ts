import type { StatCode, StatVector } from '#lib/game/types';

export interface Source {
	url: string;
	title: string;
}

export interface StatDef {
	code: StatCode;
	name: string;
	archetype: string;
	icon: string;
	color: string;
	description: string;
}

export interface GameDefaults {
	ratingMultipliers: number[];
	minScansForCard: number;
	levelDivisor: number;
	teamBonus: number;
	archetypeThreshold: number;
	maxArchetypes: number;
	roundingStep: number;
	basePoints: number;
}

export interface ArchetypesContent {
	verifiedAt: string;
	stats: StatDef[];
	hybrids: Record<string, { title: string; description: string }>;
	rating: { labels: string[]; emoji: string[]; question: string };
	defaults: GameDefaults;
}

export interface Faculty {
	code: string;
	slug: string;
	name: string;
	address: string;
	poi: string;
	website: string;
	founded: number | null;
	description: string;
	vector: StatVector;
	programs: string[];
	clubs: string[];
	sources: Source[];
	verifiedAt: string;
}

export interface Program {
	slug: string;
	name: string;
	faculty: string;
	levels: string[];
	modes: string[];
	practicalProfile: boolean;
	language: 'pl' | 'en';
	studies: string;
	description: string;
	careersIntro: string;
	careers: string[];
	didYouKnow: string;
	maturaSubjects: string[];
	vector: StatVector;
	recruitmentUrl: string;
	sources: Source[];
	verifiedAt: string;
	note?: string;
}

export interface Club {
	slug: string;
	name: string;
	faculty: string;
	tags: string[];
	vector: StatVector;
	sources: Source[];
	verifiedAt: string;
}

export interface Poi {
	id: string;
	n: number;
	name: string;
	address: string;
	kind: 'faculty' | 'lab' | 'landmark' | 'service' | 'dorm' | 'admin' | 'library' | 'sport' | 'green';
	lat: number;
	lon: number;
	approximate: boolean;
	faculty: string | null;
	x: number;
	y: number;
}

export interface CampusContent {
	map: { width: number; height: number; street: string; river: string };
	pois: Poi[];
	sources: Source[];
	verifiedAt: string;
}

export interface UniversityContent {
	name: string;
	nameEn: string;
	shortName: string;
	founded: string;
	address: string;
	email: string;
	website: string;
	intro: string;
	facts: { label: string; value: string; source: string }[];
	history: { year: number; text: string }[];
	didYouKnow: { text: string; source: string }[];
	authorities: { term: string; rector: string; prorectors: string[]; source: string };
	units: string[];
	contacts: { name: string; phone: string }[];
	sources: Source[];
	verifiedAt: string;
}

export interface StudentLifeContent {
	sections: {
		id: string;
		title: string;
		items: { name: string; text: string; url?: string }[];
		note?: string;
	}[];
	sources: Source[];
	verifiedAt: string;
}

export interface RecruitmentContent {
	intro: string;
	steps: { title: string; text: string }[];
	links: { title: string; url: string }[];
	stats2026: {
		candidates: string;
		qualified: string;
		programs: number;
		mostCompetitive: { name: string; perPlace: number }[];
		mostApplications: { name: string; count: number }[];
		source: string;
	};
	sources: Source[];
	verifiedAt: string;
}

export interface DemoStationContent {
	slug: string;
	name: string;
	kind: 'workshop' | 'presentation' | 'show' | 'place';
	faculty?: string;
	poi: string;
	description: string;
	weights: StatVector;
	team?: { mode: boolean; minMembers: number };
	sessions: { start: string; end: string; room: string }[];
}

export interface DemoEventContent {
	slug: string;
	name: string;
	description: string;
	timezone: string;
	day: { start: string; end: string };
	minScansForCard: number;
	stations: DemoStationContent[];
}
