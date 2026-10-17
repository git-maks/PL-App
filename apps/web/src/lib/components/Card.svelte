<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		variant = 'filled',
		href,
		padding = true,
		onclick,
		children,
		class: klass = ''
	}: {
		variant?: 'filled' | 'elevated' | 'outlined' | 'primary' | 'tertiary';
		href?: string;
		padding?: boolean;
		onclick?: () => void;
		children: Snippet;
		class?: string;
	} = $props();
</script>

{#if href}
	<a class="card interactive {variant} {klass}" class:padding {href}>{@render children()}</a>
{:else if onclick}
	<button class="card interactive {variant} {klass}" class:padding type="button" {onclick}>{@render children()}</button>
{:else}
	<div class="card {variant} {klass}" class:padding>{@render children()}</div>
{/if}

<style>
	.card {
		display: block;
		width: 100%;
		text-align: left;
		border: none;
		border-radius: var(--md-sys-shape-corner-large);
		color: var(--md-sys-color-on-surface);
		text-decoration: none;
		position: relative;
	}
	.padding {
		padding: 20px;
	}
	.filled {
		/* Read by text fields inside the card, so their floating label matches the background. */
		--uq-container: var(--md-sys-color-surface-container-highest);
		background: var(--uq-container);
	}
	.elevated {
		--uq-container: var(--md-sys-color-surface-container-low);
		background: var(--uq-container);
		box-shadow: var(--md-sys-elevation-1);
	}
	.outlined {
		--uq-container: var(--md-sys-color-surface);
		background: var(--uq-container);
		border: 1px solid var(--md-sys-color-outline-variant);
	}
	.primary {
		--uq-container: var(--md-sys-color-primary-container);
		background: var(--uq-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.tertiary {
		--uq-container: var(--md-sys-color-tertiary-container);
		background: var(--uq-container);
		color: var(--md-sys-color-on-tertiary-container);
	}
</style>
