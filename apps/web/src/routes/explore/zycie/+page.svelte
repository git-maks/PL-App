<script lang="ts">
	import { studentLife } from '#lib/content';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Sources from '#lib/components/Sources.svelte';
</script>

<svelte:head><title>Życie studenckie na Politechnice Lubelskiej · UniQuest</title></svelte:head>

<TopAppBar title="Życie studenckie" back={href('/explore')} />
<div class="page">
	{#each studentLife.sections as s (s.id)}
		<h2 class="section-title">{s.title}</h2>
		<Card variant="filled" padding={false}>
			{#each s.items as item (item.name)}
				<ListItem headline={item.name} supporting={item.text} href={item.url} external={!!item.url} />
			{/each}
		</Card>
		{#if s.note}<p class="body-small muted">{s.note}</p>{/if}
	{/each}
	<Sources sources={studentLife.sources} verifiedAt={studentLife.verifiedAt} />
</div>
