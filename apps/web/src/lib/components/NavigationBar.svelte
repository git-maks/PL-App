<script lang="ts" module>
	import type { IconName } from '#lib/icons/paths';

	export interface NavItem {
		href: string;
		label: string;
		icon: IconName;
		active: boolean;
		badge?: number;
		primary?: boolean;
	}
</script>

<script lang="ts">
	import Icon from './Icon.svelte';

	let { items }: { items: NavItem[] } = $props();
</script>

<nav class="nav" aria-label="Główna nawigacja">
	<ul>
		{#each items as item (item.href)}
			<li>
				<a href={item.href} class="item" class:active={item.active} class:primary={item.primary} aria-current={item.active ? 'page' : undefined}>
					<span class="indicator interactive">
						<Icon name={item.icon} />
						{#if item.badge}<span class="badge" aria-label="{item.badge} nowych">{item.badge}</span>{/if}
					</span>
					<span class="label">{item.label}</span>
				</a>
			</li>
		{/each}
	</ul>
</nav>

<style>
	.nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 20;
		background: var(--md-sys-color-surface-container);
		padding-bottom: var(--uq-safe-bottom);
	}
	ul {
		list-style: none;
		margin: 0 auto;
		padding: 0 8px;
		max-width: var(--uq-max-width);
		height: var(--uq-nav-height);
		display: grid;
		grid-template-columns: repeat(5, 1fr);
	}
	.item {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 4px;
		color: var(--md-sys-color-on-surface-variant);
		text-decoration: none;
		font: var(--md-sys-typescale-label-medium);
	}
	.indicator {
		position: relative;
		width: 64px;
		height: 32px;
		border-radius: 16px;
		display: grid;
		place-items: center;
		overflow: visible;
	}
	.indicator::before {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: inherit;
		background: var(--md-sys-color-secondary-container);
		transform: scaleX(0.4);
		opacity: 0;
		transition:
			transform var(--md-sys-motion-duration-medium2) var(--md-sys-motion-easing-emphasized-decelerate),
			opacity var(--md-sys-motion-duration-short2) linear;
		z-index: -1;
	}
	.indicator::after {
		border-radius: inherit;
	}
	.active {
		color: var(--md-sys-color-on-surface);
	}
	.active .indicator {
		color: var(--md-sys-color-on-secondary-container);
	}
	.active .indicator::before {
		transform: scaleX(1);
		opacity: 1;
	}
	.primary .indicator {
		width: 56px;
		height: 40px;
		border-radius: 14px;
		margin-top: -8px;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
		box-shadow: var(--md-sys-elevation-2);
	}
	.primary .indicator::before {
		display: none;
	}
	.label {
		font-weight: 500;
	}
	.active .label {
		font-weight: 700;
	}
	.badge {
		position: absolute;
		top: 0;
		right: 14px;
		min-width: 16px;
		height: 16px;
		padding: 0 4px;
		border-radius: 8px;
		background: var(--md-sys-color-error);
		color: var(--md-sys-color-on-error);
		font: var(--md-sys-typescale-label-small);
		line-height: 16px;
		text-align: center;
	}
</style>
