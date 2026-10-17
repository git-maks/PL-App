<script lang="ts">
	import { clubs, faculties } from '#lib/content';
	import { normalize } from '#lib/content/search';
	import { href } from '#lib/paths';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import TextField from '#lib/components/TextField.svelte';

	let q = $state('');
	const filtered = $derived(
		q.trim() ? clubs.filter((c) => normalize(c.name + ' ' + c.tags.join(' ')).includes(normalize(q.trim()))) : clubs
	);
	const groups = $derived(faculties.map((f) => ({ f, items: filtered.filter((c) => c.faculty === f.code) })).filter((g) => g.items.length));
</script>

<svelte:head>
	<title>Koła naukowe Politechniki Lubelskiej · UniQuest</title>
	<meta name="description" content="Studenckie koła naukowe Politechniki Lubelskiej — od dronów i rakiet po AI i architekturę." />
</svelte:head>

<TopAppBar title="Koła naukowe" back={href('/explore')} />
<div class="page">
	<p class="body-medium muted">Koła naukowe to miejsca, gdzie studenci budują drony, roboty i bolidy, piszą oprogramowanie i jeżdżą na konkursy — często już od pierwszego roku.</p>
	<TextField type="search" label="Filtruj (np. drony, AI, architektura)" bind:value={q} autocomplete="off" />
	{#each groups as g (g.f.code)}
		<h2 class="section-title">{g.f.name}</h2>
		<Card variant="filled" padding={false}>
			{#each g.items as c (c.slug)}
				<ListItem headline={c.name} supporting={c.tags.join(' · ')} href={href('/explore/kola/' + c.slug)} />
			{/each}
		</Card>
	{/each}
	{#if groups.length === 0}<p class="body-medium muted">Brak kół pasujących do „{q}”.</p>{/if}
</div>
