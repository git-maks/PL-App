import { describe, expect, it } from 'vitest';
import { normalize, search } from './search';

describe('knowledge-base search', () => {
	it('ignores Polish diacritics and case', () => {
		expect(normalize('Łódź Żółw')).toBe('lodz zolw');
		expect(search('cyberbezpieczenstwo')[0]?.slug).toBe('cyberbezpieczenstwo');
	});
	it('matches prefixes across fields and ranks titles first', () => {
		const r = search('robot');
		expect(r.length).toBeGreaterThan(1);
		expect(r[0]?.title.toLowerCase()).toContain('robot');
	});
	it('requires every token to match', () => {
		expect(search('energetyka sztuczna').map((d) => d.slug)).toContain('sztuczna-inteligencja-w-energetyce');
		expect(search('zzzz')).toEqual([]);
	});
});
