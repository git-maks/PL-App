/*
 * Character card rendered on a <canvas> in the browser (spec §8.6): 1080×1350 (4:5, social feeds).
 * Nothing is uploaded — the PNG is shared with the Web Share API or downloaded.
 */
import { STAT_CODES, type Stats } from '#lib/game/types';
import { statByCode } from '#lib/content/archetypes';
import { identicon } from '#lib/avatar/pixelate';

export interface CardData {
	nickname: string;
	avatarUrl: string | null;
	seed: string;
	title: string;
	description: string;
	level: number;
	xp: number;
	stats: Stats;
	archetypes: { code: keyof Stats; percent: number }[];
	programs: string[];
	eventName: string;
	date: string;
}

const W = 1080;
const H = 1350;

function loadImage(src: string): Promise<HTMLImageElement> {
	return new Promise((resolve, reject) => {
		const img = new Image();
		img.onload = () => resolve(img);
		img.onerror = reject;
		img.src = src;
	});
}

/** Draws word-wrapped text; returns the y below the last line. Truncates with "…" after maxLines. */
function wrap(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, maxWidth: number, lineHeight: number, maxLines = 3) {
	const lines: string[] = [];
	let line = '';
	for (const w of text.split(' ')) {
		const test = line ? line + ' ' + w : w;
		if (ctx.measureText(test).width > maxWidth && line) {
			lines.push(line);
			line = w;
		} else line = test;
	}
	if (line) lines.push(line);
	if (lines.length > maxLines) {
		lines.length = maxLines;
		let last = lines[maxLines - 1]!;
		while (last.length > 1 && ctx.measureText(last + '…').width > maxWidth) last = last.slice(0, -1);
		lines[maxLines - 1] = last + '…';
	}
	for (const l of lines) {
		ctx.fillText(l, x, y);
		y += lineHeight;
	}
	return y;
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
	ctx.beginPath();
	ctx.roundRect(x, y, w, h, r);
}

