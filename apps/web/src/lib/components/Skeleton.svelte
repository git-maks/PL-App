<script lang="ts">
	let { lines = 3, height = 16 }: { lines?: number; height?: number } = $props();
</script>

<div class="sk" aria-busy="true" aria-label="Ładowanie">
	{#each { length: lines } as _, i (i)}
		<span style="height: {height}px; width: {i === lines - 1 ? 60 : 100}%"></span>
	{/each}
</div>

<style>
	.sk {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	span {
		display: block;
		border-radius: 6px;
		background: linear-gradient(90deg, var(--md-sys-color-surface-container-high) 0%, var(--md-sys-color-surface-container-highest) 50%, var(--md-sys-color-surface-container-high) 100%);
		background-size: 200% 100%;
		animation: shimmer 1.4s linear infinite;
	}
	@keyframes shimmer {
		to {
			background-position: -200% 0;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		span {
			animation: none;
		}
	}
</style>
