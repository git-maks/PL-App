<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from '#lib/icons/paths';
	import Icon from './Icon.svelte';

	let {
		open = $bindable(false),
		title,
		icon,
		sheet = false,
		children,
		actions,
		onclose
	}: {
		open?: boolean;
		title?: string;
		/** MD3 hero icon above a centred title. */
		icon?: IconName;
		/** Render as an MD3 bottom sheet instead of a centred dialog. */
		sheet?: boolean;
		children: Snippet;
		actions?: Snippet;
		onclose?: () => void;
	} = $props();

	let el: HTMLDialogElement | undefined = $state();
	const titleId = 'dlg-' + Math.random().toString(36).slice(2, 8);

	$effect(() => {
		if (!el) return;
		if (open && !el.open) el.showModal();
		if (!open && el.open) el.close();
	});

	function handleClose() {
		open = false;
		onclose?.();
	}
	function backdrop(e: MouseEvent) {
		if (e.target === el) handleClose();
	}
</script>

<dialog bind:this={el} class:sheet onclose={handleClose} onclick={backdrop} aria-labelledby={title ? titleId : undefined}>
	<div class="content">
		{#if sheet}<div class="handle" aria-hidden="true"></div>{/if}
		{#if icon}<span class="hero" aria-hidden="true"><Icon name={icon} size={24} /></span>{/if}
		{#if title}<h2 id={titleId} class="headline-small" class:centered={!!icon}>{title}</h2>{/if}
		<div class="body">{@render children()}</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
</dialog>

<style>
	dialog {
		border: none;
		padding: 0;
		max-width: min(560px, calc(100vw - 48px));
		width: 100%;
		border-radius: var(--md-sys-shape-corner-extra-large);
		background: var(--md-sys-color-surface-container-high);
		color: var(--md-sys-color-on-surface);
	}
	dialog[open] {
		animation: dialog-in var(--md-sys-motion-duration-medium2) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	dialog::backdrop {
		background: rgb(0 0 0 / 0.32);
	}
	.content {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	.hero {
		align-self: center;
		color: var(--md-sys-color-secondary);
		margin-bottom: -4px;
	}
	.centered {
		text-align: center;
	}
	.body {
		color: var(--md-sys-color-on-surface-variant);
		font: var(--md-sys-typescale-body-medium);
	}
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: 8px;
		flex-wrap: wrap;
	}
	.sheet {
		margin: auto 0 0;
		max-width: 100%;
		width: 100%;
		max-height: 90dvh;
		border-radius: var(--md-sys-shape-corner-extra-large) var(--md-sys-shape-corner-extra-large) 0 0;
		background: var(--md-sys-color-surface-container-low);
	}
	.sheet .content {
		max-width: var(--uq-max-width);
		margin: 0 auto;
		padding: 8px 24px calc(24px + var(--uq-safe-bottom));
	}
	.sheet[open] {
		animation: sheet-in var(--md-sys-motion-duration-medium4) var(--md-sys-motion-easing-emphasized-decelerate);
	}
	.handle {
		width: 32px;
		height: 4px;
		border-radius: 2px;
		background: var(--md-sys-color-on-surface-variant);
		opacity: 0.4;
		margin: 8px auto 8px;
	}
	@keyframes dialog-in {
		from {
			opacity: 0;
			transform: scale(0.92);
		}
	}
	@keyframes sheet-in {
		from {
			transform: translateY(100%);
		}
	}
</style>
