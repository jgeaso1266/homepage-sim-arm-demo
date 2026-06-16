import { sveltekit } from '@sveltejs/kit/vite'
import { defineConfig } from 'vite'
import glsl from 'vite-plugin-glsl'

export default defineConfig({
	// motion-tools ships .hdr environment maps in its dist; treat them as assets.
	assetsInclude: ['**/*.hdr'],
	plugins: [glsl(), sveltekit()],
	optimizeDeps: {
		esbuildOptions: { target: 'esnext' },
		// Don't esbuild-prebundle motion-tools: it ships .glsl shaders that only the
		// glsl rollup plugin (not esbuild) can load. Excluding it routes the package
		// through the plugin pipeline in dev, matching the production build.
		exclude: ['@viamrobotics/motion-tools'],
		// ...but its CJS sub-deps still need esbuild's named-export interop, which
		// they lose when their svelte-lib parent is excluded. Pre-bundle them.
		include: ['tweakpane'],
	},
	build: { target: 'esnext' },
	ssr: {
		noExternal: ['camera-controls'],
	},
	test: {
		environment: 'node',
		include: ['src/**/*.spec.ts'],
		setupFiles: ['./vitest-setup.ts'],
	},
})
