<script lang="ts">
	import { STAT_CODES, type Stats } from '#lib/game/types';
	import { statByCode } from '#lib/content/archetypes';

	let { stats, size = 240, labels = true }: { stats: Stats; size?: number; labels?: boolean } = $props();

	const n = STAT_CODES.length;
	const c = 50;
	const r = 34;
	function pt(i: number, k: number) {
		const a = -Math.PI / 2 + (i * 2 * Math.PI) / n;
		return [c + Math.cos(a) * r * k, c + Math.sin(a) * r * k] as const;
	}
	const max = $derived(Math.max(1, ...STAT_CODES.map((s) => stats[s])));
	const shape = $derived(
		STAT_CODES.map((s, i) => pt(i, Math.max(0.04, stats[s] / max)).join(',')).join(' ')
	);
	const summary = $derived(
		STAT_CODES.map((s) => `${statByCode[s].name} ${Math.round(stats[s])}`).join(', ')
	);
</script>

<svg viewBox="0 0 100 100" width={size} height={size} role="img" aria-label="Wykres statystyk: {summary}">
	{#each [0.25, 0.5, 0.75, 1] as k (k)}
		<polygon class="grid" points={STAT_CODES.map((_, i) => pt(i, k).join(',')).join(' ')} />
	{/each}
	{#each STAT_CODES as s, i (s)}
		{@const [x, y] = pt(i, 1)}
		<line class="axis" x1={c} y1={c} x2={x} y2={y} />
	{/each}
	<polygon class="shape" points={shape} />
	{#if labels}
		{#each STAT_CODES as s, i (s)}
			{@const [x, y] = pt(i, 1.27)}
			<text {x} {y} text-anchor="middle" dominant-baseline="middle" fill={statByCode[s].color}>{statByCode[s].name}</text>
		{/each}
	{/if}
</svg>

<style>
	svg {
		overflow: visible;
		max-width: 100%;
		height: auto;
	}
	.grid {
		fill: none;
		stroke: var(--md-sys-color-outline-variant);
		stroke-width: 0.4;
	}
	.axis {
		stroke: var(--md-sys-color-outline-variant);
		stroke-width: 0.3;
	}
	.shape {
		fill: color-mix(in srgb, var(--md-sys-color-primary) 30%, transparent);
		stroke: var(--md-sys-color-primary);
		stroke-width: 1;
		stroke-linejoin: round;
		transition: all var(--md-sys-motion-duration-long2) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	text {
		font: 600 4.2px var(--md-ref-typeface-plain);
	}
</style>
