import type { Plugin } from 'vite';

const DEV_API = process.env.BUHGALTER_DEV_API ?? 'http://localhost:8765';

export type AllowedHostsConfig = string[] | true;

type EnvHosts = AllowedHostsConfig | 'auto' | null;

function hostsFromEnv(): EnvHosts {
	const raw = process.env.BUHGALTER_VITE_ALLOWED_HOSTS?.trim();
	if (!raw) return null;
	if (raw === 'true' || raw === '*') return true;
	if (raw === 'auto') return 'auto';
	const hosts = raw
		.split(',')
		.map((h) => h.trim())
		.filter(Boolean);
	return hosts.length > 0 ? hosts : null;
}

async function hostFromExternalURL(): Promise<string | null> {
	const res = await fetch(`${DEV_API}/api/v1/setup/status`, {
		signal: AbortSignal.timeout(1500)
	});
	if (!res.ok) return null;

	const data = (await res.json()) as { external_url?: string };
	const raw = data.external_url?.trim();
	if (!raw) return null;

	const host = new URL(raw).hostname;
	return host || null;
}

/** allowedHosts для Vite dev. По умолчанию true — без запросов к API. */
export async function loadAllowedHosts(): Promise<AllowedHostsConfig> {
	const fromEnv = hostsFromEnv();
	if (fromEnv === true) {
		return true;
	}
	if (Array.isArray(fromEnv)) {
		console.log('vite: allowedHosts из BUHGALTER_VITE_ALLOWED_HOSTS');
		return fromEnv;
	}
	if (fromEnv === 'auto') {
		try {
			const host = await hostFromExternalURL();
			if (host) {
				console.log(`vite: allowedHosts из external_url админки: ${host}`);
				return [host];
			}
		} catch {
			// API не запущен — нормально для dev
		}
		console.warn('vite: external_url не получен — allowedHosts: true');
	}

	return true;
}

/** Только при старте `vite dev` (configureServer), не при build/sync/check. */
export function allowedHostsPlugin(): Plugin {
	return {
		name: 'buhgalter-allowed-hosts',
		apply: 'serve',
		async configureServer(server) {
			server.config.server.allowedHosts = await loadAllowedHosts();
		}
	};
}