export async function renderCard(d: CardData): Promise<HTMLCanvasElement> {
	await Promise.allSettled([document.fonts.load('500 48px "Pixelify Sans"'), document.fonts.ready]);
	const canvas = document.createElement('canvas');
	canvas.width = W;
	canvas.height = H;
	const ctx = canvas.getContext('2d')!;
	const PIXEL = '"Pixelify Sans", system-ui, sans-serif';
	const PLAIN = 'Roboto, system-ui, -apple-system, "Segoe UI", sans-serif';

	// Background
	const bg = ctx.createLinearGradient(0, 0, W, H);
	bg.addColorStop(0, '#0f2f5c');
	bg.addColorStop(1, '#1f5aa6');
	ctx.fillStyle = bg;
	ctx.fillRect(0, 0, W, H);
	ctx.fillStyle = 'rgba(255,255,255,0.05)';
	for (let y = 0; y < H; y += 24) for (let x = (y / 24) % 2 ? 12 : 0; x < W; x += 24) ctx.fillRect(x, y, 4, 4);

	// Frame
	const main = d.archetypes[0] ? statByCode[d.archetypes[0].code].color : '#f5bf48';
	ctx.strokeStyle = '#f5bf48';
	ctx.lineWidth = 10;
	roundRect(ctx, 30, 30, W - 60, H - 60, 36);
	ctx.stroke();

	// Header
	ctx.fillStyle = '#f5bf48';
	ctx.font = `500 34px ${PIXEL}`;
	ctx.fillText('UNIQUEST · KARTA POSTACI', 80, 112);
	ctx.fillStyle = 'rgba(255,255,255,0.75)';
	ctx.font = `400 28px ${PLAIN}`;
	ctx.fillText('Politechnika Lubelska', 80, 152);

	// Avatar
	const ax = 80, ay = 200, as = 300;
	ctx.fillStyle = '#ffffff';
	roundRect(ctx, ax - 10, ay - 10, as + 20, as + 20, 40);
	ctx.fill();
	ctx.fillStyle = main;
	roundRect(ctx, ax - 4, ay - 4, as + 8, as + 8, 34);
	ctx.fill();
	ctx.save();
	roundRect(ctx, ax, ay, as, as, 30);
	ctx.clip();
	ctx.imageSmoothingEnabled = false;
	if (d.avatarUrl) {
		try {
			ctx.drawImage(await loadImage(d.avatarUrl), ax, ay, as, as);
		} catch {
			/* fall back to identicon */
		}
	} else {
		const id = identicon(d.seed);
		ctx.fillStyle = '#d6e3ff';
		ctx.fillRect(ax, ay, as, as);
		ctx.fillStyle = id.color;
		const c = as / 10;
		id.cells.forEach((on, i) => on && ctx.fillRect(ax + c + (i % 8) * c, ay + c + Math.floor(i / 8) * c, c, c));
	}
	ctx.restore();

	// Name, title, level
	const tx = 430;
	ctx.fillStyle = '#ffffff';
	ctx.font = `700 52px ${PLAIN}`;
	wrap(ctx, d.nickname, tx, 260, W - tx - 80, 58, 2);
	ctx.fillStyle = '#f5bf48';
	ctx.font = `500 58px ${PIXEL}`;
	const ty = wrap(ctx, d.title, tx, 350, W - tx - 80, 62, 2);
	ctx.fillStyle = 'rgba(255,255,255,0.85)';
	ctx.font = `500 34px ${PIXEL}`;
	ctx.fillText(`LVL ${d.level} · ${Math.round(d.xp)} XP`, tx, Math.max(ty + 8, 470));

	// Archetype split bar
	let y = 560;
	ctx.fillStyle = '#ffffff';
	ctx.font = `600 34px ${PLAIN}`;
	ctx.fillText('Twoja ścieżka', 80, y);
	y += 30;
	let bx = 80;
	const bw = W - 160;
	for (const a of d.archetypes) {
		const w = (bw * a.percent) / 100;
		ctx.fillStyle = statByCode[a.code].color;
		ctx.fillRect(bx, y, w, 48);
		bx += w;
	}
	y += 90;
	ctx.font = `500 32px ${PLAIN}`;
	let lx = 80;
	for (const a of d.archetypes) {
		const label = `${a.percent}% ${statByCode[a.code].archetype}`;
		ctx.fillStyle = statByCode[a.code].color;
		ctx.fillRect(lx, y - 24, 24, 24);
		ctx.fillStyle = '#ffffff';
		ctx.fillText(label, lx + 36, y);
		lx += ctx.measureText(label).width + 80;
	}

	// Radar
	const cx = 300, cy = 960, r = 190;
	const max = Math.max(1, ...STAT_CODES.map((c) => d.stats[c]));
	const pt = (i: number, k: number) => {
		const a = -Math.PI / 2 + (i * 2 * Math.PI) / STAT_CODES.length;
		return [cx + Math.cos(a) * r * k, cy + Math.sin(a) * r * k] as const;
	};
	ctx.strokeStyle = 'rgba(255,255,255,0.25)';
	ctx.lineWidth = 2;
	for (const k of [0.33, 0.66, 1]) {
		ctx.beginPath();
		STAT_CODES.forEach((_, i) => (i ? ctx.lineTo(...pt(i, k)) : ctx.moveTo(...pt(i, k))));
		ctx.closePath();
		ctx.stroke();
	}
	ctx.beginPath();
	STAT_CODES.forEach((c, i) => {
		const p = pt(i, Math.max(0.05, d.stats[c] / max));
		if (i) ctx.lineTo(...p);
		else ctx.moveTo(...p);
	});
	ctx.closePath();
	ctx.fillStyle = 'rgba(245,191,72,0.45)';
	ctx.fill();
	ctx.strokeStyle = '#f5bf48';
	ctx.lineWidth = 5;
	ctx.stroke();
	ctx.font = `600 24px ${PLAIN}`;
	ctx.textAlign = 'center';
	STAT_CODES.forEach((c, i) => {
		const [x, yy] = pt(i, 1.22);
		ctx.fillStyle = '#ffffff';
		ctx.fillText(statByCode[c].name, x, yy + 8);
	});
	ctx.textAlign = 'left';

	// Recommendations
	let ry = 790;
	const rx = 580;
	ctx.fillStyle = '#ffffff';
	ctx.font = `600 32px ${PLAIN}`;
	ctx.fillText('Pasujące kierunki', rx, ry);
	ry += 56;
	ctx.font = `400 30px ${PLAIN}`;
	for (const [i, p] of d.programs.slice(0, 3).entries()) {
		ctx.fillStyle = '#f5bf48';
		ctx.font = `500 32px ${PIXEL}`;
		ctx.fillText(`${i + 1}.`, rx, ry);
		ctx.fillStyle = '#ffffff';
		ctx.font = `400 30px ${PLAIN}`;
		ry = wrap(ctx, p, rx + 44, ry, W - rx - 120, 38, 2) + 18;
	}

	// Footer
	ctx.fillStyle = 'rgba(255,255,255,0.7)';
	ctx.font = `400 26px ${PLAIN}`;
	ctx.fillText(`${d.eventName} · ${d.date}`, 80, H - 80);
	return canvas;
}

export function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
	return new Promise((resolve, reject) => canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('toBlob failed'))), 'image/png'));
}

/** Web Share API with files where supported; otherwise a regular download. */
export async function shareOrDownload(blob: Blob, filename: string, text: string): Promise<'shared' | 'downloaded'> {
	const file = new File([blob], filename, { type: 'image/png' });
	if (navigator.canShare?.({ files: [file] })) {
		try {
			await navigator.share({ files: [file], title: 'Moja karta postaci UniQuest', text });
			return 'shared';
		} catch (e) {
			if ((e as DOMException).name === 'AbortError') return 'shared';
		}
	}
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	a.click();
	setTimeout(() => URL.revokeObjectURL(url), 5000);
	return 'downloaded';
}
