<script lang="ts">
	import { programBySlug, clubBySlug } from '#lib/content';
	import { statByCode } from '#lib/content/archetypes';
	import { href } from '#lib/paths';
	import { STAT_CODES } from '#lib/game/types';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Button from '#lib/components/Button.svelte';
	import Sources from '#lib/components/Sources.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';

	let { data } = $props();
	const f = $derived(data.faculty);
	const top = $derived(STAT_CODES.filter((c) => (f.vector[c] ?? 0) > 0).sort((a, b) => (f.vector[b] ?? 0) - (f.vector[a] ?? 0)));
</script>

<svelte:head>
	<title>{f.name} · UniQuest</title>
	<meta name="description" content={f.description} />
</svelte:head>

<TopAppBar title={f.code} back={href('/explore/wydzialy')} heading={false} />
<div class="page">
	<h1 class="headline-small">{f.name}</h1>
	<p class="body-large">{f.description}</p>
	<div class="row wrap">
		{#each top as c (c)}
			<span class="stat-chip label-medium" style="--c: {statByCode[c].color}"><StatIcon code={c} size={16} />{statByCode[c].archetype}</span>
		{/each}
	</div>
	<Card variant="outlined" padding={false}>
		<ListItem icon="location_on" headline={f.address} href={href('/map?poi=' + f.poi)} />
		<ListItem icon="public" headline={f.website.replace('https://', '')} href={f.website} external />
	</Card>

	<h2 class="section-title">Kierunki studiów</h2>
	<Card variant="filled" padding={false}>
		{#each f.programs as slug (slug)}
			{@const p = programBySlug[slug]}
			{#if p}<ListItem headline={p.name} supporting={p.studies} href={href('/explore/kierunki/' + slug)} />{/if}
		{/each}
	</Card>

	<h2 class="section-title">Koła naukowe</h2>
	<Card variant="outlined" padding={false}>
		{#each f.clubs as slug (slug)}
			{@const c = clubBySlug[slug]}
			{#if c}<ListItem headline={c.name} supporting={c.tags.join(' · ')} href={href('/explore/kola/' + slug)} />{/if}
		{/each}
	</Card>

	<Button variant="tonal" icon="open_in_new" href={f.website} target="_blank" rel="noopener">Strona wydziału</Button>
	<Sources sources={f.sources} verifiedAt={f.verifiedAt} />
</div>


