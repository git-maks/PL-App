<script lang="ts">
	import { install, promptInstall } from '#lib/pwa/install.svelte';
	import Card from './Card.svelte';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';
</script>

{#if !install.standalone && (install.deferred || install.ios)}
	<Card variant="outlined">
		<div class="stack">
			<div class="row">
				<Icon name="add_box" />
				<h3 class="title-medium">Dodaj UniQuest do ekranu głównego</h3>
			</div>
			{#if install.deferred}
				<p class="body-medium muted">Aplikacja uruchomi się jak natywna, bez paska przeglądarki, i zadziała offline.</p>
				<Button variant="tonal" icon="download" onclick={() => promptInstall()}>Zainstaluj</Button>
			{:else}
				<ol class="body-medium muted steps">
					<li>Stuknij <Icon name="ios_share" size={18} label="Udostępnij" /> w pasku Safari.</li>
					<li>Wybierz „Do ekranu początkowego”.</li>
					<li>Na iPhonie powiadomienia działają tylko w zainstalowanej aplikacji.</li>
				</ol>
			{/if}
		</div>
	</Card>
{/if}

<style>
	.steps {
		margin: 0;
		padding-left: 20px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.steps :global(svg) {
		display: inline;
		vertical-align: -4px;
	}
</style>
