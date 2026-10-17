<script lang="ts">
	import { onMount } from 'svelte';
	import { href } from '#lib/paths';
	import { university, faculties, programs, clubs } from '#lib/content';
	import { search, type SearchDoc } from '#lib/content/search';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import type { IconName } from '#lib/icons/paths';

	let q = $state('');
	const results = $derived<SearchDoc[]>(q.trim().length >= 2 ? search(q) : []);
	let fact = $state(0);
	onMount(() => {
		fact = Math.floor(Math.random() * university.didYouKnow.length);
	});

	const sections: { path: string; title: string; text: string; icon: IconName }[] = [
		{ path: '/explore/uczelnia', title: 'O uczelni', text: 'Historia, liczby, władze, jednostki', icon: 'school' },
		{ path: '/explore/wydzialy', title: 'Wydziały', text: `${faculties.length} wydziałów`, icon: 'apartment' },
		{ path: '/explore/kierunki', title: 'Kierunki studiów', text: `${programs.length} kierunków`, icon: 'menu_book' },
		{ path: '/explore/kola', title: 'Koła naukowe', text: `${clubs.length} kół naukowych`, icon: 'science' },
		{ path: '/map', title: 'Kampus', text: 'Mapa i budynki', icon: 'map' },
		{ path: '/explore/zycie', title: 'Życie studenckie', text: 'Akademiki, kultura, sport', icon: 'celebration' },
		{ path: '/explore/rekrutacja', title: 'Rekrutacja', text: 'Jak zostać studentem PL', icon: 'how_to_reg' }
	];
	const TYPE: Record<SearchDoc['type'], string> = { faculty: 'Wydział', program: 'Kierunek', club: 'Koło naukowe' };
</script>

<svelte:head>
	<title>Odkrywaj Politechnikę Lubelską · UniQuest</title>
	<meta name="description" content="Wydziały, kierunki studiów, koła naukowe i kampus Politechniki Lubelskiej w jednym miejscu." />
</svelte:head>

<TopAppBar title="Odkrywaj" />
<div class="page">
	<div class="search">
		<Icon name="search" />
		<input type="search" bind:value={q} placeholder="Szukaj kierunku, koła, wydziału…" aria-label="Szukaj w kompendium" />
	</div>

	{#if q.trim().length >= 2}
		<section aria-live="polite">
			{#if results.length === 0}
				<p class="body-medium muted">Brak wyników dla „{q}”.</p>
			{:else}
				<Card variant="outlined" padding={false}>
					{#each results as r (r.type + r.slug)}
						<ListItem overline={TYPE[r.type]} headline={r.title} supporting={r.subtitle} href={href(r.path)} />
					{/each}
				</Card>
			{/if}
		</section>
	{:else}
		<Card variant="primary">
			<div class="stack">
				<h2 class="title-large">{university.name}</h2>
				<p class="body-medium">{university.intro}</p>
			</div>
		</Card>

		<Card variant="tertiary">
			<div class="row fact">
				<Icon name="lightbulb" />
				<div class="stack tight">
					<p class="label-large">Czy wiesz, że…</p>
					<p class="body-medium">{university.didYouKnow[fact]?.text}</p>
				</div>
			</div>
		</Card>

		<div class="grid">
			{#each sections as s (s.path)}
				<a class="tile interactive" href={href(s.path)}>
					<span class="ti"><Icon name={s.icon} /></span>
					<span class="title-small">{s.title}</span>
					<span class="body-small muted">{s.text}</span>
				</a>
			{/each}
		</div>
	{/if}
</div>

<style>
	.search {
		display: flex;
		align-items: center;
		gap: 12px;
		height: 56px;
		padding: 0 16px;
		border-radius: var(--md-sys-shape-corner-full);
		background: var(--md-sys-color-surface-container-high);
		color: var(--md-sys-color-on-surface-variant);
	}
	.search input {
		flex: 1;
		height: 100%;
		border: none;
		background: transparent;
		color: var(--md-sys-color-on-surface);
		font: var(--md-sys-typescale-body-large);
	}
	.search input:focus {
		outline: none;
	}
	.search:focus-within {
		outline: 2px solid var(--md-sys-color-primary);
	}
	.fact {
		align-items: flex-start;
	}
	.tight {
		gap: 4px;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 12px;
	}
	.tile {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 16px;
		min-height: 120px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
		border: 1px solid var(--md-sys-color-outline-variant);
		color: var(--md-sys-color-on-surface);
		text-decoration: none;
	}
	.ti {
		width: 40px;
		height: 40px;
		border-radius: 12px;
		display: grid;
		place-items: center;
		margin-bottom: 8px;
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
</style>
