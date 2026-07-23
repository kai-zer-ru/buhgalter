<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { resolveAppPath } from '$lib/android/form-nav';
	import { _ } from 'svelte-i18n';
	import {
		loadUser,
		logout,
		user,
		clearSessionHint,
		restoreCachedUser,
		markSessionHint,
		fallbackSessionUser,
		persistLastUser
	} from '$lib/stores/auth';
	import { isPublicAppRoute, sessionExpiredTick } from '$lib/auth/session-expired';
	import { invalidateApiCache } from '$lib/api/cache';
	import { initTheme, syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import { syncRemoteI18nOnMismatch } from '$lib/i18n/remote-sync';
	import { hasServerUrl, refreshActiveServerUrl } from '$lib/platform/server-url';
	import { initAuthToken, clearAuthToken, getAuthToken } from '$lib/platform/auth-token';
	import { initNativeOfflineSync } from '$lib/offline/init';
	import { prepareBootstrapConnectivity } from '$lib/offline/bootstrap-connectivity';
	import { markServerOffline } from '$lib/offline/server-connectivity';
	import {
		appLockVisible,
		clearAppLock,
		initAppLockListener,
		lockSession,
		refreshAppLockConfig
	} from '$lib/platform/app-lock';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import AccountTransferConfirmDialog from '$lib/components/AccountTransferConfirmDialog.svelte';
	import AndroidShell from '$lib/android/AndroidShell.svelte';
	import AppLockScreen from '$lib/android/AppLockScreen.svelte';
	import VersionBlockScreen from '$lib/android/VersionBlockScreen.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';
	import { APP_VERSION } from '$lib/platform/app-version';
	import {
		fetchAppVersionInfo,
		applyVersionBlock,
		clearVersionBlock,
		versionBlockInfo
	} from '$lib/version-check';
	import { initDeepLinkListener } from '$lib/android/deep-link';
	import { initShareTargetListener } from '$lib/android/share-target';
	import { initDebugLogListeners, debugLogInfo } from '$lib/platform/debug-log';
	import type { User } from '$lib/api/client';
	import './layout.css';

	let { children } = $props();
	let ready = $state(false);
	let bootError = $state<string | null>(null);
	let pendingDeepLink = $state<string | null>(null);

	const path = $derived($page.url.pathname);
	let lastLoggedPath = $state('');

	function canNavigateDeepLink(): boolean {
		return (
			ready &&
			!bootError &&
			hasServerUrl() &&
			$user !== null &&
			!$versionBlockInfo &&
			!$appLockVisible
		);
	}

	async function consumePendingDeepLink() {
		if (!pendingDeepLink || !canNavigateDeepLink()) return;
		const route = pendingDeepLink;
		pendingDeepLink = null;
		await goto(resolveAppPath(route), { replaceState: true });
	}

	$effect(() => {
		if (!canNavigateDeepLink() || !pendingDeepLink) return;
		void consumePendingDeepLink();
	});

	$effect(() => {
		const p = path;
		if (!p || p === lastLoggedPath) return;
		lastLoggedPath = p;
		debugLogInfo('nav', `Route ${p}`);
	});

	$effect(() => {
		if ($sessionExpiredTick === 0) return;
		clearSessionHint();
		void clearAuthToken();
		void clearAppLock();
		user.set(null);
		invalidateApiCache();
		if (ready && !bootError && !isPublicAppRoute(path)) {
			void goto(resolve('/login'), { replaceState: true });
		}
	});

	$effect(() => {
		if (!ready || bootError) return;
		if (!hasServerUrl()) {
			if (path !== '/server-setup') {
				void goto(resolve('/server-setup'), { replaceState: true });
			}
			return;
		}
		if (isPublicAppRoute(path)) return;
		if ($user === null) {
			void goto(resolve('/login'), { replaceState: true });
		}
	});

	/** Version / remote i18n after lock UI is already shown (must not block PIN). */
	async function refreshRemoteSessionMeta(currentUser: User): Promise<void> {
		const versionInfo = await fetchAppVersionInfo(APP_VERSION);
		applyVersionBlock(versionInfo);
		await syncRemoteI18nOnMismatch(APP_VERSION, versionInfo.serverVersion, currentUser.language);
	}

	function scheduleBackgroundSessionRefresh(seed: User): void {
		void (async () => {
			try {
				await prepareBootstrapConnectivity();
				const auth = await loadUser();
				const current = get(user) ?? seed;
				if (auth === 'ok' || current.id !== 'local-session') {
					await refreshRemoteSessionMeta(current);
				}
			} catch {
				// background only — unlock UI already shown
			}
		})();
	}

	/**
	 * Token present → always show PIN/biometrics. Never block on /health or missing ref-cache.
	 */
	async function unlockWithExistingSession(): Promise<boolean> {
		if (!getAuthToken()) return false;

		const cached = restoreCachedUser();
		const profile = cached ?? fallbackSessionUser();
		if (cached) persistLastUser(cached);

		// Optimistic offline until probe succeeds — unlock must not wait on LAN/remote health.
		markServerOffline();
		user.set(profile);
		markSessionHint();
		setLocale(profile.language);
		syncThemeFromUser(profile.theme);
		await refreshAppLockConfig();
		lockSession();
		scheduleBackgroundSessionRefresh(profile);
		return true;
	}

	async function bootstrap() {
		bootError = null;
		ready = false;
		debugLogInfo('bootstrap', 'App bootstrap started');
		initTheme();
		await initAuthToken();

		const currentPath = $page.url.pathname;

		if (!hasServerUrl()) {
			await clearAuthToken();
			user.set(null);
			clearVersionBlock();
			if (currentPath !== '/server-setup') {
				await goto(resolve('/server-setup'), { replaceState: true });
			}
			ready = true;
			return;
		}

		const onLoginPath = currentPath === '/login' || currentPath.startsWith('/login/');

		if (isPublicAppRoute(currentPath)) {
			if (onLoginPath && getAuthToken()) {
				await refreshActiveServerUrl();
				if (await unlockWithExistingSession()) {
					await goto(resolve('/'), { replaceState: true });
					ready = true;
					return;
				}
			}
			ready = true;
			return;
		}

		await refreshActiveServerUrl();
		if (await unlockWithExistingSession()) {
			ready = true;
			return;
		}

		// No token — only then may we wait for connectivity / show unavailable.
		await prepareBootstrapConnectivity();
		const auth = await loadUser();
		if (auth === 'unauthorized') {
			await goto(resolve('/login'), { replaceState: true });
			ready = true;
			return;
		}
		if (auth === 'network') {
			ready = true;
			bootError = 'server_unavailable';
			return;
		}

		const currentUser = get(user);
		if (currentUser) {
			setLocale(currentUser.language);
			syncThemeFromUser(currentUser.theme);
			await refreshAppLockConfig();
			const versionInfo = await fetchAppVersionInfo(APP_VERSION);
			applyVersionBlock(versionInfo);
			await syncRemoteI18nOnMismatch(APP_VERSION, versionInfo.serverVersion, currentUser.language);
			if (versionInfo.versionBlocked) {
				ready = true;
				return;
			}
			lockSession();
		}

		ready = true;
	}

	onMount(() => {
		initDebugLogListeners();
		initNativeOfflineSync();
		const cleanupAppLock = initAppLockListener();
		let cleanupDeepLink: (() => void) | undefined;
		let cleanupShare: (() => void) | undefined;
		void initDeepLinkListener((route) => {
			pendingDeepLink = route;
		}).then((cleanup) => {
			cleanupDeepLink = cleanup;
		});
		void initShareTargetListener(
			(route) => {
				pendingDeepLink = route;
			},
			() => get(_)('share.imageFallback') as string
		).then((cleanup) => {
			cleanupShare = cleanup;
		});
		void bootstrap();
		return () => {
			cleanupAppLock();
			cleanupDeepLink?.();
			cleanupShare?.();
		};
	});

	async function handleLogout() {
		await logout();
		await goto(resolve('/login'));
	}
</script>

{#if bootError}
	<div class="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
		<p style:color="var(--text-muted)">
			{bootError === 'server_not_configured'
				? $_('common.server_not_configured')
				: $_('common.server_unavailable')}
		</p>
		<button type="button" class="btn-primary" onclick={() => void bootstrap()}>
			{$_('common.retry')}
		</button>
	</div>
{:else if !ready}
	<div class="flex min-h-screen items-center justify-center">
		<div class="flex items-center gap-3" style:color="var(--text-muted)">
			<span
				class="inline-block h-5 w-5 animate-spin rounded-full border-2 border-t-transparent"
				style:border-color="var(--primary)"
			></span>
			{$_('common.loading')}
		</div>
	</div>
{:else if path === '/server-setup' || path === '/login' || path.startsWith('/login/')}
	{@render children()}
{:else if $user && $versionBlockInfo}
	<VersionBlockScreen info={$versionBlockInfo} />
{:else if $user && $appLockVisible}
	<AppLockScreen onunlocked={() => appLockVisible.set(false)} />
{:else if $user}
	<AndroidShell onlogout={handleLogout}>
		{@render children()}
	</AndroidShell>
{:else}
	<div class="flex min-h-screen items-center justify-center">
		<div class="flex items-center gap-3" style:color="var(--text-muted)">
			<span
				class="inline-block h-5 w-5 animate-spin rounded-full border-2 border-t-transparent"
				style:border-color="var(--primary)"
			></span>
			{$_('common.loading')}
		</div>
	</div>
{/if}

<ConfirmDialog />
<AccountTransferConfirmDialog />
<ToastContainer />
