<script lang="ts">
	let {
		value = $bindable(3),
		labels,
		emoji,
		label
	}: { value?: number; labels: string[]; emoji: string[]; label: string } = $props();

	const max = $derived(labels.length);
	const ratio = $derived((value - 1) / (max - 1));
</script>

<div class="rating">
	<div class="face" aria-hidden="true">
		{#key value}<span class="emoji">{emoji[value - 1]}</span>{/key}
	</div>
	<p class="current title-large" aria-live="polite">{labels[value - 1]}</p>
	<div class="track-wrap" style="--ratio: {ratio}">
		<input
			type="range"
			min="1"
			{max}
			step="1"
			bind:value
			aria-label={label}
			aria-valuetext={labels[value - 1]}
		/>
		<div class="ticks" aria-hidden="true">
			{#each labels as _, i (i)}<span class:on={i < value}></span>{/each}
		</div>
	</div>
	<div class="ends label-medium muted" aria-hidden="true">
		<span>{labels[0]}</span><span>{labels[max - 1]}</span>
	</div>
</div>

<style>
	.rating {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
	}
	.face {
		height: 72px;
		display: grid;
		place-items: center;
	}
	.emoji {
		font-size: 56px;
		line-height: 1;
		animation: pop var(--md-sys-motion-duration-medium2) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	@keyframes pop {
		from {
			transform: scale(0.6);
			opacity: 0.4;
		}
	}
	.current {
		min-height: 1.75rem;
	}
	.track-wrap {
		position: relative;
		width: 100%;
		height: 48px;
	}
	input {
		-webkit-appearance: none;
		appearance: none;
		width: 100%;
		height: 48px;
		margin: 0;
		background: transparent;
		position: relative;
		z-index: 1;
		touch-action: pan-y;
	}
	input::-webkit-slider-runnable-track {
		height: 16px;
		border-radius: 8px;
		background: linear-gradient(
			to right,
			var(--md-sys-color-primary) calc(var(--ratio) * 100%),
			var(--md-sys-color-secondary-container) calc(var(--ratio) * 100%)
		);
	}
	input::-moz-range-track {
		height: 16px;
		border-radius: 8px;
		background: var(--md-sys-color-secondary-container);
	}
	input::-moz-range-progress {
		height: 16px;
		border-radius: 8px;
		background: var(--md-sys-color-primary);
	}
	input::-webkit-slider-thumb {
		-webkit-appearance: none;
		width: 8px;
		height: 44px;
		margin-top: -14px;
		border-radius: 4px;
		background: var(--md-sys-color-primary);
		box-shadow: 0 0 0 4px var(--md-sys-color-surface);
	}
	input::-moz-range-thumb {
		width: 8px;
		height: 44px;
		border: none;
		border-radius: 4px;
		background: var(--md-sys-color-primary);
	}
	input:focus-visible {
		outline: none;
	}
	input:focus-visible::-webkit-slider-thumb {
		box-shadow: 0 0 0 4px var(--md-sys-color-surface), 0 0 0 7px var(--md-sys-color-primary);
	}
	.ticks {
		position: absolute;
		inset: 0 6px;
		display: flex;
		justify-content: space-between;
		align-items: center;
		pointer-events: none;
		z-index: 2;
	}
	.ticks span {
		width: 4px;
		height: 4px;
		border-radius: 50%;
		background: var(--md-sys-color-primary);
	}
	.ticks span.on {
		background: var(--md-sys-color-on-primary);
	}
	.ends {
		width: 100%;
		display: flex;
		justify-content: space-between;
	}
</style>
