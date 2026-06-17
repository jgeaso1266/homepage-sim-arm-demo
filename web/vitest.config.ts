import { sveltekit } from '@sveltejs/kit/vite'
import { defineConfig } from 'vitest/config'

// Vitest config kept separate from vite.config.ts: vitest bundles its own (older)
// vite, so mixing its `test` types into the project's vite config trips
// svelte-check. We still need the sveltekit plugin here so vitest resolves
// @viamrobotics/motion-tools' export conditions (its "./lib" subpath only exposes
// a `svelte` condition).
export default defineConfig({
	// The sveltekit plugin is typed against the project's newer vite; vitest's
	// bundled vite types differ. The mismatch is types-only — it works at runtime.
	// @ts-expect-error -- cross-vite-version plugin type incompatibility
	plugins: [sveltekit()],
	test: {
		environment: 'node',
		include: ['src/**/*.spec.ts'],
		setupFiles: ['./vitest-setup.ts'],
	},
})
