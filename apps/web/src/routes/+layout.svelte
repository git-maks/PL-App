<script lang="ts">
	import '#lib/design/tokens.css';
	import '#lib/design/base.css';
	import '@fontsource/pixelify-sans/latin-ext-500.css';
	import '@fontsource/pixelify-sans/latin-500.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { onNavigate } from '$app/navigation';
	import { browser } from '$app/env';
	import NavigationBar, { type NavItem } from '#lib/components/NavigationBar.svelte';
	import Snackbar from '#lib/components/Snackbar.svelte';
	import { app } from '#lib/state/app.svelte';
	import { href, routePath } from '#lib/paths';
	import { registerServiceWorker } from '#lib/pwa/register';

	let { children } = $props();

	const path = $derived(routePath(page.url.pathname));
	const chromeless = $derived(
		path.startsWith('/onboarding') || path.startsWith('/scan') || path.startsWith('/staff/display') || path.startsWith('/staff/poster') || path === '/' || path.startsWith('/q/') || path.startsWith('/t/')
	);

	const items: NavItem[] = $derived([
		{ href: href('/hero'), label: 'Bohater', icon: 'person', active: path.startsWith('/hero') || path.startsWith('/card') || path.startsWith('/recommendations') || path.startsWith('/settings') || path.startsWith('/notifications'), badge: app.unread || undefined },
		{ href: href('/quests'), label: 'Zadania', icon: 'flag', active: path.startsWith('/quests') || path.startsWith('/map'), badge: app.pendingRatings || undefined },
		{ href: href('/scan'), label: 'Skanuj', icon: 'qr_code_scanner', active: false, primary: true },
		{ href: href('/team'), label: 'Drużyna', icon: 'groups', active: path.startsWith('/team') },
		{ href: href('/explore'), label: 'Odkrywaj', icon: 'explore', active: path.startsWith('/explore') }
	]);

	onMount(() => {
		void app.init();
		registerServiceWorker();
	});

	// Smooth, interruptible page transitions where the View Transitions API exists.
	onNavigate((navigation) => {
		if (!browser || !document.startViewTransition) return;
		if (matchMedia('(prefers-reduced-motion: reduce)').matches || document.documentElement.dataset.motion === 'reduced') return;
		return new Promise((resolve) => {
			document.startViewTransition(async () => {
				resolve();
				await navigation.complete;
			});
		});
	});
</script>

<svelte:head>
	<title>UniQuest · Politechnika Lubelska</title>
</svelte:head>

<a class="skip" href="#main">Przejdź do treści</a>
<main id="main">
	{@render children()}
</main>

{#if !chromeless}
	<NavigationBar {items} />
{/if}
<Snackbar />

{#if !app.online}
	<div class="offline label-medium" role="status">Tryb offline — skany zapiszemy i wyślemy później</div>
{/if}

<style>
	.skip {
		position: absolute;
		left: 8px;
		top: -48px;
		z-index: 100;
		padding: 8px 16px;
		border-radius: var(--md-sys-shape-corner-full);
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.skip:focus {
		top: 8px;
	}
	main {
		min-height: 100dvh;
	}
	.offline {
		position: fixed;
		top: calc(var(--uq-safe-top) + 8px);
		left: 50%;
		transform: translateX(-50%);
		z-index: 60;
		padding: 6px 14px;
		border-radius: var(--md-sys-shape-corner-full);
		background: var(--md-sys-color-inverse-surface);
		color: var(--md-sys-color-inverse-on-surface);
		box-shadow: var(--md-sys-elevation-2);
		white-space: nowrap;
	}
</style>
