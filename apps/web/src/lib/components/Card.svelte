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
		padding: 16px;
	}
	.filled {
		background: var(--md-sys-color-surface-container-highest);
	}
	.elevated {
		background: var(--md-sys-color-surface-container-low);
		box-shadow: var(--md-sys-elevation-1);
	}
	.outlined {
		background: var(--md-sys-color-surface);
		border: 1px solid var(--md-sys-color-outline-variant);
	}
	.primary {
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.tertiary {
		background: var(--md-sys-color-tertiary-container);
		color: var(--md-sys-color-on-tertiary-container);
	}
</style>
