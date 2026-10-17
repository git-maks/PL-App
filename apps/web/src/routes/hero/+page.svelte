<script lang="ts">
	import { onDestroy } from 'svelte';
	import { fly } from 'svelte/transition';
	import { flip } from 'svelte/animate';
	import { app } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { STAT_CODES } from '#lib/game/types';
	import { emptyStats } from '#lib/game/scoring';
	import { statByCode, titleFor } from '#lib/content/archetypes';
	import { BADGES } from '#lib/game/badges';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import PixelAvatar from '#lib/components/PixelAvatar.svelte';
	import StatBar from '#lib/components/StatBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import LinearProgress from '#lib/components/LinearProgress.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';

	const p = $derived(app.progress);
	const stats = $derived(p?.stats ?? emptyStats());
	const max = $derived(Math.max(100, ...STAT_CODES.map((c) => stats[c])));
	const ranked = $derived([...STAT_CODES].sort((a, b) => stats[b] - stats[a]));
	const title = $derived(titleFor(p?.titleKey));
	const pending = $derived(p?.scans.filter((s) => s.rating === null) ?? []);
	const recent = $derived(p?.scans.filter((s) => s.rating !== null).slice(0, 5) ?? []);

	// Show the gains from the last rating for a moment, then settle.
	let deltas = $state(app.lastDeltas);
	const t = setTimeout(() => {
		deltas = null;
		app.lastDeltas = null;
	}, 4000);
	onDestroy(() => clearTimeout(t));

	const frame = $derived(p?.archetypes[0] ? statByCode[p.archetypes[0].code].color : undefined);
</script>

<svelte:head><title>Bohater · UniQuest</title></svelte:head>

