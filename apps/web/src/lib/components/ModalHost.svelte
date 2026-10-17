<script lang="ts">
	import { modal, answer } from '#lib/state/modal.svelte';
	import Dialog from './Dialog.svelte';
	import Button from './Button.svelte';
</script>

{#key modal.current?.id}
	{#if modal.current}
		{@const req = modal.current}
		<Dialog open title={req.title} icon={req.icon} onclose={() => answer(req.id, false)}>
			<p class="body-large">{req.body}</p>
			{#snippet actions()}
				{#if req.cancel}<Button variant="text" onclick={() => answer(req.id, false)}>{req.cancel}</Button>{/if}
				<!-- svelte-ignore a11y_autofocus -->
				<Button autofocus onclick={() => answer(req.id, true)}>{req.confirm}</Button>
			{/snippet}
		</Dialog>
	{/if}
{/key}

<style>
	p {
		margin: 0;
	}
</style>
