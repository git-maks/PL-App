<script lang="ts">
	import { onDestroy } from 'svelte';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { poiById } from '#lib/content/campus';
	import { time } from '#lib/time';
	import { enablePush } from '#lib/pwa/push';
	import { toast } from '#lib/state/toast.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Chip from '#lib/components/Chip.svelte';
	import Card from '#lib/components/Card.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';
	import type { Session, Station } from '#lib/api';

	const FACULTIES = ['WM', 'WEiI', 'WBiA', 'WIŚiE', 'WZ', 'WMiIT'];
	const KIND: Record<Station['kind'], string> = { workshop: 'Warsztat', presentation: 'Prezentacja', show: 'Pokaz', place: 'Miejsce' };

	let filter = $state<string | null>(null);
	let now = $state(Date.now());
	const tick = setInterval(() => (now = Date.now()), 30_000);
	onDestroy(() => clearInterval(tick));

	const visited = $derived(new Set(app.progress?.scans.map((s) => s.stationId) ?? []));

	interface Item {
		station: Station;
		session: Session;
		bucket: 'now' | 'soon' | 'later' | 'past';
	}
	const items = $derived.by(() => {
		const out: Item[] = [];
		for (const st of app.stations) {
			if (filter === 'other' ? st.faculty !== null : filter && st.faculty !== filter) continue;
			for (const se of st.sessions) {
				const start = new Date(se.startsAt).getTime();
				const end = new Date(se.endsAt).getTime();
				const close = new Date(se.scanClosesAt).getTime();
				const bucket = now >= start && now <= close ? 'now' : start > now && start - now <= 60 * 60_000 ? 'soon' : start > now ? 'later' : end < now ? 'past' : 'later';
				out.push({ station: st, session: se, bucket });
			}
		}
		return out.sort((a, b) => a.session.startsAt.localeCompare(b.session.startsAt));
	});
	const groups = $derived(
		(
			[
				['now', 'Teraz'],
				['soon', 'W ciągu godziny'],
				['later', 'Później'],
				['past', 'Zakończone']
			] as const
		)
			.map(([key, title]) => ({ key, title, items: items.filter((i) => i.bucket === key) }))
			.filter((g) => g.items.length)
	);
	const total = $derived(app.stations.length);

	async function remind(sessionId: string) {
		try {
			const on = await app.toggleReminder(sessionId);
			if (on) {
				const res = await enablePush(app.api).catch(() => 'unsupported' as const);
				toast(res === 'enabled' ? 'Przypomnimy Ci 10 minut przed startem.' : 'Przypomnienie zapisane – zobaczysz je w aplikacji.');
			}
		} catch (e) {
			toast(errorMessage(e));
		}
	}
</script>

<svelte:head><title>Zadania · UniQuest</title></svelte:head>

<TopAppBar title="Zadania" compact>
	{#snippet actions()}
		<IconButton icon="map" label="Mapa kampusu" href={href('/map')} />
	{/snippet}
</TopAppBar>

<div class="page">
	{#if !app.ready}
		<Skeleton lines={5} />
	{:else if !app.event}
		<Card variant="tertiary">
			<p class="body-medium">Teraz nie trwa żadne wydarzenie UniQuest. Agenda pojawi się tu przed najbliższym Dniem Otwartym.</p>
		</Card>
	{:else}
		<Card variant="primary">
			<div class="row">
				<Icon name="event" />
				<div class="grow">
					<p class="title-small">{app.event.name}</p>
					<p class="body-small">Odwiedzone stacje: {visited.size}/{total}</p>
				</div>
			</div>
		</Card>

		<div class="chips" role="group" aria-label="Filtruj według wydziału">
			<Chip selected={filter === null} onclick={() => (filter = null)}>Wszystkie</Chip>
			{#each FACULTIES as f (f)}
				<Chip selected={filter === f} onclick={() => (filter = filter === f ? null : f)}>{f}</Chip>
			{/each}
			<Chip selected={filter === 'other'} onclick={() => (filter = filter === 'other' ? null : 'other')}>Kampus</Chip>
		</div>

		{#each groups as g (g.key)}
			<section class="stack" aria-labelledby="g-{g.key}">
				<h2 id="g-{g.key}" class="section-title">{g.title}</h2>
				{#each g.items as { station, session } (session.id)}
					{@const done = visited.has(station.id)}
					{@const reminded = app.reminders.includes(session.id)}
					<article class="quest" class:done class:past={g.key === 'past'}>
						<div class="time">
							<span class="title-small">{time(session.startsAt)}</span>
							<span class="body-small muted">{time(session.endsAt)}</span>
						</div>
						<div class="body">
							<div class="meta label-small">
								<span>{KIND[station.kind]}</span>
								{#if station.faculty}<span>· {station.faculty}</span>{/if}
								{#if station.team}<span class="team"><Icon name="groups" size={14} /> drużynowe</span>{/if}
							</div>
							<h3 class="title-medium">{station.name}</h3>
							<p class="body-small muted">{station.description}</p>
							<a class="where body-small" href={href('/map?poi=' + station.poi)}>
								<Icon name="location_on" size={16} />{poiById[station.poi]?.name ?? station.poi}{session.room ? ` · ${session.room}` : ''}
							</a>
						</div>
						<div class="side">
							{#if done}
								<span class="check" title="Zaliczone"><Icon name="check_circle" label="Zaliczone" /></span>
							{:else if g.key === 'soon' || g.key === 'later'}
								<IconButton icon={reminded ? 'notifications_active' : 'notifications'} label={reminded ? 'Wyłącz przypomnienie' : 'Przypomnij mi'} selected={reminded} onclick={() => remind(session.id)} />
							{/if}
						</div>
					</article>
				{/each}
			</section>
		{/each}
	{/if}
</div>

<style>
	.grow {
		flex: 1;
	}
	.chips {
		display: flex;
		gap: 8px;
		overflow-x: auto;
		margin: 0 calc(-1 * var(--uq-gutter));
		padding: 0 var(--uq-gutter) 4px;
		scrollbar-width: none;
	}
	.quest {
		display: flex;
		gap: 12px;
		padding: 12px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
		border: 1px solid var(--md-sys-color-outline-variant);
	}
	.quest.done {
		background: color-mix(in srgb, var(--md-sys-color-success-container) 45%, var(--md-sys-color-surface));
		border-color: transparent;
	}
	.quest.past:not(.done) {
		opacity: 0.6;
	}
	.time {
		width: 48px;
		flex: none;
		display: flex;
		flex-direction: column;
		align-items: center;
		padding-top: 2px;
	}
	.body {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.meta {
		display: flex;
		gap: 6px;
		align-items: center;
		color: var(--md-sys-color-primary);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}
	.team {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		color: var(--md-sys-color-tertiary);
	}
	.where {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		text-decoration: none;
	}
	.side {
		flex: none;
		display: flex;
		align-items: flex-start;
	}
	.check {
		color: var(--md-sys-color-success);
		padding: 12px;
	}
</style>
