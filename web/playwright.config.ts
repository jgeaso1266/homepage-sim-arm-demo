import { defineConfig, devices } from '@playwright/test'

// E2E runs against the production build (vite preview), which — unlike the dev
// server's esbuild prebundler — cleanly bundles motion-tools' shaders/CJS deps.
export default defineConfig({
	testDir: './e2e',
	timeout: 60_000,
	fullyParallel: false,
	use: {
		baseURL: 'http://localhost:4173',
		trace: 'on-first-retry',
	},
	projects: [
		{
			name: 'chromium',
			use: {
				...devices['Desktop Chrome'],
				// Headless Chromium has no GPU; enable software WebGL so Threlte's
				// canvas mounts.
				launchOptions: {
					args: ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader'],
				},
			},
		},
	],
	webServer: {
		command: 'pnpm build && pnpm preview --port 4173',
		port: 4173,
		reuseExistingServer: !process.env.CI,
		timeout: 180_000,
	},
})
