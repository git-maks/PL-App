<script lang="ts">
	import { page } from '$app/state';
	import { app } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { campus, poiById, facultySlugByCode } from '#lib/content/campus';
	import { time } from '#lib/time';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import Dialog from '#lib/components/Dialog.svelte';
	import Button from '#lib/components/Button.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import type { Poi } from '#lib/content/types';

	const W = campus.map.width;
	const H = campus.map.height;
	const KIND_COLOR: Record<Poi['kind'], string> = {
		faculty: 'var(--md-sys-color-primary)',
		lab: '#7c3aed',
		library: '#0e7490',
		admin: '#b45309',
		sport: '#be123c',
		dorm: '#545f71',
		service: '#545f71',
		landmark: '#5f6368',
		green: '#15803d'
	};

	let zoom = $state(1);
	let selected = $state<Poi | null>(null);
	let sheetOpen = $state(false);

	$effect(() => {
		const id = page.url.searchParams.get('poi');
		if (id && poiById[id]) open(poiById[id]);
	});

	const stationsAt = $derived(selected ? app.stations.filter((s) => s.poi === selected!.id) : []);
	const activePois = $derived(new Set(app.stations.map((s) => s.poi)));
	const visitedPois = $derived(
		new Set((app.progress?.scans ?? []).map((s) => app.stationById.get(s.stationId)?.poi).filter(Boolean) as string[])
	);

	function open(p: Poi) {
		selected = p;
		sheetOpen = true;
	}
	function directions(p: Poi) {
		return `https://www.google.com/maps/dir/?api=1&destination=${p.lat},${p.lon}&travelmode=walking`;
	}
</script>

<svelte:head><title>Mapa kampusu · UniQuest</title></svelte:head>

