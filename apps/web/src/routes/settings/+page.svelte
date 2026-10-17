<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { version } from '$app/env';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { isDemo } from '#lib/api';
	import { toast } from '#lib/state/toast.svelte';
	import { enablePush, disablePush, pushPermission } from '#lib/pwa/push';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import TextField from '#lib/components/TextField.svelte';
	import Switch from '#lib/components/Switch.svelte';
	import Chip from '#lib/components/Chip.svelte';
	import Dialog from '#lib/components/Dialog.svelte';
	import ListItem from '#lib/components/ListItem.svelte';
	import PixelAvatar from '#lib/components/PixelAvatar.svelte';
	import InstallCard from '#lib/components/InstallCard.svelte';

	let nickname = $state(app.me?.nickname ?? '');
	let nickBusy = $state(false);
	let theme = $state<'system' | 'light' | 'dark'>('system');
	let reducedMotion = $state(false);
	let push = $state(false);

	let emailOpen = $state(false);
	let email = $state('');
	let ageOk = $state(false);
	let code = $state('');
	let devCode = $state<string | null>(null);
	let stage = $state<'email' | 'code'>('email');
	let emailError = $state<string | null>(null);
	let emailBusy = $state(false);
	let deleteOpen = $state(false);

	onMount(() => {
		try {
			const t = localStorage.getItem('uq.theme');
			theme = t === 'light' || t === 'dark' ? t : 'system';
			reducedMotion = localStorage.getItem('uq.motion') === 'reduced';
		} catch {
			/* storage unavailable – defaults apply */
		}
		push = pushPermission() === 'granted' && !!app.me?.consents.notifications;
	});
	$effect(() => {
		if (app.me && !nickname) nickname = app.me.nickname;
	});

	function setTheme(t: typeof theme) {
		theme = t;
		try {
			if (t === 'system') localStorage.removeItem('uq.theme');
			else localStorage.setItem('uq.theme', t);
		} catch {
			/* ignore */
		}
		if (t === 'system') delete document.documentElement.dataset.theme;
		else document.documentElement.dataset.theme = t;
	}
	function setMotion(v: boolean) {
		try {
			if (v) localStorage.setItem('uq.motion', 'reduced');
			else localStorage.removeItem('uq.motion');
		} catch {
			/* ignore */
		}
		if (v) document.documentElement.dataset.motion = 'reduced';
		else delete document.documentElement.dataset.motion;
	}

	async function saveNick(e: SubmitEvent) {
		e.preventDefault();
		nickBusy = true;
		try {
			await app.setMe(await app.api.updateMe({ nickname }));
			toast('Pseudonim zapisany.');
		} catch (err) {
			toast(errorMessage(err));
		} finally {
			nickBusy = false;
		}
	}

	async function togglePush(on: boolean) {
		if (on) {
			const r = await enablePush(app.api).catch(() => 'unsupported' as const);
			push = r === 'enabled';
			if (r === 'denied') toast('Powiadomienia są zablokowane w ustawieniach przeglądarki.');
			else if (r === 'dismissed') return;
			else if (r === 'unsupported') toast('To urządzenie nie obsługuje powiadomień push. Na iPhonie zainstaluj aplikację na ekranie głównym.');
			else if (r === 'unavailable') toast('Powiadomienia push nie są dostępne w tej wersji – będziesz je widzieć w aplikacji.');
		} else {
			await disablePush(app.api);
			push = false;
		}
		if (app.me) await app.setMe(await app.api.updateMe({ consents: { ...app.me.consents, notifications: push } }));
	}

	async function sendCode(e: SubmitEvent) {
		e.preventDefault();
		if (!ageOk) {
			emailError = 'Potwierdź wiek lub zgodę opiekuna.';
			return;
		}
		emailBusy = true;
		emailError = null;
		try {
			const r = await app.api.emailStart(email.trim(), ageOk);
			devCode = r.devCode ?? null;
			stage = 'code';
		} catch (err) {
			emailError = errorMessage(err);
		} finally {
			emailBusy = false;
		}
	}
	async function verify(e: SubmitEvent) {
		e.preventDefault();
		emailBusy = true;
		emailError = null;
		try {
			await app.setMe(await app.api.emailVerify(email.trim(), code.trim()));
			await app.refresh();
			emailOpen = false;
			toast('Konto zapisane – możesz się zalogować na innym urządzeniu.');
		} catch (err) {
			emailError = errorMessage(err);
		} finally {
			emailBusy = false;
		}
	}

	async function exportData() {
		const data = await app.api.exportData();
		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = 'uniquest-moje-dane.json';
		a.click();
		setTimeout(() => URL.revokeObjectURL(a.href), 5000);
	}
	async function deleteAccount() {
		await app.api.deleteAccount();
		await app.logout().catch(() => {});
		deleteOpen = false;
		await goto(href('/'), { replaceState: true });
	}
	async function logout(all = false) {
		if (all) await app.api.logout();
		await app.logout();
		await goto(href('/'), { replaceState: true });
	}
	async function setDemoRole(on: boolean) {
		await app.api.setRole('', on ? 'admin' : 'participant');
		app.me = await app.api.me();
	}
