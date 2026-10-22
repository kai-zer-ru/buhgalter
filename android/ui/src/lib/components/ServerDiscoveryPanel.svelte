<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		discoverLanServers,
		mergeDiscoveredServers,
		type DiscoveredServer
	} from '$lib/platform/lan-discovery';
	import { discoverMdnsServers } from '$lib/platform/mdns-discovery';
	import { getWifiIpv4Subnet } from '$lib/platform/wifi-subnet';

	type Props = {
		onselect: (origin: string) => void;
	};

	let { onselect }: Props = $props();

	let scanning = $state(false);
	let wifiOnly = $state(false);
	let noWifi = $state(false);
	let servers = $state<DiscoveredServer[]>([]);
	let scanError = $state<string | null>(null);

	async function scan() {
		if (scanning) return;
		scanning = true;
		scanError = null;
		servers = [];
		noWifi = false;
		wifiOnly = false;

		try {
			const { Network } = await import('@capacitor/network');
			const status = await Network.getStatus();
			if (!status.connected || status.connectionType !== 'wifi') {
				noWifi = true;
				return;
			}
			wifiOnly = true;

			const subnet = await getWifiIpv4Subnet();
			if (!subnet) {
				scanError = $_('serverDiscovery.noSubnet');
				return;
			}

			const [mdnsServers, subnetServers] = await Promise.all([
				discoverMdnsServers({ timeoutMs: 4000 }),
				discoverLanServers({
					subnetBase: subnet.base,
					prefix: subnet.prefix,
					excludeIp: subnet.ip
				})
			]);
			servers = mergeDiscoveredServers(mdnsServers, subnetServers);
		} catch {
			scanError = $_('serverDiscovery.scanFailed');
		} finally {
			scanning = false;
		}
	}

	onMount(() => {
		void scan();
	});

	function formatRow(server: DiscoveredServer): string {
		try {
			const { hostname, port } = new URL(server.origin);
			return `${hostname}:${port}`;
		} catch {
			return server.origin;
		}
	}

	function formatExternalLabel(url: string): string {
		try {
			return new URL(url).host;
		} catch {
			return url;
		}
	}
</script>

<section class="space-y-3" aria-labelledby="lan-discovery-heading">
	<div class="flex items-center justify-between gap-2">
		<h2 id="lan-discovery-heading" class="text-sm font-semibold">
			{$_('serverDiscovery.title')}
		</h2>
		<button type="button" class="btn-ghost text-sm" disabled={scanning} onclick={() => void scan()}>
			{scanning ? $_('serverDiscovery.scanning') : $_('serverDiscovery.rescan')}
		</button>
	</div>

	{#if noWifi}
		<p class="text-sm" style:color="var(--text-muted)">{$_('serverDiscovery.wifiRequired')}</p>
	{:else if scanning}
		<div class="flex items-center gap-2 text-sm" style:color="var(--text-muted)">
			<span
				class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-t-transparent"
				style:border-color="var(--primary)"
			></span>
			{$_('serverDiscovery.scanning')}
		</div>
	{:else if scanError}
		<p class="text-sm" style:color="var(--danger)" role="alert">{scanError}</p>
	{:else if wifiOnly && servers.length === 0}
		<p class="text-sm" style:color="var(--text-muted)">{$_('serverDiscovery.empty')}</p>
	{:else if servers.length > 0}
		<ul class="space-y-2" role="list">
			{#each servers as server (server.origin)}
				<li>
					<button
						type="button"
						class="w-full rounded-lg border px-3 py-2.5 text-left transition hover:opacity-90"
						style:border-color="var(--border)"
						onclick={() => onselect(server.origin)}
					>
						<p class="font-medium">
							{$_('serverDiscovery.rowTitle', { values: { address: formatRow(server) } })}
						</p>
						{#if server.externalUrl}
							<p class="text-xs" style:color="var(--text-muted)">
								{$_('serverDiscovery.rowExternal', {
									values: { domain: formatExternalLabel(server.externalUrl) }
								})}
							</p>
						{/if}
						<p class="text-xs" style:color="var(--text-muted)">
							{$_('serverDiscovery.rowMeta', {
								values: { version: server.version, db: server.dbStatus }
							})}
						</p>
					</button>
				</li>
			{/each}
		</ul>
	{/if}

	<p class="text-xs" style:color="var(--text-muted)">{$_('serverDiscovery.manualHint')}</p>
</section>
