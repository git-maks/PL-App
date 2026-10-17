<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from '#lib/icons/paths';
	import Icon from './Icon.svelte';

	let {
		selected,
		icon,
		href,
		onclick,
		children
	}: { selected?: boolean; icon?: IconName; href?: string; onclick?: () => void; children: Snippet } = $props();
</script>

{#if href}
	<a class="chip interactive" {href}>
		{#if icon}<Icon name={icon} size={18} />{/if}{@render children()}
	</a>
{:else}
	<button class="chip interactive" class:selected type="button" aria-pressed={selected} {onclick}>
		{#if selected}<Icon name="check" size={18} />{:else if icon}<Icon name={icon} size={18} />{/if}
		{@render children()}
	</button>
{/if}

<style>
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		height: 32px;
		padding: 0 16px;
		border-radius: var(--md-sys-shape-corner-small);
		border: 1px solid var(--md-sys-color-outline);
		background: transparent;
		color: var(--md-sys-color-on-surface-variant);
		font: var(--md-sys-typescale-label-large);
		text-decoration: none;
		white-space: nowrap;
		flex: none;
	}
	.chip:has(:global(svg)) {
		padding-left: 8px;
	}
	.selected {
		background: var(--md-sys-color-secondary-container);
		color: var(--md-sys-color-on-secondary-container);
		border-color: transparent;
	}
</style>
