<script lang="ts">
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { time } from '#lib/time';
	import type { StaffStation } from '#lib/api';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';
	import Switch from '#lib/components/Switch.svelte';

	let list = $state<StaffStation[] | null>(null);
	let error = $state<string | null>(null);
	async function load() {
		try {
			list = await app.api.staffStations();
		} catch (e) {
			error = errorMessage(e);
		}
	}
	$effect(() => {
		if (app.me) void load();
	});
	async function toggle(s: StaffStation, open: boolean) {
		const updated = await app.api.setScanWindow(s.station.id, open, 20);
		list = list?.map((x) => (x.station.id === s.station.id ? updated : x)) ?? null;
	}
</script>

<svelte:head><title>Panel stacji · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Panel stacji" back={href('/settings')} />
	<div class="page">
		{#if app.me && app.me.role === 'participant'}
			<p class="body-medium">Ten panel jest dostępny dla obsługi stacji. Uprawnienia nadaje administrator wydarzenia.</p>
		{:else if error}
			<p role="alert">{error}</p>
		{:else if !list}
			<Skeleton lines={5} />
		{:else}
			<p class="body-medium muted">Na koniec zajęć otwórz skanowanie i wyświetl kod na ekranie lub rzutniku. Kod zmienia się co 30 sekund, więc zdjęcie kodu szybko traci ważność.</p>
			{#each list as s (s.station.id)}
				<Card variant="filled">
					<div class="stack">
						<div>
							<h2 class="title-medium">{s.station.name}</h2>
							<p class="body-small muted">{s.station.sessions.map((x) => `${time(x.startsAt)}–${time(x.endsAt)}`).join(' · ')}</p>
						</div>
						<div class="stats row wrap label-large">
							<span>Skany: <strong>{s.scansCount}</strong></span>
							<span>Ocenione: <strong>{s.ratedCount}</strong></span>
							<span>Średnia: <strong>{s.avgRating ? s.avgRating.toFixed(1) : '–'}</strong></span>
						</div>
						<Switch label="Skanowanie otwarte" supporting={s.scanOpenUntil ? `do ${time(s.scanOpenUntil)}` : 'Okno skanowania wg harmonogramu lub ręcznie'} checked={s.scanOpen} onchange={(v) => toggle(s, v)} />
						<div class="row wrap">
							<Button icon="qr_code_2" href={href('/staff/display?id=' + s.station.id)}>Wyświetl kod QR</Button>
							<Button variant="text" icon="download" href={href('/staff/poster?id=' + s.station.id)}>Plakat do druku</Button>
						</div>
					</div>
				</Card>
			{/each}
		{/if}
	</div>
</AuthGate>

<style>
	.stats {
		gap: 16px;
	}
</style>
