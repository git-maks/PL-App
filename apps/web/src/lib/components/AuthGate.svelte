<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import { app } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import Button from './Button.svelte';
	import Card from './Card.svelte';
	import Skeleton from './Skeleton.svelte';

	let { children, reason = 'Ta część wymaga profilu gracza.' }: { children: Snippet; reason?: string } = $props();
</script>

{#if !app.ready}
	<div class="page"><Skeleton lines={4} /></div>
{:else if !app.me}
	<div class="page">
		<Card variant="primary">
			<div class="stack">
				<h2 class="title-large">Zacznij przygodę</h2>
				<p class="body-medium">{reason} Profil gościa założysz w kilka sekund – bez e-maila i hasła.</p>
				<Button icon="play_arrow" href={href('/onboarding?next=' + encodeURIComponent(page.url.pathname + page.url.search))}>Utwórz bohatera</Button>
			</div>
		</Card>
	</div>
{:else}
	{@render children()}
{/if}
