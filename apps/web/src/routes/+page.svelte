<script lang="ts">
	import { goto } from '$app/navigation';
	import { app } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { isDemo } from '#lib/api';
	import Button from '#lib/components/Button.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';

	$effect(() => {
		if (app.ready && app.me) void goto(href('/hero'), { replaceState: true });
	});
</script>

<svelte:head><title>UniQuest – Politechnika Lubelska</title></svelte:head>

<div class="landing">
	<div class="hero">
		<img class="logo" src={href('/icons/icon.svg')} alt="" width="96" height="96" />
		<p class="label-large kicker">Politechnika Lubelska</p>
		<h1 class="display-small pixel">UniQuest</h1>
		<p class="body-large lead">Odwiedzaj wydziały i laboratoria, skanuj kody na koniec zajęć i zbuduj swoją kartę postaci. Na końcu podpowiemy, który kierunek pasuje do Ciebie.</p>
	</div>

	<ul class="features">
		<li><span class="ic"><Icon name="qr_code_scanner" /></span><span><strong>Skanuj</strong> kod QR na koniec warsztatu lub pokazu</span></li>
		<li><span class="ic"><StatIcon code="STR" /></span><span><strong>Oceniaj</strong>, jak bardzo Ci się podobało – tak rosną Twoje statystyki</span></li>
		<li><span class="ic"><Icon name="workspace_premium" /></span><span><strong>Odkryj</strong> swoją ścieżkę: kierunki i koła naukowe dla Ciebie</span></li>
	</ul>

	<div class="cta">
		<Button full icon="play_arrow" href={href('/onboarding')}>Zaczynamy</Button>
		<Button full variant="text" href={href('/explore')}>Tylko się rozglądam – poznaj uczelnię</Button>
	</div>

	{#if isDemo}
		<p class="demo body-small muted">Wersja demonstracyjna: dane zapisują się tylko na tym urządzeniu. Kody QR do testów znajdziesz w <a href={href('/demo/stations')}>arkuszu stacji testowych</a>.</p>
	{/if}
	<p class="foot body-small muted">Politechnika Lubelska · ul. Nadbystrzycka 38D, 20-618 Lublin</p>
</div>

<style>
	.landing {
		min-height: 100dvh;
		max-width: 520px;
		margin: 0 auto;
		padding: calc(var(--uq-safe-top) + 32px) 24px calc(var(--uq-safe-bottom) + 24px);
		display: flex;
		flex-direction: column;
		gap: 28px;
	}
	.hero {
		display: flex;
		flex-direction: column;
		align-items: center;
		text-align: center;
		gap: 8px;
	}
	.logo {
		image-rendering: pixelated;
		border-radius: 22px;
		box-shadow: var(--md-sys-elevation-2);
		margin-bottom: 8px;
		animation: float 4s ease-in-out infinite;
	}
	@keyframes float {
		50% {
			transform: translateY(-6px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.logo {
			animation: none;
		}
	}
	.kicker {
		color: var(--md-sys-color-primary);
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}
	h1 {
		font-size: 2.75rem;
		line-height: 1.1;
	}
	.lead {
		color: var(--md-sys-color-on-surface-variant);
	}
	.features {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}
	.features li {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 12px 16px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
		font: var(--md-sys-typescale-body-medium);
	}
	.ic {
		width: 40px;
		height: 40px;
		border-radius: 12px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
		flex: none;
	}
	.cta {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-top: auto;
	}
	.demo,
	.foot {
		text-align: center;
	}
</style>
