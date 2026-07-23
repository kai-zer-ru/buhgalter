<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import ServerDiscoveryPanel from '$lib/components/ServerDiscoveryPanel.svelte';
	import ServerProfileFields from '$lib/components/ServerProfileFields.svelte';
	import ServerTrustDialog from '$lib/components/ServerTrustDialog.svelte';
	import {
		getServerProfile,
		normalizeProfile,
		setServerProfile,
		type ServerProfile
	} from '$lib/platform/server-profile';
	import { refreshActiveServerUrl } from '$lib/platform/server-url';
	import { serverConnectErrorKey } from '$lib/platform/server-connect';
	import {
		addTrustedOrigin,
		SslTrustRequiredError,
		verifyServerProfile
	} from '$lib/platform/server-verify';

	const initial = getServerProfile();
	let lanUrl = $state(initial.lanUrl);
	let trustedOrigins = $state([...initial.trustedOrigins]);
	let loading = $state(false);
	let formError = $state<string | null>(null);
	let trustDialogOpen = $state(false);
	let trustOrigin = $state('');
	let trustEnabled = $state(false);
	let pendingProfile = $state<ServerProfile | null>(null);

	function currentProfile(): ServerProfile {
		return normalizeProfile({
			lanUrl,
			remoteUrl: '',
			homeSsids: [],
			lanFallbackRemote: false,
			trustedOrigins
		});
	}

	async function saveProfile(profile: ServerProfile) {
		const verified = await verifyServerProfile(profile);
		const existing = getServerProfile();
		setServerProfile({
			...verified,
			remoteUrl: existing.remoteUrl,
			homeSsids: existing.homeSsids,
			lanFallbackRemote: existing.lanFallbackRemote,
			trustedOrigins: verified.trustedOrigins
		});
		await refreshActiveServerUrl();
		await goto(resolve('/login'));
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
</script>

<svelte:head>
	<title>{$_('serverSetup.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-2 text-2xl font-semibold">{$_('serverSetup.title')}</h1>
		<p class="mb-6 text-sm" style:color="var(--text-muted)">{$_('serverSetup.subtitle')}</p>
		<ServerDiscoveryPanel
			onselect={(origin) => {
				lanUrl = origin;
				formError = null;
			}}
		/>
		<div class="my-6 border-t" style:border-color="var(--border)"></div>
		<form class="space-y-4" onsubmit={submit}>
			<ServerProfileFields variant="setup" bind:lanUrl />
			{#if formError}
				<p class="text-sm" style:color="var(--danger)" role="alert">{formError}</p>
			{/if}
			<button type="submit" class="btn-primary w-full" disabled={loading}>
				{loading ? $_('serverSetup.checking') : $_('serverSetup.continue')}
			</button>
		</form>
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