<TopAppBar title="Mapa kampusu" back={true}>
	{#snippet actions()}
		<IconButton icon="zoom_out" label="Oddal" disabled={zoom <= 1} onclick={() => (zoom = Math.max(1, zoom - 0.5))} />
		<IconButton icon="zoom_in" label="Przybliż" disabled={zoom >= 3} onclick={() => (zoom = Math.min(3, zoom + 0.5))} />
	{/snippet}
</TopAppBar>

<div class="page">
	<div class="viewport">
		<svg viewBox="0 0 {W} {H}" style="width: {zoom * 100}%" role="group" aria-label="Mapa kampusu Politechniki Lubelskiej">
			<defs>
				<pattern id="grass" width="12" height="12" patternUnits="userSpaceOnUse">
					<rect width="12" height="12" fill="var(--map-green)" />
					<rect x="2" y="3" width="2" height="2" fill="var(--map-green-dark)" />
					<rect x="8" y="8" width="2" height="2" fill="var(--map-green-dark)" />
				</pattern>
			</defs>
			<rect width={W} height={H} fill="var(--map-ground)" />
			<!-- Bystrzyca valley (schematic) -->
			<path d="M {W * 0.5} {H} C {W * 0.62} {H * 0.82}, {W * 0.8} {H * 0.9}, {W} {H * 0.7} L {W} {H} Z" fill="url(#grass)" />
			<path d="M {W * 0.55} {H} C {W * 0.66} {H * 0.88}, {W * 0.82} {H * 0.95}, {W} {H * 0.8}" fill="none" stroke="var(--map-water)" stroke-width="14" stroke-linecap="round" />
			<text x={W * 0.86} y={H * 0.97} class="map-label">Bystrzyca</text>
			<!-- ul. Nadbystrzycka (schematic) -->
			<path d="M 0 {H * 0.07} C {W * 0.3} {H * 0.05}, {W * 0.6} {H * 0.03}, {W} {H * 0.015}" fill="none" stroke="var(--map-road)" stroke-width="16" stroke-linecap="round" />
			<text x={W * 0.3} y={H * 0.035} class="map-label">ul. Nadbystrzycka</text>

			{#each campus.pois as p (p.id)}
				{@const active = activePois.has(p.id)}
				{@const visited = visitedPois.has(p.id)}
				<g
					class="poi"
					class:active
					transform="translate({p.x} {p.y})"
					role="button"
					tabindex="0"
					aria-label="{p.n}. {p.name}{visited ? ', odwiedzone' : active ? ', są tu zadania' : ''}"
					onclick={() => open(p)}
					onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), open(p))}
				>
					{#if active && !visited}<circle r="20" class="pulse" fill={KIND_COLOR[p.kind]} />{/if}
					<rect x="-13" y="-13" width="26" height="26" rx="7" fill={visited ? 'var(--md-sys-color-success)' : KIND_COLOR[p.kind]} />
					<text text-anchor="middle" dominant-baseline="central" class="num">{p.n}</text>
				</g>
			{/each}
		</svg>
	</div>
	<p class="body-small muted">Mapa poglądowa – położenie budynków wg OpenStreetMap (© współtwórcy OSM). Pulsujące punkty mają aktywne zadania, zielone są już odwiedzone.</p>

	<ul class="legend">
		{#each campus.pois as p (p.id)}
			<li>
				<button type="button" class="leg interactive" onclick={() => open(p)}>
					<span class="dot" style="background: {visitedPois.has(p.id) ? 'var(--md-sys-color-success)' : KIND_COLOR[p.kind]}">{p.n}</span>
					<span class="body-medium">{p.name}</span>
				</button>
			</li>
		{/each}
	</ul>
</div>

<Dialog bind:open={sheetOpen} title={selected?.name} sheet>
	{#if selected}
		<div class="stack">
			{#if selected.address}<p class="row"><Icon name="location_on" size={18} />{selected.address}</p>{/if}
			{#if stationsAt.length}
				<p class="title-small on-surface">Zadania w tym miejscu</p>
				{#each stationsAt as st (st.id)}
					<ListItem
						headline={st.name}
						supporting={st.sessions.map((s) => `${time(s.startsAt)}–${time(s.endsAt)}`).join(', ')}
						icon={app.progress?.scans.some((s) => s.stationId === st.id) ? 'check_circle' : 'flag'}
					/>
				{/each}
			{/if}
			<div class="row wrap">
				<Button icon="directions" href={directions(selected)} target="_blank" rel="noopener">Nawiguj</Button>
				{#if selected.faculty}
					<Button variant="tonal" icon="school" href={href('/explore/wydzialy/' + facultySlugByCode[selected.faculty])}>O wydziale</Button>
				{/if}
			</div>
		</div>
	{/if}
</Dialog>

<style>
	.page {
		--map-ground: color-mix(in srgb, var(--md-sys-color-surface-container) 70%, #e9e4d8);
		--map-green: #b9dcb0;
		--map-green-dark: #9fcb94;
		--map-water: #79b8e6;
		--map-road: var(--md-sys-color-surface-container-lowest);
	}
	.viewport {
		overflow: auto;
		border-radius: var(--md-sys-shape-corner-large);
		border: 1px solid var(--md-sys-color-outline-variant);
		background: var(--map-ground);
		max-height: 70dvh;
	}
	svg {
		display: block;
		max-width: none;
		height: auto;
		transition: width var(--md-sys-motion-duration-medium2) var(--md-sys-motion-easing-standard);
	}
	.map-label {
		font: 600 14px var(--md-ref-typeface-plain);
		fill: var(--md-sys-color-on-surface-variant);
	}
	.poi {
		cursor: pointer;
		outline: none;
	}
	.poi:focus-visible rect {
		stroke: var(--md-sys-color-on-surface);
		stroke-width: 3;
	}
	.num {
		font: 700 13px var(--md-ref-typeface-plain);
		fill: #fff;
		pointer-events: none;
	}
	.pulse {
		opacity: 0.25;
		transform-box: fill-box;
		transform-origin: center;
		animation: pulse 1.8s ease-out infinite;
	}
	@keyframes pulse {
		from {
			transform: scale(0.6);
			opacity: 0.45;
		}
		to {
			transform: scale(1.4);
			opacity: 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.pulse {
			animation: none;
		}
	}
	.legend {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
		gap: 4px;
	}
	.leg {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px;
		border: none;
		background: transparent;
		text-align: left;
		border-radius: var(--md-sys-shape-corner-medium);
	}
	.dot {
		width: 26px;
		height: 26px;
		flex: none;
		border-radius: 7px;
		display: grid;
		place-items: center;
		color: #fff;
		font: var(--md-sys-typescale-label-medium);
	}
	.on-surface {
		color: var(--md-sys-color-on-surface);
	}
</style>
