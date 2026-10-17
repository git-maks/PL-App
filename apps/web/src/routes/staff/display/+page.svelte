<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/state';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import type { StationQr } from '#lib/api';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import QrCode from '#lib/qr/QrCode.svelte';

	const id = $derived(page.url.searchParams.get('id') ?? '');
	const station = $derived(app.stationById.get(id));
	let qr = $state<StationQr | null>(null);
	let error = $state<string | null>(null);
	let now = $state(Date.now());
	let timer: ReturnType<typeof setTimeout> | null = null;
	let wakeLock: { release(): Promise<void> } | null = null;
	const tick = setInterval(() => (now = Date.now()), 250);

	async function refresh() {
		if (!id) return;
		try {
			qr = await app.api.stationQr(id);
			error = null;
			const ms = new Date(qr.validUntil).getTime() - Date.now();
			timer = setTimeout(refresh, Math.max(1000, ms + 150));
		} catch (e) {
			error = errorMessage(e);
			timer = setTimeout(refresh, 5000);
		}
	}
	$effect(() => {
		if (app.me && id && !qr && !timer) void refresh();
	});

	onMount(() => {
		// Keep the projector/tablet screen awake while the code is shown.
		const nav = navigator as Navigator & { wakeLock?: { request(t: 'screen'): Promise<{ release(): Promise<void> }> } };
		nav.wakeLock?.request('screen').then((l) => (wakeLock = l)).catch(() => {});
	});
	onDestroy(() => {
		clearInterval(tick);
		if (timer) clearTimeout(timer);
		void wakeLock?.release();
	});

	const remaining = $derived(qr ? Math.max(0, (new Date(qr.validUntil).getTime() - now) / 1000) : 0);
	const ratio = $derived(qr ? remaining / qr.periodSeconds : 0);
</script>

<svelte:head><title>Kod stacji · UniQuest</title></svelte:head>

<AuthGate>
	<div class="display">
		<div class="top">
			<IconButton icon="close" label="Zamknij" href={href('/staff')} />
		</div>
		<h1 class="name">{station?.name ?? 'Stacja'}</h1>
		{#if error}
			<p class="err" role="alert">{error}</p>
		{:else if qr}
			{#if !qr.scanOpen}
				<p class="closed">Skanowanie jest zamknięte — otwórz je w panelu stacji.</p>
			{/if}
			<div class="qr" class:dim={!qr.scanOpen}>
				{#key qr.payload}<QrCode text={qr.payload} size={520} label="Kod QR stacji {station?.name}" ecc="M" />{/key}
			</div>
			<div class="meta">
				<svg class="ring" viewBox="0 0 36 36" aria-hidden="true">
					<circle cx="18" cy="18" r="16" class="track" />
					<circle cx="18" cy="18" r="16" class="prog" style="stroke-dashoffset: {100 - ratio * 100}" pathLength="100" />
				</svg>
				<div>
					<p class="code-label">Kod ręczny</p>
					<p class="code">{qr.manualCode}</p>
				</div>
				<div class="count">
					<p class="code-label">Zeskanowało</p>
					<p class="code">{qr.scansCount}</p>
				</div>
			</div>
			<p class="hint">Otwórz UniQuest → Skanuj. Kod zmienia się co {qr.periodSeconds} s.</p>
		{/if}
	</div>
</AuthGate>

<style>
	.display {
		position: fixed;
		inset: 0;
		background: #0f2f5c;
		color: #fff;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		padding: 24px;
		overflow: auto;
	}
	.top {
		position: absolute;
		top: 12px;
		left: 12px;
		color: #fff;
	}
	.top :global(a) {
		color: #fff;
	}
	.name {
		font: 600 clamp(1.25rem, 3vw, 2.25rem) / 1.2 var(--md-ref-typeface-plain);
		text-align: center;
		max-width: 900px;
	}
	.qr {
		background: #fff;
		padding: 16px;
		border-radius: 24px;
		width: min(70vmin, 560px);
		transition: opacity var(--md-sys-motion-duration-medium2) linear;
	}
	.qr :global(svg) {
		width: 100%;
		height: auto;
	}
	.dim {
		opacity: 0.25;
	}
	.closed {
		padding: 8px 16px;
		border-radius: 999px;
		background: #ba1a1a;
	}
	.meta {
		display: flex;
		align-items: center;
		gap: 32px;
	}
	.ring {
		width: 56px;
		height: 56px;
		transform: rotate(-90deg);
	}
	.track {
		fill: none;
		stroke: rgb(255 255 255 / 0.2);
		stroke-width: 3;
	}
	.prog {
		fill: none;
		stroke: #f5bf48;
		stroke-width: 3;
		stroke-dasharray: 100;
		stroke-linecap: round;
		transition: stroke-dashoffset 250ms linear;
	}
	.code-label {
		font: var(--md-sys-typescale-label-medium);
		opacity: 0.7;
	}
	.code {
		font: 700 clamp(1.5rem, 4vw, 2.75rem) / 1 ui-monospace, 'Roboto Mono', Menlo, Consolas, monospace;
		letter-spacing: 0.12em;
	}
	.hint {
		opacity: 0.75;
		font: var(--md-sys-typescale-body-large);
		text-align: center;
	}
	.err {
		color: #ffb4ab;
	}
</style>
