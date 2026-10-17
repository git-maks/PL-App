import { defineConfig, devices } from '@playwright/test';

// E2E runs against the production build in demo mode (in-browser backend), served by `vite preview`.
// The "camera" project feeds Chromium a generated video with a station QR code (e2e/global-setup.ts).
const executablePath = process.env.PW_CHROMIUM_PATH || undefined;

export default defineConfig({
	testDir: 'e2e',
	timeout: 60_000,
	expect: { timeout: 15_000 },
	fullyParallel: true,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
	globalSetup: './e2e/global-setup.ts',
	use: {
		baseURL: 'http://localhost:4173',
		locale: 'pl-PL',
		// Also exercises the reduced-motion path (infinite decorative animations are disabled there).
		reducedMotion: 'reduce',
		trace: 'retain-on-failure',
		launchOptions: { executablePath }
	},
	webServer: {
		command: 'npm run preview -- --port 4173 --strictPort',
		port: 4173,
		reuseExistingServer: !process.env.CI
	},
	projects: [
		{ name: 'mobile', use: { ...devices['Pixel 7'] }, testIgnore: /camera\.spec/ },
		{
			name: 'camera',
			testMatch: /camera\.spec/,
			use: {
				...devices['Pixel 7'],
				permissions: ['camera'],
				launchOptions: {
					executablePath,
					args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream', '--use-file-for-fake-video-capture=e2e/.generated/station.y4m']
				}
			}
		}
	]
});
