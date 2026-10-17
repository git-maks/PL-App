/*
 * Client-side full-text search over the knowledge base (~150 documents, spec §9): tokens are
 * lower-cased and stripped of Polish diacritics; every query token must prefix-match a document
 * token (after light stemming). Title hits rank higher. No library needed at this size.
 */
import { clubs, faculties, programs, facultyByCode } from './index';

export interface SearchDoc {
	type: 'faculty' | 'program' | 'club';
	slug: string;
	title: string;
	subtitle: string;
	path: string;
	titleTokens: string[];
	tokens: Set<string>;
}

export function normalize(text: string): string {
	return text
		.toLowerCase()
		.replaceAll('ł', 'l')
		.normalize('NFD')
		.replace(/[̀-ͯ]/g, '');
}

export function tokenize(text: string): string[] {
	return normalize(text)
		.split(/[^a-z0-9]+/)
		.filter((t) => t.length > 1);
}

function doc(type: SearchDoc['type'], slug: string, title: string, subtitle: string, path: string, body: string): SearchDoc {
	const titleTokens = tokenize(title);
	return { type, slug, title, subtitle, path, titleTokens, tokens: new Set([...titleTokens, ...tokenize(subtitle), ...tokenize(body)]) };
}

let index: SearchDoc[] | null = null;
function getIndex(): SearchDoc[] {
	return (index ??= [
		...faculties.map((f) => doc('faculty', f.slug, f.name, f.address, `/explore/wydzialy/${f.slug}`, `${f.code} ${f.description}`)),
		...programs.map((p) =>
			doc('program', p.slug, p.name, facultyByCode[p.faculty]?.name ?? p.faculty, `/explore/kierunki/${p.slug}`, `${p.description} ${p.careers.join(' ')} ${p.maturaSubjects.join(' ')} ${p.faculty}`)
		),
		...clubs.map((c) => doc('club', c.slug, c.name, facultyByCode[c.faculty]?.name ?? c.faculty, `/explore/kola/${c.slug}`, `${c.tags.join(' ')} ${c.faculty}`))
	]);
}

/** Very light stemming for Polish inflection: "energetyka" should also find "energetyce". */
function stem(t: string): string {
	return t.length >= 6 ? t.slice(0, t.length - 2) : t;
}

export function search(query: string, limit = 20): SearchDoc[] {
	const q = tokenize(query).map(stem);
	if (q.length === 0) return [];
	const scored: { d: SearchDoc; score: number }[] = [];
	for (const d of getIndex()) {
		let score = 0;
		let all = true;
		for (const t of q) {
			let hit = false;
			for (const tok of d.tokens) {
				if (tok.startsWith(t)) {
					hit = true;
					break;
				}
			}
			if (!hit) {
				all = false;
				break;
			}
			score += d.titleTokens.some((tt) => tt.startsWith(t)) ? 3 : 1;
		}
		if (all) scored.push({ d, score: score + (d.type === 'program' ? 0.5 : d.type === 'faculty' ? 0.3 : 0) });
	}
	return scored.sort((a, b) => b.score - a.score || a.d.title.localeCompare(b.d.title, 'pl')).slice(0, limit).map((s) => s.d);
}
