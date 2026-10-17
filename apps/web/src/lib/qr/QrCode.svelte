<script lang="ts">
	import { qrcodegen } from './vendor/qrcodegen.js';

	let {
		text,
		size = 240,
		label,
		ecc = 'M'
	}: { text: string; size?: number; label: string; ecc?: 'L' | 'M' | 'Q' | 'H' } = $props();

	const ECC = { L: qrcodegen.QrCode.Ecc.LOW, M: qrcodegen.QrCode.Ecc.MEDIUM, Q: qrcodegen.QrCode.Ecc.QUARTILE, H: qrcodegen.QrCode.Ecc.HIGH };
	const BORDER = 3;

	const qr = $derived(qrcodegen.QrCode.encodeText(text, ECC[ecc]));
	const path = $derived.by(() => {
		let d = '';
		for (let y = 0; y < qr.size; y++) {
			for (let x = 0; x < qr.size; x++) {
				if (qr.getModule(x, y)) d += `M${x + BORDER},${y + BORDER}h1v1h-1z`;
			}
		}
		return d;
	});
	const dim = $derived(qr.size + BORDER * 2);
</script>

<svg
	class="qr"
	width={size}
	height={size}
	viewBox="0 0 {dim} {dim}"
	shape-rendering="crispEdges"
	role="img"
	aria-label={label}
>
	<rect width={dim} height={dim} fill="#fff" />
	<path d={path} fill="#000" />
</svg>

<style>
	.qr {
		border-radius: var(--md-sys-shape-corner-medium);
		max-width: 100%;
		height: auto;
	}
</style>
