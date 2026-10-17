import { expect, it } from 'vitest';
import { describePath } from './describe';

it('builds the Polish path sentence', () => {
	expect(describePath([{ code: 'STR', percent: 70 }, { code: 'INT', percent: 30 }])).toBe('Twoja ścieżka to w 70% Konstruktor i w 30% Analityk.');
	expect(describePath([{ code: 'MAN', percent: 100 }])).toBe('Twoja ścieżka to w 100% Technomag.');
	expect(describePath([])).toBe('Twoja ścieżka dopiero się kształtuje.');
});
