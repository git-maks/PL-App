<script lang="ts">
	let {
		value = $bindable(''),
		label,
		type = 'text',
		supporting,
		error,
		autocomplete,
		inputmode,
		maxlength,
		required = false,
		autofocus = false,
		id = 'tf-' + Math.random().toString(36).slice(2, 8),
		oninput
	}: {
		value?: string;
		label: string;
		type?: 'text' | 'email' | 'search';
		supporting?: string;
		error?: string | null;
		autocomplete?: HTMLInputElement['autocomplete'];
		inputmode?: 'text' | 'email' | 'numeric' | 'search';
		maxlength?: number;
		required?: boolean;
		autofocus?: boolean;
		id?: string;
		oninput?: (e: Event) => void;
	} = $props();

	function focusOnMount(node: HTMLInputElement) {
		if (autofocus) node.focus();
	}
</script>

<div class="field" class:invalid={!!error}>
	<input
		{id}
		{type}
		bind:value
		placeholder=" "
		{autocomplete}
		{inputmode}
		{maxlength}
		{required}
		aria-invalid={!!error}
		aria-describedby={supporting || error ? id + '-sup' : undefined}
		{oninput}
		use:focusOnMount
	/>
	<label for={id}>{label}</label>
	{#if error || supporting}
		<p class="sup body-small" id="{id}-sup">{error ?? supporting}</p>
	{/if}
</div>

<style>
	.field {
		position: relative;
		width: 100%;
	}
	input {
		width: 100%;
		height: 56px;
		padding: 16px;
		border: 1px solid var(--md-sys-color-outline);
		border-radius: var(--md-sys-shape-corner-extra-small);
		background: transparent;
		font: var(--md-sys-typescale-body-large);
		transition: border-color var(--md-sys-motion-duration-short2) linear;
	}
	input:focus {
		outline: none;
		border: 2px solid var(--md-sys-color-primary);
		padding: 15px;
	}
	label {
		position: absolute;
		left: 12px;
		top: 16px;
		padding: 0 4px;
		color: var(--md-sys-color-on-surface-variant);
		background: var(--uq-container, var(--md-sys-color-surface));
		pointer-events: none;
		transform-origin: left top;
		transition:
			transform var(--md-sys-motion-duration-short4) var(--md-sys-motion-easing-standard),
			color var(--md-sys-motion-duration-short2) linear;
	}
	input:focus + label,
	input:not(:placeholder-shown) + label {
		transform: translateY(-24px) scale(0.75);
	}
	input:focus + label {
		color: var(--md-sys-color-primary);
	}
	.sup {
		margin: 4px 16px 0;
		color: var(--md-sys-color-on-surface-variant);
	}
	.invalid input {
		border-color: var(--md-sys-color-error);
	}
	.invalid label,
	.invalid .sup {
		color: var(--md-sys-color-error);
	}
</style>
