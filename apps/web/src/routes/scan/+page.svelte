<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { parsePayload } from '#lib/qr/payload';
	import { loadDetector } from '#lib/scanner/detector';
	import { frameLoop, openCamera, type Camera, type CameraError } from '#lib/scanner/camera';
	import IconButton from '#lib/components/IconButton.svelte';
	import Button from '#lib/components/Button.svelte';
	import Dialog from '#lib/components/Dialog.svelte';
	import TextField from '#lib/components/TextField.svelte';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import Icon from '#lib/components/Icon.svelte';

	type Status = 'starting' | 'scanning' | 'working' | 'error';
	let video: HTMLVideoElement | undefined = $state();
	let camera: Camera | null = null;
	let stopLoop: (() => void) | null = null;
	let status = $state<Status>('starting');
	let cameraError = $state<CameraError | null>(null);
	let message = $state<string | null>(null);
	let torch = $state(false);
	let torchSupported = $state(false);
	let manualOpen = $state(false);
	let manualCode = $state('');
	let manualError = $state<string | null>(null);
	let lastRejected = '';

	const CAMERA_HELP: Record<CameraError, string> = {
		denied: 'Brak zgody na użycie aparatu. Włącz ją w ustawieniach przeglądarki (ikona kłódki obok adresu) albo wpisz kod ręcznie.',
		'not-found': 'Nie znaleźliśmy aparatu w tym urządzeniu. Wpisz kod ręcznie.',
		insecure: 'Aparat działa tylko na bezpiecznym połączeniu (HTTPS).',
		unsupported: 'Ta przeglądarka nie obsługuje aparatu. Wpisz kod ręcznie.',
		other: 'Nie udało się uruchomić aparatu. Spróbuj ponownie albo wpisz kod ręcznie.'
	};

	async function start() {
		if (!video) return;
		status = 'starting';
		cameraError = null;
		message = null;
		try {
			const [cam, detector] = await Promise.all([openCamera(video), loadDetector()]);
			camera = cam;
			torchSupported = cam.torchSupported;
			status = 'scanning';
			stopLoop = frameLoop(video, 8, async () => {
				if (status !== 'scanning' || !video) return;
				const codes = await detector.detect(video).catch(() => []);
				for (const raw of codes) {
					if (raw === lastRejected) continue;
					if (await handle(raw)) return true;
				}
			});
		} catch (e) {
			status = 'error';
			cameraError = typeof e === 'string' ? (e as CameraError) : 'other';
		}
	}

	function stop() {
		stopLoop?.();
		stopLoop = null;
		camera?.stop();
		camera = null;
	}

	/** Returns true when the scan was consumed (stop scanning). */
	async function handle(raw: string): Promise<boolean> {
		const parsed = parsePayload(raw);
		if (!parsed) {
			lastRejected = raw;
			message = 'To nie jest kod UniQuest. Szukaj kodu wyświetlanego na stacji.';
			return false;
		}
		navigator.vibrate?.(40);
		if (parsed.kind === 'team') {
			stop();
			await goto(href('/t/' + parsed.code));
			return true;
		}
		status = 'working';
		const outcome = await app.submitScan(raw);
		if (outcome.kind === 'ok') {
			stop();
			const { scan, alreadyScanned } = outcome.result;
			if (alreadyScanned && scan.rating !== null) {
				await goto(href('/scan/rate?id=' + scan.id + '&again=1'), { replaceState: true });
			} else {
				await goto(href('/scan/rate?id=' + scan.id), { replaceState: true });
			}
			return true;
		}
		if (outcome.kind === 'queued') {
			stop();
			await goto(href('/scan/rate?queued=' + outcome.item.id), { replaceState: true });
			return true;
		}
		lastRejected = raw;
		message = errorMessage(outcome.error);
		status = 'scanning';
		return false;
	}

	async function submitManual(e: SubmitEvent) {
		e.preventDefault();
		manualError = null;
		const outcome = await app.submitScan(manualCode);
		if (outcome.kind === 'error') {
			manualError = errorMessage(outcome.error);
			return;
		}
		manualOpen = false;
		stop();
		if (outcome.kind === 'ok') await goto(href('/scan/rate?id=' + outcome.result.scan.id), { replaceState: true });
		else await goto(href('/scan/rate?queued=' + outcome.item.id), { replaceState: true });
	}

	async function toggleTorch() {
		if (!camera) return;
		torch = !torch;
		await camera.setTorch(torch).catch(() => (torch = false));
	}

	onMount(() => {
		const onVisibility = () => {
			if (document.visibilityState === 'hidden') stop();
			else if (!camera && status !== 'working') void start();
		};
		document.addEventListener('visibilitychange', onVisibility);
		return () => document.removeEventListener('visibilitychange', onVisibility);
	});
	$effect(() => {
		if (video && app.me && !camera) void start();
	});
	onDestroy(stop);
</script>

<svelte:head><title>Skanuj · UniQuest</title></svelte:head>

