<script lang="ts">
	import type { Snippet } from 'svelte';
	import IconButton from './IconButton.svelte';

	let {
		title,
		back,
		large = false,
		compact = false,
		heading = true,
		actions
	}: {
		title: string;
		/** href to go back to, or true to use history.back() */
		back?: string | true;
		large?: boolean;
		/** Small all-caps title in a low bar, for top-level tabs where the content speaks for itself. */
		compact?: boolean;
		/** Render the title as the page's <h1>. Set false when the page has its own h1. */
		heading?: boolean;
		actions?: Snippet;
	} = $props();

	let scrolled = $state(false);
	function onscroll() {
		scrolled = window.scrollY > 4;
	}
</script>

<svelte:window {onscroll} />

<header class="bar" class:scrolled class:large class:compact>
	<div class="inner">
		{#if back === true}
			<IconButton icon="arrow_back" label="Wstecz" onclick={() => history.back()} />
		{:else if back}
			<IconButton icon="arrow_back" label="Wstecz" href={back} />
		{/if}
		{#if large}<span class="spacer"></span>{:else if heading}<h1 class="title">{title}</h1>{:else}<span class="title" aria-hidden="true">{title}</span>{/if}
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
	{#if large}<h1 class="large-title">{title}</h1>{/if}
</header>

<style>
	.bar {
		position: sticky;
		top: 0;
		z-index: 10;
		background: var(--md-sys-color-surface);
		padding-top: var(--uq-safe-top);
		transition: background var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard);
	}
	.scrolled {
		background: var(--md-sys-color-surface-container);
	}
	.inner {
		max-width: var(--uq-max-width);
		margin: 0 auto;
		height: var(--uq-top-bar-height);
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 0 4px 0 8px;
	}
	.title {
		flex: 1;
		font: var(--md-sys-typescale-title-large);
		margin-left: 8px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.compact .inner {
		height: 48px;
		padding: 0 8px 0 calc(var(--uq-gutter) - 8px);
	}
	.compact .title {
		font: var(--md-sys-typescale-label-large);
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--md-sys-color-on-surface-variant);
	}
	.spacer {
		flex: 1;
	}
	.actions {
		display: flex;
		align-items: center;
	}
	.large-title {
		max-width: var(--uq-max-width);
		margin: 0 auto;
		padding: 4px var(--uq-gutter) 12px;
		font: var(--md-sys-typescale-headline-medium);
	}
</style>
