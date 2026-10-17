/*
 * QR detection: the native BarcodeDetector where available (Chrome/Android), otherwise the
 * standards-compliant ponyfill (zxing-wasm), loaded lazily and served from our own origin –
 * never from a CDN – so it works offline and under a strict CSP.
 */
export interface QrDetector {
	native: boolean;
	detect(source: ImageBitmapSource): Promise<string[]>;
}

interface DetectedBarcodeLike {
	rawValue: string;
}
interface BarcodeDetectorLike {
	detect(source: ImageBitmapSource): Promise<DetectedBarcodeLike[]>;
}
interface BarcodeDetectorCtor {
	new (opts?: { formats: string[] }): BarcodeDetectorLike;
	getSupportedFormats(): Promise<string[]>;
}

let cached: Promise<QrDetector> | null = null;

export function loadDetector(): Promise<QrDetector> {
	return (cached ??= create());
}

async function create(): Promise<QrDetector> {
	const Native = (globalThis as { BarcodeDetector?: BarcodeDetectorCtor }).BarcodeDetector;
	if (Native) {
		try {
			if ((await Native.getSupportedFormats()).includes('qr_code')) {
				const d = new Native({ formats: ['qr_code'] });
				return { native: true, detect: async (s) => (await d.detect(s)).map((b) => b.rawValue) };
			}
		} catch {
			/* fall through to the ponyfill */
		}
	}
	const [{ BarcodeDetector, prepareZXingModule }, wasm] = await Promise.all([
		import('barcode-detector/ponyfill'),
		import('zxing-wasm/reader/zxing_reader.wasm?url')
	]);
	prepareZXingModule({
		overrides: { locateFile: (path: string, prefix: string) => (path.endsWith('.wasm') ? wasm.default : prefix + path) },
		fireImmediately: true
	});
	const d = new BarcodeDetector({ formats: ['qr_code'] });
	return { native: false, detect: async (s) => (await d.detect(s)).map((b) => b.rawValue) };
}

/** Warm up (download + compile) the decoder in the background, e.g. after onboarding. */
export function prefetchDetector() {
	if (typeof requestIdleCallback === 'function') requestIdleCallback(() => void loadDetector().catch(() => {}));
	else setTimeout(() => void loadDetector().catch(() => {}), 2000);
}