<AuthGate reason="Aby zbierać punkty za stacje, potrzebujesz bohatera.">
	<div class="scanner">
		<!-- svelte-ignore a11y_media_has_caption -->
		<video bind:this={video} playsinline muted aria-label="Podgląd z aparatu"></video>

		<div class="shade" aria-hidden="true">
			<div class="finder" class:active={status === 'scanning'} class:working={status === 'working'}>
				<span class="c tl"></span><span class="c tr"></span><span class="c bl"></span><span class="c br"></span>
				<span class="line"></span>
			</div>
		</div>

		<div class="top">
			<IconButton icon="close" label="Zamknij skaner" variant="tonal" onclick={() => (history.length > 1 ? history.back() : goto(href('/hero')))} />
			<span class="spacer"></span>
			{#if torchSupported}
				<IconButton icon={torch ? 'flashlight_off' : 'flashlight_on'} label={torch ? 'Wyłącz latarkę' : 'Włącz latarkę'} variant="tonal" selected={torch} onclick={toggleTorch} />
			{/if}
		</div>

		<div class="bottom">
			{#if status === 'error' && cameraError}
				<div class="panel">
					<Icon name="photo_camera" />
					<p class="body-medium">{CAMERA_HELP[cameraError]}</p>
					<div class="row wrap">
						<Button variant="tonal" icon="refresh" onclick={start}>Spróbuj ponownie</Button>
					</div>
				</div>
			{:else}
				<p class="status body-medium" role="status" aria-live="polite">
					{#if status === 'starting'}Uruchamiam aparat…
					{:else if status === 'working'}Sprawdzam kod…
					{:else if message}{message}
					{:else}Skieruj aparat na kod QR wyświetlany na stacji{/if}
				</p>
			{/if}
			<Button variant="elevated" icon="keyboard" onclick={() => (manualOpen = true)}>Wpisz kod ręcznie</Button>
		</div>
	</div>

	<Dialog bind:open={manualOpen} title="Wpisz kod stacji" sheet>
		<form class="stack" onsubmit={submitManual}>
			<p>Pod kodem QR na ekranie stacji jest 6-znakowy kod (np. K7PQ2M).</p>
			<TextField label="Kod stacji" bind:value={manualCode} maxlength={8} autocomplete="off" error={manualError} autofocus />
			<Button type="submit" full icon="check" disabled={manualCode.trim().length < 6}>Zatwierdź</Button>
		</form>
	</Dialog>
</AuthGate>

<style>
	.scanner {
		position: fixed;
		inset: 0;
		background: #000;
		color: #fff;
		overflow: hidden;
	}
	video {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.shade {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
	}
	.finder {
		position: relative;
		width: min(70vw, 300px);
		aspect-ratio: 1;
		border-radius: 24px;
		box-shadow: 0 0 0 100vmax rgb(0 0 0 / 0.45);
		transition: transform var(--md-sys-motion-duration-medium2) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	.finder.working {
		transform: scale(0.94);
	}
	.c {
		position: absolute;
		width: 40px;
		height: 40px;
		border: 4px solid #fff;
	}
	.tl { top: -2px; left: -2px; border-right: 0; border-bottom: 0; border-top-left-radius: 24px; }
	.tr { top: -2px; right: -2px; border-left: 0; border-bottom: 0; border-top-right-radius: 24px; }
	.bl { bottom: -2px; left: -2px; border-right: 0; border-top: 0; border-bottom-left-radius: 24px; }
	.br { bottom: -2px; right: -2px; border-left: 0; border-top: 0; border-bottom-right-radius: 24px; }
	.line {
		position: absolute;
		left: 12px;
		right: 12px;
		top: 12px;
		height: 2px;
		border-radius: 1px;
		background: var(--md-sys-color-inverse-primary);
		box-shadow: 0 0 12px var(--md-sys-color-inverse-primary);
		opacity: 0;
	}
	.active .line {
		opacity: 0.9;
		animation: sweep 2.2s var(--md-sys-motion-easing-standard) infinite alternate;
	}
	@keyframes sweep {
		to {
			transform: translateY(calc(min(70vw, 300px) - 26px));
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.active .line {
			animation: none;
			top: 50%;
		}
	}
	.top {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		display: flex;
		padding: calc(var(--uq-safe-top) + 12px) 12px 0;
	}
	.spacer {
		flex: 1;
	}
	.bottom {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		padding: 16px 16px calc(var(--uq-safe-bottom) + 24px);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
	}
	.status {
		text-align: center;
		max-width: 320px;
		padding: 8px 16px;
		border-radius: var(--md-sys-shape-corner-full);
		background: rgb(0 0 0 / 0.55);
	}
	.panel {
		width: 100%;
		max-width: 420px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 20px;
		border-radius: var(--md-sys-shape-corner-extra-large);
		background: var(--md-sys-color-surface-container-high);
		color: var(--md-sys-color-on-surface);
	}
</style>
