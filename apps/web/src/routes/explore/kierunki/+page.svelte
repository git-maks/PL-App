<script lang="ts">
	import { programs, faculties } from '#lib/content';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Chip from '#lib/components/Chip.svelte';
	import ListItem from '#lib/components/ListItem.svelte';

	let level = $state<'I' | 'II' | null>(null);
	let extramural = $state(false);
	let english = $state(false);

	const filtered = $derived(
		programs.filter((p) => (!level || p.levels.includes(level)) && (!extramural || p.modes.includes('niestacjonarne')) && (!english || p.language === 'en'))
	);
	const groups = $derived(faculties.map((f) => ({ f, items: filtered.filter((p) => p.faculty === f.code) })).filter((g) => g.items.length));
</script>

<svelte:head>
	<title>Kierunki studiów na Politechnice Lubelskiej · UniQuest</title>
	<meta name="description" content="Wszystkie kierunki studiów Politechniki Lubelskiej: opisy, przedmioty maturalne i ścieżki kariery." />
</svelte:head>

<TopAppBar title="Kierunki studiów" back={href('/explore')} />
<div class="page">
	<div class="chips" role="group" aria-label="Filtry">
		<Chip selected={level === 'I'} onclick={() => (level = level === 'I' ? null : 'I')}>I stopnia</Chip>
		<Chip selected={level === 'II'} onclick={() => (level = level === 'II' ? null : 'II')}>II stopnia</Chip>
		<Chip selected={extramural} onclick={() => (extramural = !extramural)}>Niestacjonarne</Chip>
		<Chip selected={english} onclick={() => (english = !english)}>Po angielsku</Chip>
	</div>
	<p class="body-small muted" aria-live="polite">{filtered.length} kierunków</p>
	{#each groups as g (g.f.code)}
		<h2 class="section-title">{g.f.name}</h2>
		<Card variant="filled" padding={false}>
			{#each g.items as p (p.slug)}
				<ListItem headline={p.name} supporting={p.studies} href={href('/explore/kierunki/' + p.slug)} />
			{/each}
		</Card>
	{/each}
</div>

<style>
	.chips {
		display: flex;
		gap: 8px;
		overflow-x: auto;
		margin: 0 calc(-1 * var(--uq-gutter));
		padding: 0 var(--uq-gutter) 4px;
	}
</style>
