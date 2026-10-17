<script lang="ts">
	import { university } from '#lib/content';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Sources from '#lib/components/Sources.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
</script>

<svelte:head>
	<title>O Politechnice Lubelskiej · UniQuest</title>
	<meta name="description" content={university.intro} />
</svelte:head>

<TopAppBar title="O uczelni" back={href('/explore')} />
<div class="page">
	<p class="body-large">{university.intro}</p>

	<div class="facts">
		{#each university.facts as f (f.label)}
			<div class="fact">
				<span class="num value">{f.value}</span>
				<span class="body-small muted">{f.label}</span>
			</div>
		{/each}
	</div>

	<h2 class="section-title">Historia</h2>
	<ol class="timeline">
		{#each university.history as h (h.year)}
			<li>
				<span class="year num">{h.year}</span>
				<p class="body-medium">{h.text}</p>
			</li>
		{/each}
	</ol>

	<h2 class="section-title">Czy wiesz, że…</h2>
	<Card variant="tertiary">
		<ul class="dyk body-medium">
			{#each university.didYouKnow as d (d.text)}
				<li>{d.text} <a href={d.source} target="_blank" rel="noopener" class="body-small">źródło</a></li>
			{/each}
		</ul>
	</Card>

	<h2 class="section-title">Władze (kadencja {university.authorities.term})</h2>
	<Card variant="filled">
		<p class="body-medium"><strong>Rektor:</strong> {university.authorities.rector}</p>
		<p class="body-medium muted">Prorektorzy: {university.authorities.prorectors.join(', ')}</p>
	</Card>

	<h2 class="section-title">Jednostki ogólnouczelniane</h2>
	<Card variant="outlined">
		<ul class="units body-medium">
			{#each university.units as u (u)}<li>{u}</li>{/each}
		</ul>
	</Card>

	<h2 class="section-title">Kontakt</h2>
	<Card variant="outlined" padding={false}>
		<ListItem icon="location_on" headline={university.address} />
		<ListItem icon="mail" headline={university.email} href="mailto:{university.email}" external />
		<ListItem icon="public" headline="pollub.pl" href={university.website} external />
		{#each university.contacts as c (c.name)}
			<ListItem icon="call" headline={c.name} supporting={c.phone} href="tel:+48{c.phone.replaceAll(' ', '')}" external />
		{/each}
	</Card>

	<Sources sources={university.sources} verifiedAt={university.verifiedAt} />
</div>

<style>
	.facts {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
		gap: 12px;
	}
	.fact {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 16px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.value {
		font-size: 1.25rem;
	}
	.timeline {
		list-style: none;
		margin: 0;
		padding: 0 0 0 8px;
		border-left: 3px solid var(--md-sys-color-primary-container);
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.timeline li {
		position: relative;
		padding-left: 20px;
	}
	.timeline li::before {
		content: '';
		position: absolute;
		left: -9px;
		top: 4px;
		width: 12px;
		height: 12px;
		border-radius: 3px;
		background: var(--md-sys-color-primary);
	}
	.year {
		color: var(--md-sys-color-primary);
	}
	.dyk,
	.units {
		margin: 0;
		padding-left: 20px;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.dyk a {
		color: inherit;
	}
</style>
