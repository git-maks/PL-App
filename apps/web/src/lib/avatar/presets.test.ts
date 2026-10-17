import { describe, expect, it } from 'vitest';
import { AVATAR_PRESETS, presetPixels } from './presets';

describe('avatar presets', () => {
	it('offers nine distinct colourways', () => {
		expect(AVATAR_PRESETS).toHaveLength(9);
		expect(new Set(AVATAR_PRESETS.map((p) => p.id)).size).toBe(9);
		expect(new Set(AVATAR_PRESETS.map((p) => p.skin)).size).toBe(9);
		expect(new Set(AVATAR_PRESETS.map((p) => p.outfit)).size).toBe(9);
	});

	it('every sprite is a full 16×16 grid that is left–right symmetric', () => {
		for (const p of AVATAR_PRESETS) {
			const px = presetPixels(p);
			expect(px).toHaveLength(256);
			for (let y = 0; y < 16; y++)
				for (let x = 0; x < 8; x++) expect(px[y * 16 + x], `${p.id} ${x},${y}`).toBe(px[y * 16 + 15 - x]);
		}
	});
});
