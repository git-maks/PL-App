<script lang="ts">
	import { iconPaths, type IconName } from '#lib/icons/paths';

	let { name, size = 24, label }: { name: IconName; size?: number; label?: string } = $props();
</script>

<svg
	class="icon"
	width={size}
	height={size}
	viewBox="0 -960 960 960"
	fill="currentColor"
	role={label ? 'img' : undefined}
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
>
	<path d={iconPaths[name]} />
</svg>

<style>
	.icon {
		flex: none;
		display: inline-block;
	}
</style>
