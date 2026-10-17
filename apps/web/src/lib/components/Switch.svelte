<script lang="ts">
	let {
		checked = $bindable(false),
		label,
		supporting,
		onchange
	}: { checked?: boolean; label: string; supporting?: string; onchange?: (v: boolean) => void } = $props();
	const id = 'sw-' + Math.random().toString(36).slice(2, 8);
</script>

<label class="row-sw" for={id}>
	<span class="text">
		<span class="body-large">{label}</span>
		{#if supporting}<span class="body-medium muted">{supporting}</span>{/if}
	</span>
	<input {id} type="checkbox" role="switch" bind:checked onchange={() => onchange?.(checked)} />
</label>

<style>
	.row-sw {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 8px 16px;
		min-height: 56px;
		cursor: pointer;
	}
	.text {
		flex: 1;
		display: flex;
		flex-direction: column;
	}
	input {
		appearance: none;
		-webkit-appearance: none;
		flex: none;
		width: 52px;
		height: 32px;
		margin: 0;
		border-radius: 16px;
		border: 2px solid var(--md-sys-color-outline);
		background: var(--md-sys-color-surface-container-highest);
		position: relative;
		cursor: pointer;
		transition: background var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard);
	}
	input::before {
		content: '';
		position: absolute;
		top: 6px;
		left: 6px;
		width: 16px;
		height: 16px;
		border-radius: 50%;
		background: var(--md-sys-color-outline);
		transition: all var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard);
	}
	input:checked {
		background: var(--md-sys-color-primary);
		border-color: var(--md-sys-color-primary);
	}
	input:checked::before {
		left: 22px;
		top: 2px;
		width: 24px;
		height: 24px;
		background: var(--md-sys-color-on-primary);
	}
</style>
