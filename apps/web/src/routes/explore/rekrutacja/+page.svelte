<script lang="ts">
	import { recruitment } from '#lib/content';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Sources from '#lib/components/Sources.svelte';

	const s = recruitment.stats2026;
	const maxApps = Math.max(...s.mostApplications.map((x) => x.count));
</script>

<svelte:head><title>Rekrutacja na Politechnikę Lubelską · UniQuest</title></svelte:head>

<TopAppBar title="Rekrutacja" back={href('/explore')} />
<div class="page">
	<p class="body-large">{recruitment.intro}</p>
	<ol class="steps">
		{#each recruitment.steps as step, i (step.title)}
			<li>
				<span class="n pixel">{i + 1}</span>
				<div>
					<p class="title-small">{step.title}</p>
					<p class="body-medium muted">{step.text}</p>
				</div>
			</li>
		{/each}
	</ol>

	<h2 class="section-title">Rekrutacja 2026 w liczbach</h2>
	<Card variant="primary">
		<p class="body-medium">Kandydatów na studia stacjonarne I stopnia: <strong>{s.candidates}</strong>, zakwalifikowanych: <strong>{s.qualified}</strong> na {s.programs} kierunkach.</p>
	</Card>
	<h3 class="title-small">Najwięcej kandydatów na miejsce</h3>
	<Card variant="filled" padding={false}>
		{#each s.mostCompetitive as x (x.name)}
			<ListItem headline={x.name}>
				{#snippet trailing()}<span class="pixel">{x.perPlace.toLocaleString('pl-PL')}</span>{/snippet}
			</ListItem>
		{/each}
	</Card>
	<h3 class="title-small">Najwięcej zgłoszeń</h3>
	<div class="bars" role="list">
		{#each s.mostApplications as x (x.name)}
			<div class="bar-row" role="listitem">
				<span class="body-small name">{x.name}</span>
				<span class="bar"><span style="width: {(x.count / maxApps) * 100}%"></span></span>
				<span class="label-medium">{x.count}</span>
			</div>
		{/each}
	</div>
	<p class="body-small muted"><a href={s.source} target="_blank" rel="noopener">Źródło: pollub.pl, 14.07.2026</a></p>

	<h2 class="section-title">Przydatne linki</h2>
	<Card variant="outlined" padding={false}>
		{#each recruitment.links as l (l.url + l.title)}
			<ListItem headline={l.title} href={l.url} external />
		{/each}
	</Card>
	<Sources sources={recruitment.sources} verifiedAt={recruitment.verifiedAt} />
</div>

<style>
	.steps {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.steps li {
		display: flex;
		gap: 16px;
	}
	.n {
		flex: none;
		width: 36px;
		height: 36px;
		border-radius: 10px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.bars {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.bar-row {
		display: grid;
		grid-template-columns: minmax(110px, 40%) 1fr 40px;
		align-items: center;
		gap: 8px;
	}
	.name {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.bar {
		height: 12px;
		border-radius: 6px;
		background: var(--md-sys-color-surface-container-highest);
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		border-radius: 6px;
		background: var(--md-sys-color-primary);
	}
</style>
