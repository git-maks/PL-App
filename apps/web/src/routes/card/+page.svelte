<script lang="ts">
	import { onDestroy } from 'svelte';
	import { scale } from 'svelte/transition';
	import { backOut } from 'svelte/easing';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { statByCode, titleFor } from '#lib/content/archetypes';
	import { programBySlug } from '#lib/content';
	import { describePath } from '#lib/game/describe';
	import { renderCard, canvasToBlob, shareOrDownload } from '#lib/card/render';
	import { date } from '#lib/time';
	import { toast } from '#lib/state/toast.svelte';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import RadarChart from '#lib/components/RadarChart.svelte';
	import LinearProgress from '#lib/components/LinearProgress.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';
	import type { Character } from '#lib/api';

	let character = $state<Character | null>(null);
	let error = $state<string | null>(null);
	let revealed = $state(false);
	let preview = $state<string | null>(null);
	let blob: Blob | null = null;
	let busy = $state(false);

	$effect(() => {
		if (app.me && app.event && !character) {
			app.api
				.character()
				.then((c) => (character = c))
				.catch((e) => (error = errorMessage(e)));
		}
	});

	const title = $derived(titleFor(character?.progress.titleKey));
	const sentence = $derived(character ? describePath(character.progress.archetypes) : '');

	async function buildImage() {
		if (!character || !app.me) return;
		const canvas = await renderCard({
			nickname: app.me.nickname,
			avatarUrl: app.me.avatarUrl,
			seed: app.me.id,
			title: title?.title ?? 'Poszukiwacz',
			description: sentence,
			level: character.progress.level,
			xp: character.progress.xp,
			stats: character.progress.stats,
			archetypes: character.progress.archetypes,
			programs: character.programs.map((p) => programBySlug[p.slug]?.name ?? p.slug),
			eventName: app.event?.name ?? 'UniQuest',
			date: date(new Date())
		});
		blob = await canvasToBlob(canvas);
		if (preview) URL.revokeObjectURL(preview);
		preview = URL.createObjectURL(blob);
	}
	$effect(() => {
		if (revealed && character?.unlocked && !preview) void buildImage();
	});
	onDestroy(() => preview && URL.revokeObjectURL(preview));

	async function share() {
		busy = true;
		try {
			if (!blob) await buildImage();
			if (!blob) return;
			const r = await shareOrDownload(blob, 'uniquest-karta-postaci.png', `${sentence} #UniQuest #PolitechnikaLubelska`);
			if (r === 'downloaded') toast('Karta zapisana w pobranych plikach.');
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Karta postaci · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Karta postaci" back={href('/hero')} heading={false} />
	<div class="page">
		{#if error}
			<p class="body-medium" role="alert">{error}</p>
		{:else if !app.event}
			<Card variant="tertiary"><p class="body-medium">Karta postaci powstaje podczas wydarzenia UniQuest.</p></Card>
		{:else if !character}
			<Skeleton lines={6} />
		{:else if !character.unlocked}
			{@const p = character.progress}
			<div class="locked">
				<div class="lock-ic"><Icon name="lock" size={48} /></div>
				<h1 class="headline-small">Karta jeszcze zablokowana</h1>
				<p class="body-medium muted">Oceń jeszcze {p.minScansForCard - p.ratedCount} {p.minScansForCard - p.ratedCount === 1 ? 'aktywność' : 'aktywności'}, aby poznać swoją ścieżkę.</p>
				<div class="bar"><LinearProgress value={p.ratedCount / p.minScansForCard} label="Postęp do karty" /></div>
				<p class="label-large">{p.ratedCount}/{p.minScansForCard}</p>
				{#if p.archetypes.length}
					<Card variant="filled">
						<p class="body-medium">Na razie: <strong>{sentence}</strong></p>
					</Card>
				{/if}
				<Button icon="flag" href={href('/quests')}>Zobacz zadania</Button>
			</div>
		{:else if !revealed}
			<button class="reveal interactive" type="button" onclick={() => (revealed = true)}>
				<span class="pixel back-title">?</span>
				<span class="title-medium">Odkryj swoją kartę</span>
				<span class="body-small">Stuknij, aby odsłonić</span>
			</button>
		{:else}
			<div class="result" in:scale={{ start: 0.85, duration: 500, easing: backOut }}>
				<p class="label-large kicker">Twoja klasa postaci</p>
				<h1 class="pixel class-title">{title?.title}</h1>
				<p class="title-medium">{sentence}</p>
				<p class="body-medium muted">{title?.description}</p>
			</div>

			<Card variant="filled">
				<div class="split" role="img" aria-label={sentence}>
					{#each character.progress.archetypes as a (a.code)}
						<span style="flex: {a.percent}; background: {statByCode[a.code].color}"></span>
					{/each}
				</div>
				<ul class="arch">
					{#each character.progress.archetypes as a (a.code)}
						<li style="--c: {statByCode[a.code].color}">
							<span class="ai"><StatIcon code={a.code} size={20} /></span>
							<div class="grow">
								<p class="title-small">{a.percent}% {statByCode[a.code].archetype}</p>
								<p class="body-small muted">{statByCode[a.code].description}</p>
							</div>
						</li>
					{/each}
				</ul>
			</Card>

			<Card variant="outlined">
				<div class="radar"><RadarChart stats={character.progress.stats} size={300} /></div>
			</Card>

			{#if preview}
				<figure class="preview">
					<img src={preview} alt="Grafika karty postaci do udostępnienia" width="1080" height="1350" />
				</figure>
			{/if}
			<div class="actions">
				<Button full icon="share" loading={busy} onclick={share}>Udostępnij lub pobierz kartę</Button>
				<Button full variant="tonal" icon="school" href={href('/recommendations')}>Pasujące kierunki i koła naukowe</Button>
			</div>
		{/if}
	</div>
</AuthGate>

<style>
	.locked {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 12px;
		padding-top: 24px;
	}
	.lock-ic {
		width: 96px;
		height: 96px;
		border-radius: 28px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-surface-container-high);
		color: var(--md-sys-color-on-surface-variant);
	}
	.bar {
		width: 100%;
		max-width: 320px;
	}
	.reveal {
		align-self: center;
		width: min(100%, 320px);
		aspect-ratio: 4 / 5;
		margin-top: 16px;
		border: 6px solid var(--md-sys-color-tertiary);
		border-radius: 28px;
		background: linear-gradient(135deg, #0f2f5c, #1f5aa6);
		color: #fff;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 8px;
		box-shadow: var(--md-sys-elevation-3);
		animation: wiggle 2.4s ease-in-out infinite;
	}
	@keyframes wiggle {
		0%,
		100% {
			transform: rotate(0);
		}
		50% {
			transform: rotate(-1.5deg) translateY(-4px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.reveal {
			animation: none;
		}
	}
	.back-title {
		font-size: 96px;
		line-height: 1;
		color: #f5bf48;
	}
	.result {
		text-align: center;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding: 8px 0;
	}
	.kicker {
		color: var(--md-sys-color-primary);
	}
	.class-title {
		font-size: 2.5rem;
		line-height: 1.1;
		color: var(--md-sys-color-tertiary);
	}
	.split {
		display: flex;
		height: 16px;
		border-radius: 8px;
		overflow: hidden;
		gap: 2px;
		margin-bottom: 16px;
	}
	.arch {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}
	.arch li {
		display: flex;
		gap: 12px;
	}
	.ai {
		width: 36px;
		height: 36px;
		flex: none;
		border-radius: 10px;
		display: grid;
		place-items: center;
		color: var(--c);
		background: color-mix(in srgb, var(--c) 14%, var(--md-sys-color-surface));
	}
	.grow {
		flex: 1;
	}
	.radar {
		display: grid;
		place-items: center;
		padding: 16px 0;
	}
	.preview img {
		width: 100%;
		height: auto;
		border-radius: var(--md-sys-shape-corner-large);
		box-shadow: var(--md-sys-elevation-2);
	}
	.preview {
		margin: 0;
	}
	.actions {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
</style>
