import vercel from '@sveltejs/adapter-vercel'
import staticAdapter from '@sveltejs/adapter-static'
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

// On Vercel (VERCEL=1) use adapter-vercel so the build output matches what
// Vercel's SvelteKit preset serves. Everywhere else use adapter-static so
// `pnpm build` + `pnpm preview` (and the Playwright e2e) keep working. The app
// is fully prerendered (ssr=false, prerender=true), so adapter-vercel emits pure
// static output too — no serverless function.
const adapter = process.env.VERCEL ? vercel() : staticAdapter()

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter,
		paths: {
			base: process.env.BASE_PATH ?? '',
		},
	},
}

export default config
