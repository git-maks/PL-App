// Generates a Y4M video showing a station QR code, used as Chromium's fake camera.
import { mkdirSync, writeFileSync } from 'node:fs';
import { qrcodegen } from '../src/lib/qr/vendor/qrcodegen.js';
import { staticPayload } from './demo';

export default function globalSetup() {
	const W = 640;
	const H = 480;
	const qr = qrcodegen.QrCode.encodeText(staticPayload('weii-roboty-iot'), qrcodegen.QrCode.Ecc.MEDIUM);
	const scale = Math.floor(380 / (qr.size + 8));
	const size = (qr.size + 8) * scale;
	const x0 = Math.floor((W - size) / 2);
	const y0 = Math.floor((H - size) / 2);
	const y = Buffer.alloc(W * H, 200);
	for (let py = 0; py < size; py++) {
		for (let px = 0; px < size; px++) {
			const mx = Math.floor(px / scale) - 4;
			const my = Math.floor(py / scale) - 4;
			const dark = mx >= 0 && my >= 0 && mx < qr.size && my < qr.size && qr.getModule(mx, my);
			y[(y0 + py) * W + x0 + px] = dark ? 16 : 235;
		}
	}
	const uv = Buffer.alloc((W / 2) * (H / 2), 128);
	const frame = Buffer.concat([Buffer.from('FRAME\n'), y, uv, uv]);
	const header = Buffer.from(`YUV4MPEG2 W${W} H${H} F10:1 Ip A1:1 C420jpeg\n`);
	mkdirSync('e2e/.generated', { recursive: true });
	writeFileSync('e2e/.generated/station.y4m', Buffer.concat([header, ...Array(5).fill(frame)]));
}
