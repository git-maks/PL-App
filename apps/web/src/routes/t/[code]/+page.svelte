<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import Button from '#lib/components/Button.svelte';
	import Card from '#lib/components/Card.svelte';
	import Icon from '#lib/components/Icon.svelte';

	const code = $derived(page.params.code ?? '');
	let busy = $state(false);
	let error = $state<string | null>(null);

	async function join() {
		busy = true;
		error = null;
		try {
			await app.joinTeam(code);
			await goto(href('/team'), { replaceState: true });
		} catch (e) {
			error = errorMessage(e);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Zaproszenie do drużyny · UniQuest</title></svelte:head>

<AuthGate reason="Aby dołączyć do drużyny, utwórz bohatera.">
	<div class="page center">
		<span class="ic"><Icon name="groups" size={48} /></span>
		<h1 class="headline-small">Zaproszenie do drużyny</h1>
		{#if app.team}
			<Card variant="tertiary"><p class="body-medium">Należysz już do drużyny „{app.team.name}”. Aby dołączyć do innej, najpierw ją opuść.</p></Card>
			<Button href={href('/team')}>Moja drużyna</Button>
		{:else}
			<p class="body-medium muted">Kod zaproszenia: <strong>{code}</strong></p>
			{#if error}<p class="error" role="alert">{error}</p>{/if}
			<Button icon="group_add" loading={busy} onclick={join}>Dołącz do drużyny</Button>
			<Button variant="text" href={href('/hero')}>Nie teraz</Button>
		{/if}
	</div>
</AuthGate>

<style>
	.center {
		align-items: center;
		text-align: center;
		padding-top: calc(var(--uq-safe-top) + 64px);
	}
	.ic {
		width: 96px;
		height: 96px;
		border-radius: 28px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.error {
		color: var(--md-sys-color-error);
	}
</style>
