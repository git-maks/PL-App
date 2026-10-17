<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from './Icon.svelte';
	import IconButton from './IconButton.svelte';

	let { facts, interval = 7000 }: { facts: { text: string; source?: string }[]; interval?: number } = $props();

	let track: HTMLDivElement | undefined = $state();
	let index = $state(0);
	let pausedUntil = 0;
	let hovering = false;

	function host(url: string) {
		try {
			return new URL(url).hostname.replace(/^www\./, '');
		} catch {
			return url;
		}
	}

	function slideWidth() {
		const first = track?.firstElementChild as HTMLElement | null;
		if (!track || !first) return 1;
		return first.offsetWidth + parseFloat(getComputedStyle(track).columnGap || '0');
	}

	function show(i: number, smooth = true) {
		if (!track) return;
		index = (i + facts.length) % facts.length;
		track.scrollTo({ left: index * slideWidth(), behavior: smooth ? 'smooth' : 'instant' });
	}

	/** Manual navigation pauses the automatic rotation for a while. */
	function pause(ms = 12_000) {
		pausedUntil = Date.now() + ms;
	}

	function onscroll() {
		if (track) index = Math.min(facts.length - 1, Math.round(track.scrollLeft / slideWidth()));
	}

	onMount(() => {
		show(Math.floor(Math.random() * facts.length), false);
		const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
		if (reduced) return;
		const timer = setInterval(() => {
			if (hovering || document.hidden || Date.now() < pausedUntil) return;
			if (track?.contains(document.activeElement)) return;
			show(index + 1);
		}, interval);
		return () => clearInterval(timer);
	});
</script>

<section class="carousel" aria-roledescription="karuzela" aria-label="Ciekawostki o Politechnice">
	<div class="head">
		<Icon name="lightbulb" />
		<h2 class="title-small">Czy wiesz, że…</h2>
		<span class="spacer"></span>
		<span class="label-medium num" aria-hidden="true">{index + 1}/{facts.length}</span>
		<IconButton icon="arrow_back" label="Poprzednia ciekawostka" onclick={() => (pause(), show(index - 1))} />
		<IconButton icon="arrow_forward" label="Następna ciekawostka" onclick={() => (pause(), show(index + 1))} />
	</div>
	<div
		class="track"
		role="presentation"
		bind:this={track}
		{onscroll}
		onpointerdown={() => pause()}
		onpointerenter={() => (hovering = true)}
		onpointerleave={() => (hovering = false)}
		aria-live="off"
	>
		{#each facts as f, i (i)}
			<article class="slide" role="group" aria-roledescription="slajd" aria-label="{i + 1} z {facts.length}">
				<p class="body-large">{f.text}</p>
				{#if f.source}
					<a class="label-medium src" href={f.source} target="_blank" rel="noopener" tabindex={i === index ? 0 : -1}>Źródło: {host(f.source)}</a>
				{/if}
			</article>
		{/each}
	</div>
	<div class="progress" aria-hidden="true"><span style="width: {((index + 1) / facts.length) * 100}%"></span></div>
</section>

<style>
	.carousel {
		border-radius: var(--md-sys-shape-corner-extra-large, 28px);
		background: var(--md-sys-color-tertiary-container);
		color: var(--md-sys-color-on-tertiary-container);
		padding: 8px 0 16px;
		overflow: hidden;
	}
	.head {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 0 8px 0 20px;
	}
	.head h2 {
		margin: 0;
	}
	.track {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: 100%;
		column-gap: 0;
		overflow-x: auto;
		scroll-snap-type: x mandatory;
		overscroll-behavior-x: contain;
		scrollbar-width: none;
	}
	.track::-webkit-scrollbar {
		display: none;
	}
	.slide {
		scroll-snap-align: start;
		padding: 4px 20px 12px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		min-width: 0;
	}
	.slide p {
		margin: 0;
	}
	.src {
		color: inherit;
		align-self: flex-start;
	}
	.progress {
		height: 3px;
		margin: 0 20px;
		border-radius: 2px;
		background: color-mix(in srgb, currentColor 15%, transparent);
	}
	.progress span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: currentColor;
		transition: width var(--md-sys-motion-duration-medium2, 300ms) var(--md-sys-motion-easing-standard);
	}
</style>