</script>

<svelte:head><title>Ustawienia · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Profil i ustawienia" back={href('/hero')} />
	<div class="page">
		<Card variant="filled">
			<div class="row">
				<PixelAvatar src={app.me?.avatarUrl} seed={app.me?.id ?? ''} size={72} />
				<div class="grow">
					<p class="title-medium">{app.me?.nickname}</p>
					<p class="body-small muted">{app.me?.kind === 'registered' ? app.me.email : 'Konto gościa (tylko to urządzenie)'}</p>
				</div>
			</div>
			<div class="row wrap actions">
				<Button variant="tonal" icon="face" href={href('/onboarding?next=' + encodeURIComponent(href('/settings')))}>Zmień awatar</Button>
				{#if app.me?.avatarUrl}
					<Button variant="text" onclick={async () => app.setMe(await app.api.deleteAvatar())}>Usuń awatar</Button>
				{/if}
			</div>
		</Card>

		<form class="row" onsubmit={saveNick}>
			<div class="grow"><TextField label="Pseudonim" bind:value={nickname} maxlength={24} autocomplete="off" /></div>
			<Button type="submit" variant="tonal" loading={nickBusy} disabled={nickname.trim() === app.me?.nickname}>Zapisz</Button>
		</form>

		<h2 class="section-title">Konto</h2>
		{#if app.me?.kind === 'guest'}
			<Card variant="primary">
				<div class="stack">
					<p class="title-small">Zapisz postęp</p>
					<p class="body-medium">Dodaj e-mail, aby nie stracić bohatera przy zmianie telefonu. Wyślemy jednorazowy kod – bez haseł.</p>
					<Button icon="mail" onclick={() => ((emailOpen = true), (stage = 'email'))}>Dodaj e-mail</Button>
				</div>
			</Card>
		{/if}
		<Card variant="outlined" padding={false}>
			{#if app.me?.kind === 'registered'}
				<ListItem icon="logout" headline="Wyloguj" onclick={() => logout()} />
				<ListItem icon="shield" headline="Wyloguj na wszystkich urządzeniach" onclick={() => logout(true)} />
			{:else}
				<ListItem icon="account_circle" headline="Mam już konto – zaloguj e-mailem" supporting="Postęp z tego urządzenia zostanie połączony" onclick={() => ((emailOpen = true), (stage = 'email'))} />
			{/if}
		</Card>

		<h2 class="section-title">Powiadomienia</h2>
		<Card variant="outlined" padding={false}>
			<Switch label="Powiadomienia push" supporting="Przypomnienia o zajęciach i zdarzenia drużyny" checked={push} onchange={togglePush} />
		</Card>
		<InstallCard />

		<h2 class="section-title">Wygląd</h2>
		<Card variant="outlined">
			<div class="stack">
				<div class="row wrap" role="radiogroup" aria-label="Motyw">
					<Chip selected={theme === 'system'} icon="tune" onclick={() => setTheme('system')}>Systemowy</Chip>
					<Chip selected={theme === 'light'} icon="light_mode" onclick={() => setTheme('light')}>Jasny</Chip>
					<Chip selected={theme === 'dark'} icon="dark_mode" onclick={() => setTheme('dark')}>Ciemny</Chip>
				</div>
			</div>
			<Switch label="Ogranicz animacje" bind:checked={reducedMotion} onchange={setMotion} />
		</Card>

		{#if app.me?.role === 'staff' || app.me?.role === 'admin' || isDemo}
			<h2 class="section-title">Organizatorzy</h2>
			<Card variant="outlined" padding={false}>
				{#if isDemo}
					<Switch label="Tryb organizatora (demo)" supporting="Odblokowuje panel stacji i administratora" checked={app.me?.role === 'admin'} onchange={setDemoRole} />
				{/if}
				{#if app.me?.role === 'staff' || app.me?.role === 'admin'}
					<ListItem icon="qr_code_2" headline="Panel stacji" supporting="Wyświetl kod QR na koniec zajęć" href={href('/staff')} />
				{/if}
				{#if app.me?.role === 'admin'}
					<ListItem icon="admin_panel_settings" headline="Administracja" supporting="Statystyki i komunikaty" href={href('/admin')} />
				{/if}
				{#if isDemo}<ListItem icon="qr_code_2" headline="Arkusz stacji testowych" href={href('/demo/stations')} />{/if}
			</Card>
		{/if}

		<h2 class="section-title">Prywatność</h2>
		<Card variant="outlined" padding={false}>
			<ListItem icon="download" headline="Pobierz moje dane" supporting="Plik JSON ze wszystkimi danymi konta" onclick={exportData} />
			<ListItem icon="delete" headline="Usuń konto" supporting="Trwale usuwa bohatera, skany i drużynę" onclick={() => (deleteOpen = true)} />
			<ListItem icon="info" headline="Zasady gry i polityka prywatności" href={href('/legal')} />
		</Card>
		<p class="body-small muted center">UniQuest {isDemo ? 'demo' : ''} · wersja {version.slice(0, 12)}</p>
	</div>

	<Dialog bind:open={emailOpen} title={stage === 'email' ? 'Twój e-mail' : 'Wpisz kod z e-maila'}>
		{#if stage === 'email'}
			<form class="stack" onsubmit={sendCode}>
				<TextField type="email" label="Adres e-mail" bind:value={email} autocomplete="email" inputmode="email" autofocus />
				<label class="consent body-small">
					<input type="checkbox" bind:checked={ageOk} />
					<span>Mam ukończone 16 lat albo mój opiekun wyraził zgodę na podanie e-maila.</span>
				</label>
				{#if emailError}<p class="error body-small" role="alert">{emailError}</p>{/if}
				<Button type="submit" loading={emailBusy} disabled={!email.includes('@')}>Wyślij kod</Button>
			</form>
		{:else}
			<form class="stack" onsubmit={verify}>
				<p>Wysłaliśmy 6-cyfrowy kod na {email}. Jest ważny 10 minut.</p>
				{#if devCode}<p class="dev body-small">Tryb demo: Twój kod to <strong>{devCode}</strong></p>{/if}
				<TextField label="Kod" bind:value={code} inputmode="numeric" maxlength={6} autocomplete="one-time-code" autofocus />
				{#if emailError}<p class="error body-small" role="alert">{emailError}</p>{/if}
				<Button type="submit" loading={emailBusy} disabled={code.trim().length !== 6}>Potwierdź</Button>
				<Button variant="text" onclick={() => (stage = 'email')}>Zmień e-mail</Button>
			</form>
		{/if}
	</Dialog>

	<Dialog bind:open={deleteOpen} title="Usunąć konto?">
		<p>Tej operacji nie można cofnąć. Usuniemy pseudonim, awatar, e-mail, wszystkie skany i członkostwo w drużynie.</p>
		{#snippet actions()}
			<Button variant="text" onclick={() => (deleteOpen = false)}>Anuluj</Button>
			<Button variant="danger" icon="delete" onclick={deleteAccount}>Usuń na zawsze</Button>
		{/snippet}
	</Dialog>
</AuthGate>

<style>
	.grow {
		flex: 1;
		min-width: 0;
	}
	.actions {
		margin-top: 12px;
	}
	.consent {
		display: flex;
		gap: 10px;
		align-items: flex-start;
	}
	.consent input {
		width: 18px;
		height: 18px;
		accent-color: var(--md-sys-color-primary);
		flex: none;
	}
	.error {
		color: var(--md-sys-color-error);
	}
	.dev {
		padding: 8px 12px;
		border-radius: 8px;
		background: var(--md-sys-color-tertiary-container);
		color: var(--md-sys-color-on-tertiary-container);
	}
	.center {
		text-align: center;
	}
</style>
