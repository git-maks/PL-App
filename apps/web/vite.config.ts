import { sveltekit } from '@sveltejs/kit/vite';
import adapter from '@sveltejs/adapter-static';
import { defineConfig } from 'vitest/config';

// BASE_PATH: '' for the Go server / local, '/PL-App' for GitHub Pages.
const base = (process.env.BASE_PATH ?? '') as '' | `/${string}`;
// Pages needs 404.html as the SPA fallback; the Go server serves 200.html.
const fallback = process.env.SPA_FALLBACK ?? '200.html';

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter({ fallback, strict: false }),
			paths: { base, relative: false },
			prerender: { handleHttpError: 'warn', handleMissingId: 'warn' },
			version: { name: process.env.APP_VERSION ?? String(Date.now()) },
			serviceWorker: { register: false },
			// Hash-based CSP for SvelteKit's own inline bootstrap script (the Go server adds the other directives).
			csp: { mode: 'hash', directives: { 'script-src': ['self', 'wasm-unsafe-eval'] } }
		})
	],
	server: {
		fs: { allow: ['../../content'] },
		proxy: { '/api': { target: process.env.API_PROXY ?? 'http://localhost:8080', changeOrigin: false } }
	},
	define: {
		__API_MODE__: JSON.stringify(process.env.PUBLIC_API_MODE ?? 'http')
	},
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node'
	}
});
