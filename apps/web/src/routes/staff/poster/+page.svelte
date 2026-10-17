<script lang="ts">
	import { page } from '$app/state';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { time } from '#lib/time';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import Button from '#lib/components/Button.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import QrCode from '#lib/qr/QrCode.svelte';

	// Printable fallback for stations without a screen: a static code that the server accepts
	// only while the station's scan window is open (spec §8.3).
	const id = $derived(page.url.searchParams.get('id') ?? '');
	const station = $derived(app.stationById.get(id));
	let payload = $state<string | null>(null);
	let error = $state<string | null>(null);

	$effect(() => {
		if (!app.me || !id || payload) return;
		app.api.stationPoster(id).then(
			(p) => (payload = p),
			(e) => (error = errorMessage(e))
		);
	});
</script>

<svelte:head><title>Plakat stacji · UniQuest</title></svelte:head>

<AuthGate>
	<div class="actions no-print">
		<IconButton icon="close" label="Zamknij" href={href('/staff')} />
		<span class="spacer"></span>
		<Button variant="tonal" icon="download" disabled={!payload} onclick={() => print()}>Drukuj</Button>
	</div>
	<article class="poster">
		<p class="kicker label-large">UniQuest · Politechnika Lubelska</p>
		<h1 class="headline-medium">{station?.name ?? 'Stacja'}</h1>
		{#if error}
			<p role="alert">{error}</p>
		{:else if payload}
			<QrCode text={payload} size={360} label="Kod QR stacji {station?.name ?? ''}" />
		{/if}
		<p class="title-medium">Zeskanuj aparatem lub w aplikacji, aby zdobyć punkty</p>
		{#if station?.sessions.length}
			<p class="body-medium">{station.sessions.map((s) => `${time(s.startsAt)}–${time(s.endsAt)}`).join(' · ')}</p>
		{/if}
		<p class="body-small note">Kod działa tylko w czasie otwartego okna skanowania tej stacji.</p>
	</article>
</AuthGate>

<style>
	.actions {
		display: flex;
		align-items: center;
		padding: 8px 12px;
	}
	.poster {
		max-width: 560px;
		margin: 0 auto;
		padding: 24px 16px 48px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
		text-align: center;
	}
	.poster :global(svg) {
		width: min(100%, 360px);
		height: auto;
	}
	.kicker {
		color: var(--md-sys-color-primary);
		margin: 0;
	}
	h1 {
		margin: 0;
	}
	.note {
		color: var(--md-sys-color-on-surface-variant);
	}
	@media print {
		.no-print {
			display: none !important;
		}
		.poster {
			color: #000;
		}
	}
</style>
