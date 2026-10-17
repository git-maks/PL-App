<script lang="ts">
	import { fly } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { toasts, dismissToast } from '#lib/state/toast.svelte';
	import { prefersReducedMotion } from 'svelte/motion';
</script>

<div class="host" aria-live="polite" aria-atomic="true">
	{#each toasts.items as t (t.id)}
		<div class="snack" transition:fly={{ y: prefersReducedMotion.current ? 0 : 24, duration: 250, easing: cubicOut }}>
			<span class="text">{t.message}</span>
			{#if t.action}
				<button
					class="action interactive"
					type="button"
					onclick={() => {
						t.action?.run();
						dismissToast(t.id);
					}}>{t.action.label}</button
				>
			{/if}
		</div>
	{/each}
</div>

<style>
	.host {
		position: fixed;
		left: 0;
		right: 0;
		bottom: calc(var(--uq-nav-height) + var(--uq-safe-bottom) + 12px);
		z-index: 50;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		pointer-events: none;
		padding: 0 var(--uq-gutter);
	}
	.snack {
		pointer-events: auto;
		width: 100%;
		max-width: 560px;
		min-height: 48px;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 4px 8px 4px 16px;
		border-radius: var(--md-sys-shape-corner-extra-small);
		background: var(--md-sys-color-inverse-surface);
		color: var(--md-sys-color-inverse-on-surface);
		box-shadow: var(--md-sys-elevation-3);
		font: var(--md-sys-typescale-body-medium);
	}
	.text {
		flex: 1;
		padding: 10px 0;
	}
	.action {
		border: none;
		background: transparent;
		color: var(--md-sys-color-inverse-primary);
		font: var(--md-sys-typescale-label-large);
		padding: 10px 12px;
		border-radius: 20px;
	}
</style>
