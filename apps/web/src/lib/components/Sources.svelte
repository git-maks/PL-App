<script lang="ts">
	import type { Source } from '#lib/content/types';
	import { date } from '#lib/time';

	let { sources, verifiedAt }: { sources: Source[]; verifiedAt: string } = $props();
</script>

<footer class="src body-small muted">
	<p>Źródła (zweryfikowano {date(verifiedAt)}):</p>
	<ul>
		{#each sources as s (s.url)}
			<li><a href={s.url} target="_blank" rel="noopener">{s.title}</a></li>
		{/each}
	</ul>
</footer>

<style>
	.src {
		padding: 12px 4px 0;
		border-top: 1px solid var(--md-sys-color-outline-variant);
	}
	ul {
		margin: 4px 0 0;
		padding-left: 18px;
	}
	a {
		color: inherit;
	}
</style>
