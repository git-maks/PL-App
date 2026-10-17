<script lang="ts">
	import { identicon } from '#lib/avatar/pixelate';

	let {
		src,
		seed,
		size = 96,
		frame,
		label = 'Awatar'
	}: { src?: string | null; seed: string; size?: number; frame?: string | null; label?: string } = $props();

	const ident = $derived(identicon(seed));
</script>

<div class="avatar" style="--s: {size}px; --frame: {frame ?? 'var(--md-sys-color-primary)'}" role="img" aria-label={label}>
	{#if src}
		<img {src} alt="" width={size} height={size} />
	{:else}
		<svg viewBox="-1 -1 10 10" width={size} height={size} shape-rendering="crispEdges" aria-hidden="true">
			<rect x="-1" y="-1" width="10" height="10" fill="var(--md-sys-color-primary-container)" />
			{#each ident.cells as on, i (i)}
				{#if on}<rect x={i % 8} y={Math.floor(i / 8)} width="1" height="1" fill={ident.color} />{/if}
			{/each}
		</svg>
	{/if}
</div>

<style>
	.avatar {
		width: var(--s);
		height: var(--s);
		border-radius: calc(var(--s) * 0.18);
		overflow: hidden;
		flex: none;
		box-shadow:
			0 0 0 3px var(--md-sys-color-surface),
			0 0 0 6px var(--frame);
		background: var(--md-sys-color-primary-container);
	}
	img,
	svg {
		width: 100%;
		height: 100%;
		image-rendering: pixelated;
		display: block;
	}
</style>
