<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from '#lib/icons/paths';
	import Icon from './Icon.svelte';

	type Variant = 'filled' | 'tonal' | 'outlined' | 'text' | 'elevated' | 'danger';
	let {
		variant = 'filled',
		icon,
		trailingIcon,
		href,
		type = 'button',
		disabled = false,
		loading = false,
		full = false,
		onclick,
		children,
		...rest
	}: {
		variant?: Variant;
		icon?: IconName;
		trailingIcon?: IconName;
		href?: string;
		type?: 'button' | 'submit';
		disabled?: boolean;
		loading?: boolean;
		full?: boolean;
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
		[key: string]: unknown;
	} = $props();
</script>

{#if href && !disabled}
	<a class="btn interactive {variant}" class:full {href} {...rest}>
		{#if icon}<Icon name={icon} size={18} />{/if}
		<span>{@render children()}</span>
		{#if trailingIcon}<Icon name={trailingIcon} size={18} />{/if}
	</a>
{:else}
	<button
		class="btn interactive {variant}"
		class:full
		{type}
		disabled={disabled || loading}
		aria-busy={loading || undefined}
		{onclick}
		{...rest}
	>
		{#if loading}
			<span class="spinner" aria-hidden="true"></span>
		{:else if icon}
			<Icon name={icon} size={18} />
		{/if}
		<span>{@render children()}</span>
		{#if trailingIcon}<Icon name={trailingIcon} size={18} />{/if}
	</button>
{/if}

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		min-height: 40px;
		padding: 0 24px;
		border-radius: var(--md-sys-shape-corner-full);
		border: none;
		font: var(--md-sys-typescale-label-large);
		text-decoration: none;
		white-space: nowrap;
		transition:
			box-shadow var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard),
			transform var(--md-sys-motion-duration-short2) var(--md-sys-motion-easing-standard);
	}
	.btn:active {
		transform: scale(0.98);
	}
	.btn:has(:global(svg)) {
		padding-left: 16px;
	}
	.full {
		width: 100%;
		min-height: 48px;
	}
	.filled {
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.filled:hover {
		box-shadow: var(--md-sys-elevation-1);
	}
	.tonal {
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
	}
	.outlined {
		background: transparent;
		color: var(--md-sys-color-primary);
		border: 1px solid var(--md-sys-color-outline);
	}
	.text {
		background: transparent;
		color: var(--md-sys-color-primary);
		padding: 0 12px;
	}
	.elevated {
		background: var(--md-sys-color-surface-container-low);
		color: var(--md-sys-color-primary);
		box-shadow: var(--md-sys-elevation-1);
	}
	.danger {
		background: var(--md-sys-color-error);
		color: var(--md-sys-color-on-error);
	}
	.btn:disabled {
		background: color-mix(in srgb, var(--md-sys-color-on-surface) 12%, transparent);
		color: color-mix(in srgb, var(--md-sys-color-on-surface) 38%, transparent);
		box-shadow: none;
		border-color: transparent;
	}
	.spinner {
		width: 18px;
		height: 18px;
		border-radius: 50%;
		border: 2px solid currentColor;
		border-right-color: transparent;
		animation: spin 0.8s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
