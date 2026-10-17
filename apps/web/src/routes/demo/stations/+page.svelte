<script lang="ts">
	import { isDemo, api } from '#lib/api';
	import { href } from '#lib/paths';
	import { app } from '#lib/state/app.svelte';
	import type { DemoApi } from '#lib/api/demo';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Button from '#lib/components/Button.svelte';
	import QrCode from '#lib/qr/QrCode.svelte';

	let posters = $state<{ id: string; name: string; payload: string; team: boolean }[]>([]);
	$effect(() => {
		if (!isDemo || posters.length || app.stations.length === 0) return;
		void (async () => {
			const client = (await api()) as DemoApi;
			posters = await Promise.all(
				app.stations.map(async (s) => ({ id: s.id, name: s.name, payload: await client.staticPayload(s.id), team: !!s.team }))
			);
		})();
	});
</script>

<svelte:head><title>Arkusz stacji testowych · UniQuest</title></svelte:head>

<TopAppBar title="Stacje testowe (demo)" back={href('/')} />
<div class="page">
	{#if !isDemo}
		<p class="body-medium">Arkusz jest dostępny tylko w wersji demonstracyjnej. W prawdziwym wydarzeniu kody wyświetla panel stacji.</p>
	{:else}
		<p class="body-medium">
			Aby przetestować grę, otwórz tę stronę na drugim urządzeniu (albo wydrukuj) i zeskanuj kody aplikacją UniQuest na telefonie.
			To statyczne kody plakatowe; rotujący kod co 30 s pokazuje <a href={href('/staff')}>panel stacji</a> (włącz tryb organizatora w ustawieniach).
		</p>
		<Button variant="tonal" icon="download" onclick={() => print()}>Drukuj arkusz</Button>
		<div class="sheet">
			{#each posters as p (p.id)}
				<figure class="poster">
					<QrCode text={p.payload} size={200} label="Kod stacji {p.name}" />
					<figcaption class="body-small">{p.name}{p.team ? ' · drużynowe' : ''}</figcaption>
				</figure>
			{/each}
		</div>
	{/if}
</div>

<style>
	.sheet {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: 16px;
	}
	.poster {
		margin: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 12px;
		border-radius: var(--md-sys-shape-corner-large);
		border: 1px solid var(--md-sys-color-outline-variant);
		background: #fff;
		color: #000;
		text-align: center;
		break-inside: avoid;
	}
	@media print {
		:global(nav),
		:global(header) {
			display: none !important;
		}
	}
</style>
