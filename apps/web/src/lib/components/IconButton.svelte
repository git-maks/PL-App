<script lang="ts">
	import type { IconName } from '#lib/icons/paths';
	import Icon from './Icon.svelte';

	let {
		icon,
		label,
		variant = 'standard',
		href,
		disabled = false,
		selected,
		onclick
	}: {
		icon: IconName;
		label: string;
		variant?: 'standard' | 'filled' | 'tonal' | 'outlined';
		href?: string;
		disabled?: boolean;
		selected?: boolean;
		onclick?: (e: MouseEvent) => void;
	} = $props();
</script>

{#if href}
	<a class="ib interactive {variant}" {href} aria-label={label} title={label}><Icon name={icon} /></a>
{:else}
	<button
		class="ib interactive {variant}"
		type="button"
		aria-label={label}
		title={label}
		aria-pressed={selected}
		{disabled}
		{onclick}><Icon name={icon} /></button
	>
{/if}

<style>
	.ib {
		display: inline-grid;
		place-items: center;
		width: 48px;
		height: 48px;
		border-radius: 50%;
		border: none;
		background: transparent;
		color: var(--md-sys-color-on-surface-variant);
		text-decoration: none;
		flex: none;
	}
	.filled {
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.tonal {
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
	.outlined {
		border: 1px solid var(--md-sys-color-outline);
	}
	.ib[aria-pressed='true'] {
		color: var(--md-sys-color-primary);
	}
	.ib:disabled {
		opacity: 0.38;
	}
</style>
