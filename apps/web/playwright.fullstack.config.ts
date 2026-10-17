import { defineConfig, devices } from '@playwright/test';

// Full-stack E2E: the browser talks to the real Go backend + PostgreSQL (the binary serves the web
// app built with PUBLIC_API_MODE=http). Start it with `make e2e-fullstack` or in CI (see .github).
export default defineConfig({
	testDir: 'e2e-fullstack',
	timeout: 60_000,
	expect: { timeout: 15_000 },
	retries: process.env.CI ? 1 : 0,
	reporter: 'list',
	use: {
		...devices['Pixel 7'],
		baseURL: process.env.UQ_E2E_URL ?? 'http://localhost:8080',
		locale: 'pl-PL',
		reducedMotion: 'reduce',
		trace: 'retain-on-failure',
		launchOptions: { executablePath: process.env.PW_CHROMIUM_PATH || undefined }
	}
});
