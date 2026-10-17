export type CameraError = 'denied' | 'not-found' | 'insecure' | 'unsupported' | 'other';

export interface Camera {
	stream: MediaStream;
	track: MediaStreamTrack;
	torchSupported: boolean;
	setTorch(on: boolean): Promise<void>;
	stop(): void;
}

export async function openCamera(video: HTMLVideoElement): Promise<Camera> {
	if (!window.isSecureContext) throw 'insecure' satisfies CameraError;
	if (!navigator.mediaDevices?.getUserMedia) throw 'unsupported' satisfies CameraError;
	let stream: MediaStream;
	try {
		stream = await navigator.mediaDevices.getUserMedia({
			audio: false,
			video: { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } }
		});
	} catch (e) {
		const name = (e as DOMException)?.name;
		throw (name === 'NotAllowedError' || name === 'SecurityError' ? 'denied' : name === 'NotFoundError' || name === 'OverconstrainedError' ? 'not-found' : 'other') satisfies CameraError;
	}
	video.srcObject = stream;
	video.setAttribute('playsinline', '');
	video.muted = true;
	await video.play().catch(() => {});
	const track = stream.getVideoTracks()[0]!;
	const caps = (track.getCapabilities?.() ?? {}) as MediaTrackCapabilities & { torch?: boolean };
	return {
		stream,
		track,
		torchSupported: !!caps.torch,
		setTorch: (on) => track.applyConstraints({ advanced: [{ torch: on } as MediaTrackConstraintSet] }),
		stop: () => stream.getTracks().forEach((t) => t.stop())
	};
}

/**
 * Runs `onFrame` for video frames at most `fps` times per second until stopped.
 * Uses requestVideoFrameCallback where supported (only fires on new frames).
 */
export function frameLoop(video: HTMLVideoElement, fps: number, onFrame: () => Promise<boolean | void>): () => void {
	let stopped = false;
	let busy = false;
	let last = 0;
	const interval = 1000 / fps;
	const v = video as HTMLVideoElement & { requestVideoFrameCallback?: (cb: () => void) => number };
	const schedule = () => {
		if (stopped) return;
		if (v.requestVideoFrameCallback) v.requestVideoFrameCallback(tick);
		else requestAnimationFrame(tick);
	};
	const tick = async () => {
		if (stopped) return;
		const now = performance.now();
		if (!busy && now - last >= interval && video.readyState >= 2) {
			busy = true;
			last = now;
			try {
				if ((await onFrame()) === true) stopped = true;
			} finally {
				busy = false;
			}
		}
		schedule();
	};
	schedule();
	return () => (stopped = true);
}
