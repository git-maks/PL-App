/*
 * Selfie → pixel-art avatar, entirely on the device (spec §10). Only the 48×48 result ever
 * leaves the phone; the photo itself is discarded.
 *
 * Pipeline: square crop → downscale → contrast/saturation → background separation (border colour
 * heuristic) → quantisation to a fixed palette (optional Floyd–Steinberg) → 1-px outline.
 */

export const AVATAR_SIZE = 48;

/** Fixed palette: PICO-8 (CC0) plus skin and hair tones, so every avatar shares one style. */
export const PALETTE: [number, number, number][] = [
	[0, 0, 0], [29, 43, 83], [126, 37, 83], [0, 135, 81], [171, 82, 54], [95, 87, 79], [194, 195, 199],
	[255, 241, 232], [255, 0, 77], [255, 163, 0], [255, 236, 39], [0, 228, 54], [41, 173, 255],
	[131, 118, 156], [255, 119, 168], [255, 204, 170],
	[141, 85, 36], [198, 134, 66], [224, 172, 105], [241, 194, 125], [255, 219, 172], [59, 34, 25],
	[107, 68, 35], [46, 46, 46]
];

export interface PixelateOptions {
	dither?: boolean;
	removeBackground?: boolean;
	/** Background colour used where the photo background was removed. */
	background?: [number, number, number];
	/** Horizontal/vertical crop offset in [-1, 1] relative to the free space. */
	offsetX?: number;
	offsetY?: number;
	zoom?: number;
}

const OUTLINE: [number, number, number] = [29, 43, 83];

function clamp(v: number) {
	return v < 0 ? 0 : v > 255 ? 255 : v;
}

/** Weighted RGB distance ("redmean"), a cheap approximation of perceptual difference. */
function dist(r1: number, g1: number, b1: number, r2: number, g2: number, b2: number) {
	const rm = (r1 + r2) / 2;
	const dr = r1 - r2;
	const dg = g1 - g2;
	const db = b1 - b2;
	return (2 + rm / 256) * dr * dr + 4 * dg * dg + (2 + (255 - rm) / 256) * db * db;
}

export function nearest(r: number, g: number, b: number): [number, number, number] {
	let best = PALETTE[0]!;
	let bd = Infinity;
	for (const p of PALETTE) {
		const d = dist(r, g, b, p[0], p[1], p[2]);
		if (d < bd) {
			bd = d;
			best = p;
		}
	}
	return best;
}

