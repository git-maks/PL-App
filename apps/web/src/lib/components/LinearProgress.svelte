<script lang="ts">
	let { value, label, indeterminate = false }: { value?: number; label: string; indeterminate?: boolean } = $props();
</script>

<div
	class="lp"
	class:indeterminate
	role="progressbar"
	aria-label={label}
	aria-valuemin={indeterminate ? undefined : 0}
	aria-valuemax={indeterminate ? undefined : 100}
	aria-valuenow={indeterminate ? undefined : Math.round((value ?? 0) * 100)}
>
	<div class="fill" style:transform={indeterminate ? undefined : `scaleX(${Math.max(0, Math.min(1, value ?? 0))})`}></div>
</div>

<style>
	.lp {
		height: 6px;
		border-radius: 3px;
		background: var(--md-sys-color-secondary-container);
		overflow: hidden;
	}
	.fill {
		height: 100%;
		background: var(--md-sys-color-primary);
		border-radius: 3px;
		transform-origin: left;
		transition: transform var(--md-sys-motion-duration-long2) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	.indeterminate .fill {
		width: 40%;
		animation: slide 1.4s var(--md-sys-motion-easing-standard) infinite;
	}
	@keyframes slide {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(250%);
		}
	}
</style>
