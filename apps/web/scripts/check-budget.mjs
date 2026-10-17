// Fails when the JavaScript needed for a first visit exceeds the budget (spec §5.6):
// app shell (entry/start + entry/app + root layout) plus the heaviest single route, each with
// its static imports, measured gzip-compressed from the Vite manifest.
import { readFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { join } from 'node:path';

const SHELL_BUDGET = 80 * 1024;
const ROUTE_BUDGET = 50 * 1024;
const root = new URL('../.svelte-kit/output/client/', import.meta.url).pathname;
const manifest = JSON.parse(readFileSync(join(root, '.vite/manifest.json'), 'utf8'));

const size = new Map();
const gz = (file) => {
	if (!size.has(file)) size.set(file, gzipSync(readFileSync(join(root, file))).length);
	return size.get(file);
};
function closure(keys) {
	const seen = new Set();
	const visit = (k) => {
		const e = manifest[k];
		if (!e || seen.has(e.file)) return;
		seen.add(e.file);
		for (const i of e.imports ?? []) visit(i);
	};
	keys.forEach(visit);
	return seen;
}
const keys = Object.keys(manifest);
const pick = (re) => keys.filter((k) => re.test(k));
const shell = closure([...pick(/entry\/start|client-optimized\/app\.js$/), ...pick(/nodes\/0\.js$/)]);
const shellSize = [...shell].reduce((a, f) => a + gz(f), 0);

let worst = { key: '', size: 0 };
for (const k of pick(/nodes\/\d+\.js$/)) {
	const extra = [...closure([k])].filter((f) => !shell.has(f)).reduce((a, f) => a + gz(f), 0);
	if (extra > worst.size) worst = { key: k, size: extra };
}
const kb = (n) => (n / 1024).toFixed(1) + ' kB';
console.log(`app shell:      ${kb(shellSize)} (budget ${kb(SHELL_BUDGET)})`);
console.log(`heaviest route: ${kb(worst.size)} ${worst.key.split('/').pop()} (budget ${kb(ROUTE_BUDGET)})`);
if (shellSize > SHELL_BUDGET || worst.size > ROUTE_BUDGET) {
	console.error('JavaScript budget exceeded');
	process.exit(1);
}
