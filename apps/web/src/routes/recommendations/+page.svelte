<script lang="ts">
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { statByCode } from '#lib/content/archetypes';
	import { programBySlug, clubBySlug, facultyByCode } from '#lib/content';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';
	import type { Character, Recommendation } from '#lib/api';

	let character = $state<Character | null>(null);
	let error = $state<string | null>(null);
	$effect(() => {
		if (app.me && app.event && !character) {
			app.api.character().then((c) => (character = c)).catch((e) => (error = errorMessage(e)));
		}
	});

	function why(r: Recommendation): string {
		const names = r.because.map((c) => statByCode[c].name.toLowerCase());
		if (names.length === 0) return '';
		return `Pasuje do Twojej ${names.join(' i ')}.`;
	}
</script>

<svelte:head><title>Rekomendacje · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Twoje rekomendacje" back={true} />
	<div class="page">
		{#if error}
			<p role="alert">{error}</p>
		{:else if !character}
			<Skeleton lines={6} />
		{:else if character.programs.length === 0}
			<Card variant="tertiary"><p class="body-medium">Oceń kilka aktywności, a dopasujemy kierunki i koła naukowe do Twojego profilu.</p></Card>
		{:else}
			{#if !character.unlocked}
				<Card variant="tertiary"><p class="body-medium">To wstępne dopasowanie — stanie się dokładniejsze z każdą oceną.</p></Card>
			{/if}
			<p class="body-medium muted">Dopasowanie liczymy, porównując Twoje statystyki z profilem kierunku (podobieństwo cosinusowe). Procent pokazuje, jak bardzo się pokrywają.</p>

			<h2 class="section-title">Kierunki studiów</h2>
			{#each character.programs as r, i (r.slug)}
				{@const prog = programBySlug[r.slug]}
				{#if prog}
					<Card variant={i === 0 ? 'primary' : 'filled'}>
						<div class="rec">
							<div class="score pixel" aria-label="Dopasowanie {r.score}%">{r.score}%</div>
							<div class="grow stack tight">
								<span class="label-small muted">{facultyByCode[prog.faculty]?.name}</span>
								<h3 class="title-medium">{prog.name}</h3>
								<p class="body-small">{why(r)}</p>
								<div class="row wrap">
									{#each r.because as c (c)}
										<span class="stat-chip label-small" style="--c: {statByCode[c].color}"><StatIcon code={c} size={14} />{statByCode[c].name}</span>
									{/each}
								</div>
								<div class="row wrap">
									<Button variant="text" trailingIcon="chevron_right" href={href('/explore/kierunki/' + prog.slug)}>Więcej o kierunku</Button>
									<Button variant="text" trailingIcon="open_in_new" href={prog.recruitmentUrl} target="_blank" rel="noopener">Rekrutacja</Button>
								</div>
							</div>
						</div>
					</Card>
				{/if}
			{/each}

			<h2 class="section-title">Koła naukowe</h2>
			{#each character.clubs as r (r.slug)}
				{@const club = clubBySlug[r.slug]}
				{#if club}
					<Card variant="outlined" href={href('/explore/kola/' + club.slug)}>
						<div class="rec">
							<div class="score small pixel">{r.score}%</div>
							<div class="grow">
								<span class="label-small muted">{facultyByCode[club.faculty]?.name}</span>
								<h3 class="title-small">{club.name}</h3>
								<p class="body-small muted">{club.tags.join(' · ')}</p>
							</div>
						</div>
					</Card>
				{/if}
			{/each}
		{/if}
	</div>
</AuthGate>

<style>
	.rec {
		display: flex;
		gap: 16px;
		align-items: flex-start;
	}
	.score {
		flex: none;
		width: 64px;
		height: 64px;
		border-radius: 16px;
		display: grid;
		place-items: center;
		font-size: 20px;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.score.small {
		width: 52px;
		height: 52px;
		font-size: 16px;
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
	.grow {
		flex: 1;
		min-width: 0;
	}
	.tight {
		gap: 4px;
	}
</style>
