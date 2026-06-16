import tailwindcss from '@tailwindcss/vite'
import { sveltekit } from '@sveltejs/kit/vite'
import { defineConfig } from 'vite'
import glsl from 'vite-plugin-glsl'

export default defineConfig({
	// motion-tools ships .hdr environment maps in its dist; treat them as assets.
	assetsInclude: ['**/*.hdr'],
	plugins: [glsl(), tailwindcss(), sveltekit()],
	optimizeDeps: {
		esbuildOptions: { target: 'esnext' },
	},
	build: { target: 'esnext' },
	ssr: {
		noExternal: ['camera-controls'],
	},
})
