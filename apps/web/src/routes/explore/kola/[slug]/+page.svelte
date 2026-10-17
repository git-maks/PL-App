<script lang="ts">
	import { facultyByCode, programs } from '#lib/content';
	import { statByCode } from '#lib/content/archetypes';
	import { recommend, emptyStats } from '#lib/game/scoring';
	import { STAT_CODES } from '#lib/game/types';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Sources from '#lib/components/Sources.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';

	let { data } = $props();
	const c = $derived(data.club);
	const faculty = $derived(facultyByCode[c.faculty]);
	const attrs = $derived(STAT_CODES.filter((s) => (c.vector[s] ?? 0) > 0).sort((a, b) => (c.vector[b] ?? 0) - (c.vector[a] ?? 0)));
	const related = $derived(recommend({ ...emptyStats(), ...c.vector }, programs, 3));
</script>

<svelte:head><title>{c.name} · UniQuest</title></svelte:head>

<TopAppBar title="Koło naukowe" back={href('/explore/kola')} heading={false} />
<div class="page">
	<a class="label-large fac" href={href('/explore/wydzialy/' + faculty?.slug)}>{faculty?.name}</a>
	<h1 class="headline-small">{c.name}</h1>
	<div class="row wrap">
		{#each c.tags as t (t)}<span class="tag label-large">{t}</span>{/each}
	</div>
	<h2 class="section-title">Dla jakich bohaterów?</h2>
	<div class="row wrap">
		{#each attrs as s (s)}
			<span class="stat-chip label-medium" style="--c: {statByCode[s].color}"><StatIcon code={s} size={16} />{statByCode[s].archetype}</span>
		{/each}
	</div>
	<p class="body-medium muted">Koła działają przy katedrach wydziału — o dołączenie najłatwiej zapytać opiekuna koła lub starszych studentów na zajęciach. Aktualne informacje znajdziesz na stronie wydziału.</p>
	<h2 class="section-title">Pokrewne kierunki</h2>
	<Card variant="filled" padding={false}>
		{#each related as r (r.slug)}
			{@const p = programs.find((x) => x.slug === r.slug)}
			{#if p}<ListItem headline={p.name} supporting={facultyByCode[p.faculty]?.code} href={href('/explore/kierunki/' + p.slug)} />{/if}
		{/each}
	</Card>
	<Card variant="outlined" padding={false}>
		<ListItem icon="public" headline="Strona wydziału" supporting={faculty?.website} href={faculty?.website} external />
	</Card>
	<Sources sources={c.sources} verifiedAt={c.verifiedAt} />
</div>

<style>
	.fac {
		text-decoration: none;
	}
	.tag {
		padding: 6px 12px;
		border-radius: var(--md-sys-shape-corner-small);
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
</style>
