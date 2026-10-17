<script lang="ts">
	// Entry point when a station QR code is opened with the phone's own camera app.
	import { goto } from '$app/navigation';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import LinearProgress from '#lib/components/LinearProgress.svelte';
	import Button from '#lib/components/Button.svelte';

	let error = $state<string | null>(null);
	let started = false;

	$effect(() => {
		if (!app.ready || !app.me || started) return;
		started = true;
		void (async () => {
			const outcome = await app.submitScan(location.href);
			if (outcome.kind === 'ok') await goto(href('/scan/rate?id=' + outcome.result.scan.id), { replaceState: true });
			else if (outcome.kind === 'queued') await goto(href('/scan/rate?queued=' + outcome.item.id), { replaceState: true });
			else error = errorMessage(outcome.error);
		})();
	});
</script>

<svelte:head><title>Kod stacji · UniQuest</title></svelte:head>

<AuthGate reason="Zeskanowany kod stacji zaliczymy, gdy utworzysz bohatera.">
	<div class="page">
		{#if error}
			<h1 class="headline-small">Nie udało się zaliczyć stacji</h1>
			<p class="body-medium" role="alert">{error}</p>
			<Button icon="qr_code_scanner" href={href('/scan')}>Otwórz skaner</Button>
		{:else}
			<p class="title-medium">Sprawdzam kod stacji…</p>
			<LinearProgress indeterminate label="Sprawdzanie kodu" />
		{/if}
	</div>
</AuthGate>
