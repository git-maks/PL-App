<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { pixelateFile } from '#lib/avatar/pixelate';
	import { AVATAR_PRESETS, renderPreset } from '#lib/avatar/presets';
	import { prefetchDetector } from '#lib/scanner/detector';
	import { STAT_CODES } from '#lib/game/types';
	import { emptyStats } from '#lib/game/scoring';
	import Button from '#lib/components/Button.svelte';
	import TextField from '#lib/components/TextField.svelte';
	import Switch from '#lib/components/Switch.svelte';
	import StatBar from '#lib/components/StatBar.svelte';
	import PixelAvatar from '#lib/components/PixelAvatar.svelte';
	import LinearProgress from '#lib/components/LinearProgress.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import InstallCard from '#lib/components/InstallCard.svelte';
	import Icon from '#lib/components/Icon.svelte';

	const next = $derived(page.url.searchParams.get('next'));
	let step = $state(0);
	let dir = $state(1);
	let nickname = $state('');
	let terms = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);

	let photo: File | null = $state(null);
	let preview = $state<string | null>(null);
	let removeBackground = $state(true);
	let dither = $state(false);
	let zoom = $state(1.2);
	let photoInput: HTMLInputElement | undefined = $state();
	let galleryInput: HTMLInputElement | undefined = $state();
	let preset = $state<string | null>(null);
	const presetImages = $derived(step === 2 ? AVATAR_PRESETS.map((p) => ({ ...p, src: renderPreset(p) })) : []);

	function pickPreset(id: string, src: string) {
		photo = null;
		preset = id;
		preview = src;
	}

	const STEPS = 4;
	const stats = emptyStats();

	// Returning players who already have a profile skip straight to the avatar step.
	$effect(() => {
		if (app.ready && app.me && step === 0) {
			nickname = app.me.nickname;
			terms = true;
			step = 2;
		}
	});

	function go(to: number) {
		dir = to > step ? 1 : -1;
		error = null;
		step = to;
	}

	async function createProfile(e: SubmitEvent) {
		e.preventDefault();
		if (!terms) {
			error = 'Zaakceptuj zasady gry, aby kontynuować.';
			return;
		}
		busy = true;
		error = null;
		try {
			const nick = nickname.trim();
			if (!app.me) await app.startAsGuest(nick || undefined);
			const me = await app.api.updateMe({ ...(nick ? { nickname: nick } : {}), consents: { terms: true, notifications: false } });
			await app.setMe(me);
			prefetchDetector();
			go(2);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	async function render() {
		if (!photo) return;
		try {
			preview = await pixelateFile(photo, { removeBackground, dither, zoom });
		} catch {
			error = 'Nie udało się odczytać zdjęcia. Spróbuj innego.';
		}
	}
	function onFile(e: Event) {
		const f = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!f) return;
		preset = null;
		photo = f; // the $effect below renders the preview
	}
	$effect(() => {
		// Re-render when an option changes.
		void removeBackground;
		void dither;
		void zoom;
		if (photo) void render();
	});

	async function saveAvatar() {
		if (!preview) return go(3);
		busy = true;
		try {
			await app.setMe(await app.api.uploadAvatar(preview));
			photo = null; // the photo is discarded; only the 48×48 pixel art is kept
			go(3);
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}

	function finish() {
		void goto(next && next.startsWith('/') ? next : href('/hero'), { replaceState: true });
	}
</script>

<svelte:head><title>Nowy bohater · UniQuest</title></svelte:head>

<div class="ob">
	<div class="top">
		{#if step > 0 && step < 3}
			<IconButton icon="arrow_back" label="Wstecz" onclick={() => go(step - 1)} />
		{:else}
			<span class="ph"></span>
		{/if}
		<div class="progress"><LinearProgress value={(step + 1) / STEPS} label="Postęp tworzenia bohatera" /></div>
		<span class="label-medium muted">{step + 1}/{STEPS}</span>
	</div>

	{#key step}
		<section class="step" in:fly={{ x: 40 * dir, duration: 300, easing: cubicOut }}>
			{#if step === 0}
				<img class="robot" src={href('/icons/icon.svg')} alt="" width="120" height="120" />
				<h1 class="headline-medium">Witaj w UniQuest!</h1>
				<p class="body-large muted">
					Każdy warsztat, pokaz i laboratorium na kampusie Politechniki Lubelskiej to zadanie. Na koniec zajęć zeskanujesz kod
					QR i powiesz, jak bardzo Ci się podobało. Z tego zbudujemy Twoją postać.
				</p>
				<div class="spacer"></div>
				<Button full icon="arrow_forward" onclick={() => go(1)}>Stwórz bohatera</Button>
			{:else if step === 1}
				<h1 class="headline-medium">Jak mamy Cię nazywać?</h1>
				<p class="body-medium muted">
					Pseudonim zobaczą członkowie Twojej drużyny i ranking. Nie podawaj imienia i nazwiska – e-mail też nie jest potrzebny.
				</p>
				<form class="stack" onsubmit={createProfile}>
					<TextField
						label="Pseudonim (opcjonalnie)"
						bind:value={nickname}
						maxlength={24}
						autocomplete="off"
						supporting="Zostaw puste, a wylosujemy coś epickiego."
					/>
					<label class="consent body-medium">
						<input type="checkbox" bind:checked={terms} />
						<span>
							Akceptuję <a href={href('/legal')} target="_blank">zasady gry i politykę prywatności</a>. Grę może prowadzić
							osoba w każdym wieku – konto gościa nie zawiera danych osobowych.
						</span>
					</label>
					{#if error}<p class="error body-medium" role="alert">{error}</p>{/if}
					<Button full type="submit" loading={busy} icon="check">Dalej</Button>
				</form>
			{:else if step === 2}
				<h1 class="headline-medium">Twój awatar</h1>
				<p class="body-medium muted">Wybierz gotową postać albo zrób selfie, a zamienimy je w pixel-art.</p>
				<div class="avatar-box">
					<PixelAvatar src={preview ?? app.me?.avatarUrl} seed={app.me?.id ?? 'x'} size={144} label="Podgląd awatara" />
				</div>
				<div class="presets" role="radiogroup" aria-label="Gotowe postacie">
					{#each presetImages as p (p.id)}
						<button
							type="button"
							role="radio"
							aria-checked={preset === p.id}
							aria-label="Postać: {p.label}"
							class="preset interactive"
							class:on={preset === p.id}
							onclick={() => pickPreset(p.id, p.src)}
						>
							<img src={p.src} alt="" width="48" height="48" />
						</button>
					{/each}
				</div>
				<p class="body-small muted">Zdjęcie przetwarzamy wyłącznie na Twoim telefonie i od razu je odrzucamy – do gry trafia tylko obrazek 48×48 pikseli.</p>
				<input bind:this={photoInput} class="visually-hidden" type="file" accept="image/*" capture="user" onchange={onFile} />
				<input bind:this={galleryInput} class="visually-hidden" type="file" accept="image/*" onchange={onFile} />
				<div class="row wrap center">
					<Button icon="photo_camera" variant="tonal" onclick={() => photoInput?.click()}>Zrób selfie</Button>
					<Button icon="image" variant="outlined" onclick={() => galleryInput?.click()}>Z galerii</Button>
				</div>
				{#if photo}
					<div class="opts">
						<Switch label="Usuń tło" bind:checked={removeBackground} />
						<Switch label="Efekt retro (dithering)" bind:checked={dither} />
						<label class="zoom body-medium">
							<span>Przybliżenie</span>
							<input type="range" min="1" max="2" step="0.05" bind:value={zoom} />
						</label>
					</div>
				{/if}
				{#if error}<p class="error body-medium" role="alert">{error}</p>{/if}
				<div class="spacer"></div>
				<Button full icon="check" loading={busy} onclick={saveAvatar}>{preview ? 'Zapisz awatar' : 'Pomiń – użyję pixelowego herbu'}</Button>
			{:else}
				<h1 class="headline-medium">Gotowe, {app.me?.nickname}!</h1>
				<p class="body-medium muted">
					Na razie wszystkie statystyki są na zero. Każda odwiedzona stacja doda punkty atrybutom, a Twoja ocena zdecyduje, jak
					mocno.
				</p>
				<div class="zero">
					{#each STAT_CODES as code (code)}
						<StatBar {code} value={stats[code]} max={100} />
					{/each}
				</div>
				<InstallCard />
				<div class="tip body-medium">
					<Icon name="lightbulb" />
					<span>Kody QR pojawiają się na ekranie pod koniec zajęć. Zbierz {app.event?.config.minScansForCard ?? 5} ocenionych aktywności, aby odblokować kartę postaci.</span>
				</div>
				<Button full icon="arrow_forward" onclick={finish}>Do panelu bohatera</Button>
			{/if}
		</section>
	{/key}
</div>

<style>
	.ob {
		min-height: 100dvh;
		max-width: 560px;
		margin: 0 auto;
		padding: calc(var(--uq-safe-top) + 8px) 24px calc(var(--uq-safe-bottom) + 24px);
		display: flex;
		flex-direction: column;
		overflow-x: hidden;
	}
	.top {
		display: flex;
		align-items: center;
		gap: 12px;
		height: 56px;
	}
	.ph {
		width: 48px;
	}
	.progress {
		flex: 1;
	}
	.step {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 16px;
		padding-top: 16px;
	}
	.robot {
		align-self: center;
		image-rendering: pixelated;
		border-radius: 28px;
		margin: 16px 0;
		box-shadow: var(--md-sys-elevation-2);
	}
	.spacer {
		flex: 1;
		min-height: 16px;
	}
	.consent {
		display: flex;
		gap: 12px;
		align-items: flex-start;
	}
	.consent input {
		width: 20px;
		height: 20px;
		margin-top: 2px;
		accent-color: var(--md-sys-color-primary);
		flex: none;
	}
	.error {
		color: var(--md-sys-color-error);
	}
	.avatar-box {
		display: grid;
		place-items: center;
		padding: 8px 0;
	}
	.presets {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 88px));
		justify-content: center;
		gap: 12px;
	}
	.preset {
		aspect-ratio: 1;
		padding: 0;
		border: 2px solid transparent;
		border-radius: var(--md-sys-shape-corner-medium);
		overflow: hidden;
		background: none;
		transition: transform var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard);
	}
	.preset img {
		width: 100%;
		height: 100%;
		display: block;
		image-rendering: pixelated;
	}
	.preset.on {
		border-color: var(--md-sys-color-primary);
		box-shadow: 0 0 0 2px var(--md-sys-color-primary);
		transform: scale(1.04);
	}
	.center {
		justify-content: center;
	}
	.opts {
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
	}
	.zoom {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 8px 16px 16px;
	}
	.zoom input {
		flex: 1;
		accent-color: var(--md-sys-color-primary);
	}
	.zero {
		display: flex;
		flex-direction: column;
		gap: 10px;
		padding: 16px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
	}
	.tip {
		display: flex;
		gap: 12px;
		padding: 12px 16px;
		border-radius: var(--md-sys-shape-corner-medium);
		background: var(--md-sys-color-tertiary-container);
		color: var(--md-sys-color-on-tertiary-container);
	}
</style>
