<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import ServerDiscoveryPanel from '$lib/components/ServerDiscoveryPanel.svelte';
	import ServerProfileFields from '$lib/components/ServerProfileFields.svelte';
	import ServerTrustDialog from '$lib/components/ServerTrustDialog.svelte';
	import { copyOutboxExportToClipboard } from '$lib/offline/export';
	import { clearRefCache } from '$lib/offline/ref-cache';
	import { clearAuthToken } from '$lib/platform/auth-token';
	import { clearAppLock } from '$lib/platform/app-lock';
	import {
		activeServerMode,
		getServerProfile,
		normalizeProfile,
		setServerProfile,
		type ServerProfile
	} from '$lib/platform/server-profile';
	import { clearServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
	import { getCurrentWifiSsid } from '$lib/platform/wifi-subnet';
	import { serverConnectErrorKey } from '$lib/platform/server-connect';
	import {
		addTrustedOrigin,
		SslTrustRequiredError,
		verifyServerProfile
	} from '$lib/platform/server-verify';
	import { user, clearLastUser } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import { confirm } from '$lib/confirm';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import {
		exportDebugLogToDownloads,
		isDebugLogEnabled,
		setDebugLogEnabled
	} from '$lib/platform/debug-log';

	const initial = getServerProfile();
	let lanUrl = $state(initial.lanUrl);
	let remoteUrl = $state(initial.remoteUrl);
	let homeSsids = $state([...initial.homeSsids]);
	let lanFallbackRemote = $state(initial.lanFallbackRemote);
	let trustedOrigins = $state([...initial.trustedOrigins]);
	let currentSsid = $state<string | null>(null);
	let loading = $state(false);
	let formError = $state<string | null>(null);
	let trustDialogOpen = $state(false);
	let trustOrigin = $state('');
	let trustEnabled = $state(false);
	let pendingProfile = $state<ServerProfile | null>(null);
	let exportingOutbox = $state(false);
	let loggingEnabled = $state(isDebugLogEnabled());
	let savingLog = $state(false);

	onMount(() => {
		void refreshActiveServerUrl().then(() => void refreshSsid());
	});

	async function refreshSsid() {
		const result = await getCurrentWifiSsid();
		currentSsid = result.ssid;
	}

	function currentProfile(): ServerProfile {
		return normalizeProfile({ lanUrl, remoteUrl, homeSsids, lanFallbackRemote, trustedOrigins });
	}

	async function saveProfile(profile: ServerProfile) {
		const verified = await verifyServerProfile(profile);
		setServerProfile(verified);
		await refreshActiveServerUrl();
		await refreshSsid();
		toast($_('common.saved'));
	}

	async function submit(e: Event) {
		e.preventDefault();
		if (loading) return;

		const profile = currentProfile();
		if (!profile.lanUrl) {
			formError = $_('serverSetup.urlRequired');
			return;
		}

		loading = true;
		formError = null;
		pendingProfile = profile;
		try {
			await saveProfile(profile);
		} catch (err) {
			if (SslTrustRequiredError.is(err)) {
				trustOrigin = err.origin;
				trustEnabled = profile.trustedOrigins.includes(err.origin);
				trustDialogOpen = true;
				return;
			}
			formError = $_(serverConnectErrorKey(err));
		} finally {
			loading = false;
		}
	}

	async function confirmTrust() {
		if (!pendingProfile || !trustOrigin) return;
		if (!trustEnabled) {
			formError = $_('serverSetup.ssl.trustRequired');
			trustDialogOpen = false;
			return;
		}
		const profile = addTrustedOrigin(pendingProfile, trustOrigin, true);
		lanUrl = profile.lanUrl;
		remoteUrl = profile.remoteUrl;
		homeSsids = [...profile.homeSsids];
		lanFallbackRemote = profile.lanFallbackRemote;
		trustedOrigins = [...profile.trustedOrigins];
		trustDialogOpen = false;
		loading = true;
		formError = null;
		try {
			await saveProfile(profile);
		} catch (err) {
			if (SslTrustRequiredError.is(err)) {
				trustOrigin = err.origin;
				trustEnabled = profile.trustedOrigins.includes(err.origin);
				trustDialogOpen = true;
				return;
			}
			formError = $_(serverConnectErrorKey(err));
		} finally {
			loading = false;
		}
	}

	async function exportOutbox() {
		if (exportingOutbox) return;
		exportingOutbox = true;
		try {
			await copyOutboxExportToClipboard();
			toast($_('offline.export.copied'), 'success');
		} catch {
			toast($_('offline.export.failed'), 'error');
		} finally {
			exportingOutbox = false;
		}
	}

	async function toggleLogging() {
		if (loggingEnabled) {
			const download = await confirm({
				title: $_('serverSettings.logging.downloadTitle'),
				message: $_('serverSettings.logging.downloadMessage'),
				confirmLabel: $_('serverSettings.logging.download'),
				cancelLabel: $_('serverSettings.logging.skip')
			});
			setDebugLogEnabled(false);
			loggingEnabled = false;
			if (download) {
				savingLog = true;
				try {
					const path = await exportDebugLogToDownloads();
					toast($_('serverSettings.logging.saved', { values: { path } }), 'success');
				} catch {
					toast($_('serverSettings.logging.saveFailed'), 'error');
				} finally {
					savingLog = false;
				}
			}
			return;
		}
		setDebugLogEnabled(true);
		loggingEnabled = true;
		toast($_('serverSettings.logging.enabled'), 'success');
	}

	async function disconnect() {
		clearServerUrl();
		const { clearWidgetsOnLogout } = await import('$lib/widgets/publish');
		await clearWidgetsOnLogout();
		await clearAuthToken();
		await clearAppLock();
		clearRefCache();
		clearLastUser();
		user.set(null);
		await goto(resolve('/server-setup'));
	}
</script>

<svelte:head>
	<title>{$_('serverSettings.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="card mx-auto max-w-lg space-y-6">
	<div>
		<h1 class="text-2xl font-semibold">{$_('serverSettings.title')}</h1>
		<p class="mt-1 text-sm" style:color="var(--text-muted)">{$_('serverSettings.subtitle')}</p>
	</div>
	<ServerDiscoveryPanel
		onselect={(origin) => {
			lanUrl = origin;
			formError = null;
		}}
	/>
	<div class="border-t" style:border-color="var(--border)"></div>
	<form class="space-y-4" onsubmit={submit}>
		<ServerProfileFields
			bind:lanUrl
			bind:remoteUrl
			bind:homeSsids
			bind:lanFallbackRemote
			showActiveStatus
			activeMode={$activeServerMode}
			{currentSsid}
		/>
		{#if formError}
			<p class="text-sm" style:color="var(--danger)" role="alert">{formError}</p>
		{/if}
		<button type="submit" class="btn-primary" disabled={loading}>
			{loading ? $_('serverSetup.checking') : $_('common.save')}
		</button>
	</form>
	<div class="space-y-2 border-t pt-4" style:border-color="var(--border)">
		<div class="flex items-start justify-between gap-3">
			<div class="min-w-0">
				<h2 class="text-sm font-medium">{$_('serverSettings.logging.title')}</h2>
				<p class="mt-1 text-xs leading-snug" style:color="var(--text-muted)">
					{$_('serverSettings.logging.hint')}
				</p>
			</div>
			<ToggleSwitch
				checked={loggingEnabled}
				disabled={savingLog}
				label={$_('serverSettings.logging.title')}
				onchange={() => void toggleLogging()}
			/>
		</div>
	</div>
	<div class="space-y-2 border-t pt-4" style:border-color="var(--border)">
		<div>
			<h2 class="text-sm font-medium">{$_('serverSettings.exportOutbox')}</h2>
			<p class="mt-1 text-xs leading-snug" style:color="var(--text-muted)">
				{$_('serverSettings.exportOutboxHint')}
			</p>
		</div>
		<button
			type="button"
			class="btn-ghost text-sm"
			disabled={exportingOutbox}
			onclick={() => void exportOutbox()}
		>
			{exportingOutbox ? $_('common.loading') : $_('offline.export.button')}
		</button>
	</div>
	<div class="border-t pt-4" style:border-color="var(--border)">
		<button type="button" class="btn-ghost text-sm" onclick={() => void disconnect()}>
			{$_('serverSettings.disconnect')}
		</button>
	</div>
</div>

<ServerTrustDialog
	bind:open={trustDialogOpen}
	origin={trustOrigin}
	bind:trusted={trustEnabled}
	onconfirm={() => void confirmTrust()}
	oncancel={() => {
		trustDialogOpen = false;
	}}
/>
