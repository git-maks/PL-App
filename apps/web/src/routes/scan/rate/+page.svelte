<script lang="ts">
	import { page } from '$app/state';
	import { scale, fly } from 'svelte/transition';
	import { backOut } from 'svelte/easing';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { archetypes, statByCode } from '#lib/content/archetypes';
	import { STAT_CODES } from '#lib/game/types';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import RatingSlider from '#lib/components/RatingSlider.svelte';
	import Button from '#lib/components/Button.svelte';
	import Card from '#lib/components/Card.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import LinearProgress from '#lib/components/LinearProgress.svelte';
	import type { Scan } from '#lib/api';

	const id = $derived(page.url.searchParams.get('id'));
	const queuedId = $derived(page.url.searchParams.get('queued'));
	const again = $derived(page.url.searchParams.has('again'));

	let rating = $state(3);
	let busy = $state(false);
	let error = $state<string | null>(null);
	let result = $state<{ scan: Scan; levelUp: number | null; unlocked: boolean } | null>(null);
	let queuedSaved = $state(false);

	const scan = $derived(app.progress?.scans.find((s) => s.id === id) ?? null);
	const station = $derived(scan ? app.stationById.get(scan.stationId) : null);
	const queued = $derived(app.queue.find((q) => q.id === queuedId) ?? null);
	const gained = $derived(result ? STAT_CODES.filter((c) => result!.scan.deltas[c] > 0).sort((a, b) => result!.scan.deltas[b] - result!.scan.deltas[a]) : []);
	const xp = $derived(result ? Math.round(STAT_CODES.reduce((a, c) => a + result!.scan.deltas[c], 0)) : 0);

	$effect(() => {
		if (scan?.rating) rating = scan.rating;
	});
	$effect(() => {
		// Opened from a deep link before progress loaded.
		if (id && app.me && app.progress && !scan) void app.refresh();
	});

	async function submit() {
		busy = true;
		error = null;
		try {
			if (queuedId) {
				await app.rateQueued(queuedId, rating);
				queuedSaved = true;
				return;
			}
			if (!id) return;
			const before = app.progress;
			const res = await app.rate(id, rating);
			navigator.vibrate?.([30, 40, 30]);
			result = {
				scan: res.scan,
				levelUp: before && res.progress.level > before.level ? res.progress.level : null,
				unlocked: !!before && !before.cardUnlocked && res.progress.cardUnlocked
			};
		} catch (e) {
			error = errorMessage(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Oceń · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title={result ? 'Zdobyte punkty' : 'Jak było?'} back={href('/hero')} heading={false} />
	<div class="page">
		{#if queuedId}
			<Card variant="tertiary">
				<div class="row">
					<Icon name="wifi_off" />
					<p class="body-medium">Brak zasięgu — skan zapisaliśmy na telefonie i wyślemy, gdy wróci internet.</p>
				</div>
			</Card>
			{#if queuedSaved}
				<div class="done" in:scale={{ start: 0.8, duration: 300, easing: backOut }}>
					<Icon name="check_circle" size={64} />
					<h2 class="headline-small">Ocena zapisana</h2>
					<p class="body-medium muted">Punkty pojawią się po synchronizacji.</p>
				</div>
				<Button full href={href('/hero')} icon="person">Do panelu bohatera</Button>
			{:else if queued}
				<RatingSlider bind:value={rating} labels={archetypes.rating.labels} emoji={archetypes.rating.emoji} label={archetypes.rating.question} />
				<Button full loading={busy} icon="check" onclick={submit}>Zapisz ocenę</Button>
			{:else}
				<p class="body-medium muted">Ten skan został już wysłany.</p>
				<Button full href={href('/hero')}>Do panelu bohatera</Button>
			{/if}
		{:else if !scan}
			<LinearProgress indeterminate label="Wczytywanie" />
		{:else if result}
			<div class="reward" in:scale={{ start: 0.7, duration: 450, easing: backOut }}>
				<span class="xp pixel">+{xp} XP</span>
				<p class="title-medium">{scan.stationName}</p>
				{#if result.scan.teamStatus === 'pending'}
					<p class="body-medium muted">Bonus drużynowy ×1,2 dostaniesz, gdy dotrze reszta drużyny ({result.scan.teamProgress?.scanned}/{result.scan.teamProgress?.required}).</p>
				{:else if result.scan.team}
					<p class="body-medium">Bonus drużynowy ×1,2 naliczony!</p>
				{/if}
			</div>
			<Card variant="filled">
				<ul class="gains">
					{#each gained as code, i (code)}
						<li in:fly={{ x: -16, delay: 150 + i * 90, duration: 300 }} style="--c: {statByCode[code].color}">
							<span class="gi"><StatIcon {code} size={20} /></span>
							<span class="body-large grow">{statByCode[code].name}</span>
							<span class="pixel title-medium plus">+{Math.round(result.scan.deltas[code])}</span>
						</li>
					{/each}
				</ul>
			</Card>
			{#if result.levelUp}
				<Card variant="primary">
					<div class="row" in:scale={{ delay: 500, duration: 350, easing: backOut }}>
						<Icon name="celebration" size={32} />
						<div>
							<p class="title-medium">Awans na poziom {result.levelUp}!</p>
							<p class="body-small">Twoja postać rośnie w siłę.</p>
						</div>
					</div>
				</Card>
			{/if}
			{#if result.unlocked}
				<Card variant="tertiary" href={href('/card')}>
					<div class="row">
						<Icon name="workspace_premium" size={32} />
						<div class="grow">
							<p class="title-medium">Karta postaci odblokowana!</p>
							<p class="body-small">Zobacz, jaką ścieżką podążasz.</p>
						</div>
						<Icon name="chevron_right" />
					</div>
				</Card>
			{/if}
			<div class="actions">
				<Button full icon="qr_code_scanner" href={href('/scan')}>Skanuj kolejny kod</Button>
				<Button full variant="tonal" icon="person" href={href('/hero')}>Panel bohatera</Button>
			</div>
		{:else}
			<header class="intro">
				<p class="label-large kicker">{station?.kind === 'workshop' ? 'Warsztat' : station?.kind === 'show' ? 'Pokaz' : station?.kind === 'presentation' ? 'Prezentacja' : 'Stacja'}</p>
				<h1 class="headline-small">{scan.stationName}</h1>
				{#if again}<p class="body-medium muted">Ta stacja jest już zaliczona. Możesz poprawić ocenę przez 15 minut od skanu.</p>{/if}
			</header>
			<Card variant="filled">
				<div class="stack">
					<p class="title-medium q">{archetypes.rating.question}</p>
					<RatingSlider bind:value={rating} labels={archetypes.rating.labels} emoji={archetypes.rating.emoji} label={archetypes.rating.question} />
				</div>
			</Card>
			<p class="body-small muted">Im wyżej ocenisz, tym mocniej Twoja postać przesunie się w stronę atrybutów tej stacji.</p>
			{#if error}<p class="error body-medium" role="alert">{error}</p>{/if}
			<Button full loading={busy} icon="check" onclick={submit}>Zatwierdź ocenę</Button>
		{/if}
	</div>
</AuthGate>

<style>
	.intro {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.kicker {
		color: var(--md-sys-color-primary);
	}
	.q {
		text-align: center;
	}
	.error {
		color: var(--md-sys-color-error);
	}
	.reward,
	.done {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 8px;
		padding: 24px 0 8px;
	}
	.done {
		color: var(--md-sys-color-success);
	}
	.xp {
		font-size: 56px;
		line-height: 1;
		color: var(--md-sys-color-primary);
	}
	.gains {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.gains li {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.gi {
		width: 36px;
		height: 36px;
		border-radius: 10px;
		display: grid;
		place-items: center;
		color: var(--c);
		background: color-mix(in srgb, var(--c) 14%, var(--md-sys-color-surface));
	}
	.grow {
		flex: 1;
	}
	.plus {
		color: var(--md-sys-color-success);
	}
	.actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
</style>