/** Pure pixel pipeline on RGBA data (exported for unit tests). */
export function pixelatePixels(data: Uint8ClampedArray, size: number, opts: PixelateOptions = {}): Uint8ClampedArray {
	const out = new Uint8ClampedArray(data);
	const n = size * size;

	// 1. contrast + saturation
	for (let i = 0; i < n; i++) {
		const o = i * 4;
		let r = out[o]!, g = out[o + 1]!, b = out[o + 2]!;
		const l = 0.299 * r + 0.587 * g + 0.114 * b;
		r = l + (r - l) * 1.25;
		g = l + (g - l) * 1.25;
		b = l + (b - l) * 1.25;
		out[o] = clamp((r - 128) * 1.15 + 128);
		out[o + 1] = clamp((g - 128) * 1.15 + 128);
		out[o + 2] = clamp((b - 128) * 1.15 + 128);
	}

	// 2. background mask: pixels close to the average border colour, flood-filled from the edges
	const bg = new Uint8Array(n);
	if (opts.removeBackground) {
		let br = 0, bgc = 0, bb = 0, cnt = 0;
		for (let i = 0; i < size; i++) {
			for (const idx of [i, (size - 1) * size + i, i * size, i * size + size - 1]) {
				br += out[idx * 4]!;
				bgc += out[idx * 4 + 1]!;
				bb += out[idx * 4 + 2]!;
				cnt++;
			}
		}
		br /= cnt; bgc /= cnt; bb /= cnt;
		const threshold = 2600;
		const stack: number[] = [];
		for (let i = 0; i < size; i++) stack.push(i, (size - 1) * size + i, i * size, i * size + size - 1);
		while (stack.length) {
			const idx = stack.pop()!;
			if (bg[idx]) continue;
			const o = idx * 4;
			if (dist(out[o]!, out[o + 1]!, out[o + 2]!, br, bgc, bb) > threshold) continue;
			bg[idx] = 1;
			const x = idx % size, y = (idx / size) | 0;
			if (x > 0) stack.push(idx - 1);
			if (x < size - 1) stack.push(idx + 1);
			if (y > 0) stack.push(idx - size);
			if (y < size - 1) stack.push(idx + size);
		}
		// Too little or too much removed → the heuristic failed; keep the photo background.
		const removed = bg.reduce((a, v) => a + v, 0) / n;
		if (removed < 0.08 || removed > 0.92) bg.fill(0);
	}

	// 3. quantise (with optional error diffusion)
	const err = opts.dither ? new Float32Array(n * 3) : null;
	for (let y = 0; y < size; y++) {
		for (let x = 0; x < size; x++) {
			const i = y * size + x;
			const o = i * 4;
			if (bg[i]) continue;
			const r = out[o]! + (err ? err[i * 3]! : 0);
			const g = out[o + 1]! + (err ? err[i * 3 + 1]! : 0);
			const b = out[o + 2]! + (err ? err[i * 3 + 2]! : 0);
			const p = nearest(clamp(r), clamp(g), clamp(b));
			out[o] = p[0];
			out[o + 1] = p[1];
			out[o + 2] = p[2];
			out[o + 3] = 255;
			if (err) {
				const er = r - p[0], eg = g - p[1], eb = b - p[2];
				const spread = (dx: number, dy: number, f: number) => {
					const nx = x + dx, ny = y + dy;
					if (nx < 0 || nx >= size || ny >= size) return;
					const j = (ny * size + nx) * 3;
					err[j] = err[j]! + er * f;
					err[j + 1] = err[j + 1]! + eg * f;
					err[j + 2] = err[j + 2]! + eb * f;
				};
				spread(1, 0, 7 / 16);
				spread(-1, 1, 3 / 16);
				spread(0, 1, 5 / 16);
				spread(1, 1, 1 / 16);
			}
		}
	}

	// 4. background fill + outline around the subject
	const fill = opts.background ?? [214, 227, 255];
	for (let i = 0; i < n; i++) {
		if (!bg[i]) continue;
		const x = i % size, y = (i / size) | 0;
		const touchesSubject =
			(x > 0 && !bg[i - 1]) || (x < size - 1 && !bg[i + 1]) || (y > 0 && !bg[i - size]) || (y < size - 1 && !bg[i + size]);
		const c = touchesSubject ? OUTLINE : fill;
		const o = i * 4;
		out[o] = c[0];
		out[o + 1] = c[1];
		out[o + 2] = c[2];
		out[o + 3] = 255;
	}
	return out;
}

/** Loads an image file (respecting EXIF orientation) and returns a 48×48 PNG data URL. */
export async function pixelateFile(file: Blob, opts: PixelateOptions = {}): Promise<string> {
	const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
	try {
		return pixelateBitmap(bitmap, opts);
	} finally {
		bitmap.close();
	}
}

export function pixelateBitmap(bitmap: ImageBitmap, opts: PixelateOptions = {}): string {
	const side = Math.min(bitmap.width, bitmap.height) / (opts.zoom ?? 1);
	const sx = (bitmap.width - side) / 2 + ((opts.offsetX ?? 0) * (bitmap.width - side)) / 2;
	const sy = (bitmap.height - side) / 2 + ((opts.offsetY ?? 0) * (bitmap.height - side)) / 2;
	const canvas = document.createElement('canvas');
	canvas.width = canvas.height = AVATAR_SIZE;
	const ctx = canvas.getContext('2d', { willReadFrequently: true })!;
	ctx.imageSmoothingEnabled = true;
	ctx.imageSmoothingQuality = 'high';
	ctx.drawImage(bitmap, sx, sy, side, side, 0, 0, AVATAR_SIZE, AVATAR_SIZE);
	const img = ctx.getImageData(0, 0, AVATAR_SIZE, AVATAR_SIZE);
	img.data.set(pixelatePixels(img.data, AVATAR_SIZE, opts));
	ctx.putImageData(img, 0, 0);
	return canvas.toDataURL('image/png');
}

/** Deterministic symmetric 8×8 "identicon" avatar for players without a selfie. */
export function identicon(seed: string): { cells: boolean[]; color: string } {
	let h = 2166136261;
	for (const ch of seed) h = Math.imul(h ^ ch.charCodeAt(0), 16777619);
	const cells: boolean[] = [];
	for (let y = 0; y < 8; y++) {
		const row: boolean[] = [];
		for (let x = 0; x < 4; x++) {
			h = Math.imul(h ^ (h >>> 13), 0x5bd1e995) >>> 0;
			row.push((h & 7) > 2);
		}
		cells.push(...row, ...row.slice().reverse());
	}
	const hues = [214, 262, 330, 18, 140, 42, 190];
	return { cells, color: `hsl(${hues[(h >>> 3) % hues.length]} 60% 42%)` };
}
