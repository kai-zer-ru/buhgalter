import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { allowedHostsPlugin } from './vite-allowed-hosts';

export default defineConfig({
	plugins: [
		allowedHostsPlugin(),
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				fallback: 'index.html',
				strict: false
			})
		})
	],
	server: {
		allowedHosts: true,
		proxy: {
			'/api': {
				target: 'http://localhost:8765',
				changeOrigin: true
			},
			'/docs': {
				target: 'http://localhost:8765',
				changeOrigin: true
			}
		}
	}
});
