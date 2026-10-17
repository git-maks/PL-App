<script lang="ts" module>
	import type { StatCode } from '#lib/game/types';

	// 8×8 pixel-art icons drawn for UniQuest ('#' = filled pixel).
	const SPRITES: Record<StatCode, string[]> = {
		STR: ['..#..#..', '.######.', '##.##.##', '.##..##.', '.##..##.', '##.##.##', '.######.', '..#..#..'], // gear
		INT: ['########', '##......', '.##.....', '..##....', '..##....', '.##.....', '##......', '########'], // sigma
		MAN: ['....##..', '...##...', '..##....', '.######.', '....##..', '...##...', '..##....', '.##.....'], // bolt
		KRE: ['...##...', '...##...', '..#..#..', '..#..#..', '.#....#.', '.#....#.', '#......#', '#......#'], // compass
		ENE: ['......##', '....####', '..######', '.######.', '.#####..', '.####...', '.#......', '#.......'], // leaf
		CHA: ['#..##..#', '##.##.##', '########', '########', '.######.', '.######.', '........', '.######.'], // crown
		WIS: ['........', '.##..##.', '#..##..#', '#..##..#', '#..##..#', '#..##..#', '.##..##.', '...##...'], // book
		VIT: ['........', '.##..##.', '########', '########', '########', '.######.', '..####..', '...##...'] // heart
	};

	function toPath(rows: string[]): string {
		let d = '';
		rows.forEach((row, y) => {
			[...row].forEach((ch, x) => {
				if (ch === '#') d += `M${x} ${y}h1v1h-1z`;
			});
		});
		return d;
	}
	const PATHS = Object.fromEntries(Object.entries(SPRITES).map(([k, v]) => [k, toPath(v)])) as Record<StatCode, string>;
</script>

<script lang="ts">
	let { code, size = 24, color = 'currentColor' }: { code: StatCode; size?: number; color?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 8 8" shape-rendering="crispEdges" aria-hidden="true" style="color: {color}">
	<path d={PATHS[code]} fill="currentColor" />
</svg>

<style>
	svg {
		flex: none;
	}
</style>
