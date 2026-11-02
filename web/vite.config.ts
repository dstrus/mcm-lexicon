import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { copy } from './copy.ts';
import { thirdParty } from './third-party.ts';

// During development, `make dev` runs MCM Lexicon and this dev server in front
// of it. API requests are passed through to MCM Lexicon's port with the token
// both were started with, since the page Vite serves can't carry it.
const devToken = process.env.LEXICON_DEV_TOKEN ?? '';
const devPort = process.env.LEXICON_DEV_PORT ?? '8776';

export default defineConfig({
	plugins: [
		copy(),
		sveltekit({
			// A single-page app: MCM Lexicon serves index.html for every view's
			// address, and the page routes itself.
			adapter: adapter({ pages: 'build', fallback: 'index.html' })
		}),
		thirdParty()
	],
	server: {
		port: 5174, // Hermaeus's dev server uses 5173
		proxy: {
			'/api': {
				target: `http://127.0.0.1:${devPort}`,
				changeOrigin: true,
				headers: { 'X-Lexicon-Token': devToken }
			}
		}
	}
});
