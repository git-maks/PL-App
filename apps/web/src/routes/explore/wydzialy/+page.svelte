<script lang="ts">
	import { faculties } from '#lib/content';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Icon from '#lib/components/Icon.svelte';
</script>

<svelte:head><title>Wydziały Politechniki Lubelskiej · UniQuest</title></svelte:head>

<TopAppBar title="Wydziały" back={href('/explore')} />
<div class="page">
	{#each faculties as f (f.code)}
		<Card variant="filled" href={href('/explore/wydzialy/' + f.slug)}>
			<div class="row top">
				<span class="code pixel">{f.code}</span>
				<div class="grow">
					<h2 class="title-medium">{f.name}</h2>
					<p class="body-small muted">{f.address}</p>
					<p class="body-small">{f.programs.length} kierunków · {f.clubs.length} kół naukowych</p>
				</div>
				<Icon name="chevron_right" />
			</div>
		</Card>
	{/each}
</div>

<style>
	.top {
		align-items: flex-start;
	}
	.grow {
		flex: 1;
	}
	.code {
		flex: none;
		min-width: 64px;
		height: 48px;
		padding: 0 8px;
		border-radius: 12px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
</style>