<AuthGate>
	<div class="page">
		<header class="head">
			<a href={href('/settings')} class="who" aria-label="Profil i ustawienia">
				<PixelAvatar src={app.me?.avatarUrl} seed={app.me?.id ?? ''} size={64} {frame} />
				<span class="names">
					<span class="title-medium nick">{app.me?.nickname}</span>
					<span class="body-small muted">{title ? title.title : 'Początkujący poszukiwacz'}</span>
				</span>
			</a>
			<span class="spacer"></span>
			<IconButton icon={app.unread ? 'notifications_active' : 'notifications'} label="Powiadomienia{app.unread ? ` (${app.unread} nowych)` : ''}" href={href('/notifications')} />
			<IconButton icon="settings" label="Ustawienia" href={href('/settings')} />
		</header>

		{#if !app.event}
			<Card variant="tertiary">
				<div class="stack">
					<h2 class="title-medium">Teraz nie trwa żadne wydarzenie</h2>
					<p class="body-medium">Twój bohater czeka na kolejny Dzień Otwarty. W międzyczasie poznaj uczelnię.</p>
					<Button variant="tonal" icon="explore" href={href('/explore')}>Odkrywaj Politechnikę</Button>
				</div>
			</Card>
		{/if}

		<Card variant="primary">
			<div class="level">
				<div class="lvl-badge pixel" aria-label="Poziom {p?.level ?? 1}">
					<span class="label-small">LVL</span>
					<span class="lvl-num">{p?.level ?? 1}</span>
				</div>
				<div class="lvl-main">
					<div class="row">
						<span class="title-small">Doświadczenie</span>
						<span class="spacer"></span>
						<span class="pixel label-large">{Math.round(p?.xp ?? 0)} XP</span>
					</div>
					<LinearProgress value={p?.levelProgress.ratio ?? 0} label="Postęp do następnego poziomu" />
					<span class="body-small">Do poziomu {(p?.level ?? 1) + 1}: {Math.max(0, Math.ceil((p?.levelProgress.next ?? 25) - (p?.xp ?? 0)))} XP</span>
				</div>
			</div>
		</Card>

		{#if p}
			{@const need = p.minScansForCard}
			<Card variant={p.cardUnlocked ? 'tertiary' : 'outlined'} href={href(p.cardUnlocked ? '/card' : '/quests')}>
				<div class="row">
					<Icon name={p.cardUnlocked ? 'workspace_premium' : 'lock'} />
					<div class="grow stack tight">
						<span class="title-small">{p.cardUnlocked ? 'Twoja karta postaci jest gotowa!' : 'Karta postaci'}</span>
						{#if !p.cardUnlocked}
							<LinearProgress value={p.ratedCount / need} label="Postęp do karty postaci" />
							<span class="body-small muted">{p.ratedCount}/{need} ocenionych aktywności</span>
						{:else}
							<span class="body-small">Zobacz swoją ścieżkę i rekomendacje kierunków.</span>
						{/if}
					</div>
					<Icon name="chevron_right" />
				</div>
			</Card>
		{/if}

		{#if pending.length || app.queue.length}
			<section class="stack" aria-labelledby="pending-h">
				<h2 id="pending-h" class="section-title">Do oceny</h2>
				{#each pending as s (s.id)}
					<Card variant="filled" padding={false}>
						<ListItem headline={s.stationName} supporting="Oceń, aby zdobyć punkty" icon="star" href={href('/scan/rate?id=' + s.id)} />
					</Card>
				{/each}
				{#each app.queue as q (q.id)}
					<Card variant="outlined" padding={false}>
						<ListItem headline="Skan zapisany offline" supporting={q.rating ? `Ocena ${q.rating}/5 — wyślemy po odzyskaniu zasięgu` : 'Wyślemy po odzyskaniu zasięgu'} icon="cloud_upload" />
					</Card>
				{/each}
			</section>
		{/if}

		<section class="stack" aria-labelledby="stats-h">
			<div class="row">
				<h2 id="stats-h" class="section-title">Statystyki</h2>
				<span class="spacer"></span>
				{#if p?.archetypes.length}
					<span class="label-medium muted path">
						{#each p.archetypes as a, i (a.code)}
							{i > 0 ? ' · ' : ''}{a.percent}% {statByCode[a.code].archetype}
						{/each}
					</span>
				{/if}
			</div>
			<Card variant="filled">
				<div class="stats">
					{#each ranked as code (code)}
						<div animate:flip={{ duration: 400 }}>
							<StatBar {code} value={stats[code]} {max} delta={deltas?.[code] ?? 0} />
						</div>
					{/each}
				</div>
			</Card>
			{#if (p?.xp ?? 0) === 0}
				<p class="body-medium muted hint">
					<Icon name="lightbulb" size={18} /> Wszystkie statystyki są na zero. Zeskanuj kod na koniec warsztatu lub pokazu, a Twoja ocena zdecyduje, ile punktów dostaniesz.
				</p>
			{/if}
		</section>

		{#if p?.badges.length}
			<section class="stack" aria-labelledby="badges-h">
				<h2 id="badges-h" class="section-title">Odznaki</h2>
				<div class="badges">
					{#each p.badges as b (b.id)}
						<div class="badge" title={BADGES[b.id]?.description} in:fly={{ y: 8, duration: 250 }}>
							<span class="bi"><Icon name={BADGES[b.id]?.icon ?? 'star'} /></span>
							<span class="label-medium">{BADGES[b.id]?.title ?? b.id}</span>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		{#if recent.length}
			<section class="stack" aria-labelledby="recent-h">
				<h2 id="recent-h" class="section-title">Ostatnie aktywności</h2>
				<Card variant="outlined" padding={false}>
					{#each recent as s (s.id)}
						{@const top = STAT_CODES.filter((c) => s.deltas[c] > 0).sort((a, b) => s.deltas[b] - s.deltas[a])[0]}
						<ListItem headline={s.stationName} supporting="Ocena {s.rating}/5 · +{Math.round(STAT_CODES.reduce((a, c) => a + s.deltas[c], 0))} XP{s.team ? ' · bonus drużynowy' : s.teamStatus === 'pending' ? ' · czekamy na drużynę' : ''}">
							{#snippet leading()}
								{#if top}<span class="lead-ic" style="color: {statByCode[top].color}"><StatIcon code={top} size={20} /></span>{/if}
							{/snippet}
						</ListItem>
					{/each}
				</Card>
			</section>
		{/if}

		<Button variant="tonal" icon="qr_code_scanner" href={href('/scan')} full>Skanuj kod stacji</Button>
	</div>
</AuthGate>

<style>
	.head {
		display: flex;
		align-items: center;
		gap: 4px;
		padding-top: calc(var(--uq-safe-top) + 8px);
	}
	.who {
		display: flex;
		align-items: center;
		gap: 16px;
		text-decoration: none;
		color: inherit;
		min-width: 0;
		padding: 6px;
	}
	.names {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.nick {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.level {
		display: flex;
		gap: 16px;
		align-items: center;
	}
	.lvl-badge {
		width: 64px;
		height: 64px;
		flex: none;
		border-radius: 16px;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		line-height: 1;
	}
	.lvl-num {
		font-size: 28px;
	}
	.lvl-main {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.grow {
		flex: 1;
	}
	.tight {
		gap: 4px;
	}
	.stats {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.path {
		text-align: right;
	}
	.hint {
		display: flex;
		gap: 8px;
		padding: 0 4px;
	}
	.hint :global(svg) {
		flex: none;
		margin-top: 2px;
	}
	.badges {
		display: flex;
		gap: 8px;
		overflow-x: auto;
		padding-bottom: 4px;
	}
	.badge {
		flex: none;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 6px;
		width: 96px;
		padding: 12px 8px;
		text-align: center;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-tertiary-container);
		color: var(--md-sys-color-on-tertiary-container);
	}
	.bi {
		width: 40px;
		height: 40px;
		border-radius: 50%;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-tertiary);
		color: var(--md-sys-color-on-tertiary);
	}
	.lead-ic {
		width: 40px;
		height: 40px;
		display: grid;
		place-items: center;
		border-radius: 12px;
		background: var(--md-sys-color-surface-container-high);
	}
</style>
