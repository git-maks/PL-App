import { describe, expect, it } from 'vitest';
import { PALETTE, identicon, pixelatePixels } from './pixelate';

function solid(size: number, rgb: [number, number, number]) {
	const d = new Uint8ClampedArray(size * size * 4);
	for (let i = 0; i < size * size; i++) d.set([...rgb, 255], i * 4);
	return d;
}

describe('pixelate', () => {
	it('maps every pixel onto the fixed palette', () => {
		const size = 8;
		const d = new Uint8ClampedArray(size * size * 4);
		for (let i = 0; i < size * size; i++) d.set([(i * 37) % 256, (i * 91) % 256, (i * 13) % 256, 255], i * 4);
		const out = pixelatePixels(d, size, { dither: true });
		const keys = new Set(PALETTE.map((p) => p.join(',')));
		for (let i = 0; i < size * size; i++) expect(keys.has([out[i * 4], out[i * 4 + 1], out[i * 4 + 2]].join(','))).toBe(true);
	});

	it('separates a centred subject from a uniform background and outlines it', () => {
		const size = 16;
		const d = solid(size, [240, 240, 240]);
		for (let y = 5; y < 11; y++) for (let x = 5; x < 11; x++) d.set([200, 40, 40, 255], (y * size + x) * 4);
		const out = pixelatePixels(d, size, { removeBackground: true, background: [10, 20, 30] });
		const at = (x: number, y: number) => [out[(y * size + x) * 4], out[(y * size + x) * 4 + 1], out[(y * size + x) * 4 + 2]];
		expect(at(0, 0)).toEqual([10, 20, 30]);
		expect(at(4, 7)).toEqual([29, 43, 83]); // outline next to the subject
		const subject = at(7, 7) as number[];
		expect(subject).not.toEqual([10, 20, 30]);
		expect(subject[0]!).toBeGreaterThan(subject[1]! + 80); // still red, quantised to the palette
	});

	it('identicons are deterministic and mirror-symmetric', () => {
		const a = identicon('user-1');
		expect(identicon('user-1')).toEqual(a);
		for (let y = 0; y < 8; y++) for (let x = 0; x < 4; x++) expect(a.cells[y * 8 + x]).toBe(a.cells[y * 8 + 7 - x]);
	});
});
