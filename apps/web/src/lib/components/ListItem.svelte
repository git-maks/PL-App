<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from '#lib/icons/paths';
	import Icon from './Icon.svelte';

	let {
		headline,
		supporting,
		overline,
		icon,
		href,
		external = false,
		trailing,
		leading,
		onclick
	}: {
		headline: string;
		supporting?: string;
		overline?: string;
		icon?: IconName;
		href?: string;
		external?: boolean;
		trailing?: Snippet;
		leading?: Snippet;
		onclick?: () => void;
	} = $props();
</script>

{#snippet inner()}
	{#if leading}
		<span class="lead">{@render leading()}</span>
	{:else if icon}
		<span class="lead icon"><Icon name={icon} /></span>
	{/if}
	<span class="text">
		{#if overline}<span class="label-small muted overline">{overline}</span>{/if}
		<span class="body-large headline">{headline}</span>
		{#if supporting}<span class="body-medium muted supporting">{supporting}</span>{/if}
	</span>
	{#if trailing}
		<span class="trail">{@render trailing()}</span>
	{:else if href}
		<span class="trail muted"><Icon name={external ? 'open_in_new' : 'chevron_right'} size={20} /></span>
	{/if}
{/snippet}

{#if href}
	<a class="li interactive" {href} target={external ? '_blank' : undefined} rel={external ? 'noopener' : undefined}>{@render inner()}</a>
{:else if onclick}
	<button class="li interactive" type="button" {onclick}>{@render inner()}</button>
{:else}
	<div class="li">{@render inner()}</div>
{/if}

<style>
	.li {
		display: flex;
		align-items: center;
		gap: 16px;
		width: 100%;
		min-height: 60px;
		padding: 12px 20px;
		border: none;
		background: transparent;
		color: var(--md-sys-color-on-surface);
		text-decoration: none;
		text-align: left;
		border-radius: var(--md-sys-shape-corner-medium);
	}
	.lead {
		flex: none;
		display: grid;
		place-items: center;
	}
	.icon {
		color: var(--md-sys-color-on-surface-variant);
	}
	.text {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}
	.headline {
		overflow-wrap: anywhere;
	}
	.trail {
		flex: none;
		display: flex;
		align-items: center;
	}
</style>
