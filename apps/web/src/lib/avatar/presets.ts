/**
 * Ready-made pixel avatars for players who do not want to take a selfie. Three hairstyles × nine
 * colourways (skin tone, hair, outfit, background), rendered to the same 48×48 PNG as a selfie avatar,
 * so both backends store them through the existing avatar upload.
 */

type Style = 'short' | 'long' | 'curly';

// 16×16 sprites: . background, h hair, s skin, e eyes, m mouth, c outfit, k collar.
const FACE_AND_BODY = [
	'...ss' + 'e' + 'ssss' + 'e' + 'ss...',
	'...ssssssssss...',
	'...ssssmmssss...',
	'....ssssssss....',
	'......ssss......',
	'...cccckkcccc...',
	'..cccccccccccc..',
	'.cccccccccccccc.',
	'.cccccccccccccc.'
];

const SPRITES: Record<Style, string[]> = {
	short: [
		'................',
		'................',
		'.....hhhhhh.....',
		'....hhhhhhhh....',
		'...hhhhhhhhhh...',
		'...hhsssssshh...',
		'...hssssssssh...',
		...FACE_AND_BODY
	],
	long: [
		'................',
		'................',
		'.....hhhhhh.....',
		'....hhhhhhhh....',
		'...hhhhhhhhhh...',
		'..hhhsssssshhh..',
		'..hhsssssssshh..',
		'..hss' + 'e' + 'ssss' + 'e' + 'ssh..',
		'..hssssssssssh..',
		'..hssssmmssssh..',
		'..hhsssssssshh..',
		'..hhh.ssss.hhh..',
		'..hcccckkcccch..',
		'..cccccccccccc..',
		'.cccccccccccccc.',
		'.cccccccccccccc.'
	],
	curly: [
		'......hhhh......',
		'....hhhhhhhh....',
		'...hhhhhhhhhh...',
		'..hhhhhhhhhhhh..',
		'..hhhhhhhhhhhh..',
		'..hhhsssssshhh..',
		'..hhsssssssshh..',
		...FACE_AND_BODY
	]
};

export interface AvatarPreset {
	id: string;
	label: string;
	style: Style;
	bg: string;
	skin: string;
	hair: string;
	outfit: string;
	collar: string;
}

export const AVATAR_PRESETS: AvatarPreset[] = [
	{ id: 'navy', label: 'Granatowy', style: 'short', bg: '#dbe6ff', skin: '#f6d3b3', hair: '#3b2a1e', outfit: '#1f5aa6', collar: '#ffffff' },
	{ id: 'forest', label: 'Leśny', style: 'long', bg: '#dff3e0', skin: '#e8b48f', hair: '#a8432a', outfit: '#2e7d32', collar: '#c8e6c9' },
	{ id: 'sun', label: 'Słoneczny', style: 'curly', bg: '#fff3cd', skin: '#8d5524', hair: '#1b1b1b', outfit: '#f9a825', collar: '#fff8e1' },
	{ id: 'berry', label: 'Malinowy', style: 'short', bg: '#fde0ec', skin: '#c68642', hair: '#141414', outfit: '#c2185b', collar: '#f8bbd0' },
	{ id: 'violet', label: 'Fioletowy', style: 'long', bg: '#ede0f7', skin: '#f1c27d', hair: '#e2bc4f', outfit: '#6a1b9a', collar: '#e1bee7' },
	{ id: 'teal', label: 'Morski', style: 'curly', bg: '#d6f3ef', skin: '#5c3a1e', hair: '#2b1a10', outfit: '#00897b', collar: '#b2dfdb' },
	{ id: 'ember', label: 'Pomarańczowy', style: 'short', bg: '#ffe6d1', skin: '#e0ac69', hair: '#2962ff', outfit: '#ef6c00', collar: '#ffe0b2' },
	{ id: 'graphite', label: 'Grafitowy', style: 'long', bg: '#e3e8ea', skin: '#a86b3c', hair: '#3a2418', outfit: '#37474f', collar: '#cfd8dc' },
	{ id: 'candy', label: 'Różowy', style: 'curly', bg: '#dcecff', skin: '#ffdbc4', hair: '#ff6fae', outfit: '#1565c0', collar: '#bbdefb' }
];

const EYES = '#1d1b20';

function shade(hex: string, f: number): string {
	const n = parseInt(hex.slice(1), 16);
	const c = (v: number) => Math.round(Math.min(255, Math.max(0, v * f)));
	return `rgb(${c(n >> 16)}, ${c((n >> 8) & 255)}, ${c(n & 255)})`;
}

/** Colour of every pixel of the 16×16 sprite (row-major). */
export function presetPixels(p: AvatarPreset): string[] {
	const map: Record<string, string> = {
		'.': p.bg,
		h: p.hair,
		s: p.skin,
		e: EYES,
		m: shade(p.skin, 0.72),
		c: p.outfit,
		k: p.collar
	};
	return SPRITES[p.style].flatMap((row) => [...row].map((ch) => map[ch] ?? p.bg));
}

/** 48×48 PNG data URL (3× upscale, crisp pixels), ready for `uploadAvatar`. */
export function renderPreset(p: AvatarPreset): string {
	const canvas = document.createElement('canvas');
	canvas.width = canvas.height = 48;
	const ctx = canvas.getContext('2d')!;
	presetPixels(p).forEach((color, i) => {
		ctx.fillStyle = color;
		ctx.fillRect((i % 16) * 3, Math.floor(i / 16) * 3, 3, 3);
	});
	return canvas.toDataURL('image/png');
}
