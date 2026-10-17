<script lang="ts">
	import { facultyByCode, clubs } from '#lib/content';
	import { statByCode } from '#lib/content/archetypes';
	import { recommend } from '#lib/game/scoring';
	import { emptyStats } from '#lib/game/scoring';
	import { STAT_CODES } from '#lib/game/types';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Sources from '#lib/components/Sources.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';
	import Icon from '#lib/components/Icon.svelte';

	let { data } = $props();
	const p = $derived(data.program);
	const faculty = $derived(facultyByCode[p.faculty]);
	const attrs = $derived(STAT_CODES.filter((c) => (p.vector[c] ?? 0) > 0).sort((a, b) => (p.vector[b] ?? 0) - (p.vector[a] ?? 0)));
	// Clubs whose profile is closest to this programme's profile.
	const related = $derived(recommend({ ...emptyStats(), ...p.vector }, clubs, 4));
</script>

<svelte:head>
	<title>{p.name} — Politechnika Lubelska · UniQuest</title>
	<meta name="description" content={p.description.slice(0, 160)} />
</svelte:head>

<TopAppBar title={p.name} back={href('/explore/kierunki')} heading={false} />
<div class="page">
	<div class="stack tight">
		<a class="label-large fac" href={href('/explore/wydzialy/' + faculty?.slug)}>{faculty?.name}</a>
		<h1 class="headline-small">{p.name}</h1>
		<p class="body-medium muted">{p.studies}{p.practicalProfile ? ' · profil praktyczny' : ''}{p.language === 'en' ? ' · w języku angielskim' : ''}</p>
	</div>
	{#if p.note}<Card variant="tertiary"><p class="body-medium">{p.note}</p></Card>{/if}

	<div class="row wrap">
		{#each attrs as c (c)}
			<span class="stat-chip label-medium" style="--c: {statByCode[c].color}"><StatIcon code={c} size={16} />{statByCode[c].archetype} {Math.round((p.vector[c] ?? 0) * 100)}%</span>
		{/each}
	</div>

	<p class="body-large">{p.description}</p>

	{#if p.careersIntro || p.careers.length}
		<h2 class="section-title">Co po studiach?</h2>
		<Card variant="filled">
			{#if p.careersIntro}<p class="body-medium">{p.careersIntro}</p>{/if}
			{#if p.careers.length}
				<ul class="careers body-medium">
					{#each p.careers as c (c)}<li>{c}</li>{/each}
				</ul>
			{/if}
		</Card>
	{/if}

	{#if p.didYouKnow}
		<Card variant="tertiary">
			<div class="row top">
				<Icon name="lightbulb" />
				<p class="body-medium">{p.didYouKnow}</p>
			</div>
		</Card>
	{/if}

	{#if p.maturaSubjects.length}
		<h2 class="section-title">Przedmioty maturalne do wyboru</h2>
		<div class="row wrap">
			{#each p.maturaSubjects as m (m)}<span class="subject label-large">{m}</span>{/each}
		</div>
	{/if}

	<h2 class="section-title">Koła naukowe dla Ciebie</h2>
	<Card variant="outlined" padding={false}>
		{#each related as r (r.slug)}
			{@const c = clubs.find((x) => x.slug === r.slug)}
			{#if c}<ListItem headline={c.name} supporting={facultyByCode[c.faculty]?.code + ' · ' + c.tags.join(' · ')} href={href('/explore/kola/' + c.slug)} />{/if}
		{/each}
	</Card>

	<Button icon="open_in_new" href={p.recruitmentUrl} target="_blank" rel="noopener">Rekrutacja na ten kierunek</Button>
	<Sources sources={p.sources} verifiedAt={p.verifiedAt} />
</div>

<style>
	.tight {
		gap: 4px;
	}
	.top {
		align-items: flex-start;
	}
	.fac {
		text-decoration: none;
	}
	.careers {
		margin: 8px 0 0;
		padding-left: 20px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.subject {
		padding: 6px 12px;
		border-radius: var(--md-sys-shape-corner-small);
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
</style>
