<script lang="ts">
	import { Tween } from 'svelte/motion';
	import { cubicOut } from 'svelte/easing';
	import type { StatCode } from '#lib/game/types';
	import { statByCode } from '#lib/content/archetypes';
	import StatIcon from './StatIcon.svelte';

	let {
		code,
		value,
		max,
		delta = 0,
		segments = 10
	}: { code: StatCode; value: number; max: number; delta?: number; segments?: number } = $props();

	const stat = $derived(statByCode[code]);
	const shown = new Tween(0, { duration: 700, easing: cubicOut });
	$effect(() => {
		shown.target = value;
	});
	const ratio = $derived(max > 0 ? Math.min(1, shown.current / max) : 0);
</script>

<div class="stat" style="--c: {stat.color}">
	<span class="icon"><StatIcon {code} size={20} /></span>
	<div class="main">
		<div class="row">
			<span class="name label-large">{stat.name}</span>
			<span class="arch label-small muted">{stat.archetype}</span>
			<span class="spacer"></span>
			{#if delta > 0}<span class="delta num label-medium">+{Math.round(delta)}</span>{/if}
			<span class="value num title-small">{Math.round(shown.current)}</span>
		</div>
		<div
			class="bar"
			role="meter"
			aria-label="{stat.name} ({stat.archetype})"
			aria-valuemin="0"
			aria-valuemax={Math.max(1, Math.round(max))}
			aria-valuenow={Math.round(value)}
		>
			<div class="fill" style="transform: scaleX({ratio})"></div>
			<div class="segs" aria-hidden="true">
				{#each { length: segments - 1 } as _, i (i)}<span></span>{/each}
			</div>
		</div>
	</div>
</div>

<style>
	.stat {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.icon {
		width: 36px;
		height: 36px;
		border-radius: var(--md-sys-shape-corner-small);
		display: grid;
		place-items: center;
		color: var(--c);
		background: color-mix(in srgb, var(--c) 14%, var(--md-sys-color-surface));
		flex: none;
	}
	.main {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.row {
		display: flex;
		align-items: baseline;
		gap: 8px;
	}
	.spacer {
		flex: 1;
	}
	.delta {
		color: var(--md-sys-color-success);
		animation: rise 1.2s var(--md-sys-motion-easing-emphasized-decelerate);
	}
	@keyframes rise {
		from {
			opacity: 0;
			transform: translateY(8px);
		}
	}
	.bar {
		position: relative;
		height: 10px;
		border-radius: 3px;
		background: color-mix(in srgb, var(--c) 14%, var(--md-sys-color-surface-container-highest));
		overflow: hidden;
	}
	.fill {
		position: absolute;
		inset: 0;
		background: var(--c);
		transform-origin: left;
	}
	.segs {
		position: absolute;
		inset: 0;
		display: flex;
		justify-content: space-evenly;
	}
	.segs span {
		width: 2px;
		background: var(--md-sys-color-surface);
		opacity: 0.7;
	}
</style>
