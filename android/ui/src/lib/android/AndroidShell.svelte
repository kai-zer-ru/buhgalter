<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import OfflineSyncBanner from '$lib/components/OfflineSyncBanner.svelte';
	import ConnectionStatusBar from '$lib/android/ConnectionStatusBar.svelte';
	import { initAndroidBackHandler } from '$lib/android/back-handler';
	import { shellHeader } from '$lib/android/shell-header';
	import { user } from '$lib/stores/auth';
	import { featureFlags, featureRequiredForPath, isFeatureEnabled } from '$lib/features';
	import {
		androidHomeNavItem,
		androidMainNavItemsAfterHome,
		isAndroidAdminGroupActive,
		isAndroidSettingsGroupActive
	} from '$lib/android/nav-items';
	import {
		countInterceptDrafts,
		getCurrentInterceptSettings,
		interceptDraftsTick
	} from '$lib/android/notification-intercept';
	import AndroidDrawerSync from '$lib/android/AndroidDrawerSync.svelte';
	import AndroidDrawerVersion from '$lib/android/AndroidDrawerVersion.svelte';
	import UpdateAvailableModal from '$lib/components/UpdateAvailableModal.svelte';
	import { APP_VERSION } from '$lib/platform/app-version';
	import { fetchAppVersionInfo, applyVersionBlock, type AppVersionInfo } from '$lib/version-check';
	import './android-shell.css';

	type Props = {
		children: Snippet;
		onlogout: () => void | Promise<void>;
	};

	let { children, onlogout }: Props = $props();

	let drawerOpen = $state(false);
	let appVersionInfo = $state<AppVersionInfo>({
		appVersion: APP_VERSION,
		serverVersion: null,
		releaseUrl: null,
		versionMismatch: false,
		versionBlocked: false
	});
	let showUpdateModal = $state(false);

	const path = $derived($page.url.pathname);
	const chrome = $derived($shellHeader);
	const homeNav = $derived(androidHomeNavItem());
	const mainNav = $derived.by(() => {
		void $featureFlags;
		return androidMainNavItemsAfterHome();
	});
	const settingsHref = resolve('/settings');
	const adminHref = resolve('/admin');
	const draftCount = $derived.by(() => {
		void $interceptDraftsTick;
		void $user?.id;
		if (!getCurrentInterceptSettings().enabled) return 0;
		return countInterceptDrafts($user?.id);
	});

	$effect(() => {
		if (!$user || !$featureFlags) return;
		if (path === '/feature-disabled') return;
		const required = featureRequiredForPath(path);
		if (required && !isFeatureEnabled(required, $featureFlags)) {
			void goto(resolve('/feature-disabled'), { replaceState: true });
		}
	});

	function closeDrawer() {
		drawerOpen = false;
	}

	function toggleDrawer() {
		drawerOpen = !drawerOpen;
	}

	function linkClass(active: boolean) {
		return active ? 'android-drawer-link active' : 'android-drawer-link';
	}

	async function refreshAppVersionInfo() {
		const info = await fetchAppVersionInfo(APP_VERSION);
		appVersionInfo = info;
		applyVersionBlock(info);
	}

	onMount(() => {
		// Version check is throttled to 1×/day inside fetchAppVersionInfo — defer off first paint.
		const versionTimer = setTimeout(() => void refreshAppVersionInfo(), 5_000);
		let cleanup: (() => void) | undefined;
		void initAndroidBackHandler({
			isDrawerOpen: () => drawerOpen,
			closeDrawer
		}).then((fn) => {
			cleanup = fn;
		});
		return () => {
			clearTimeout(versionTimer);
			cleanup?.();
		};
	});
</script>

<div class="android-shell-layout" class:drawer-open={drawerOpen}>
	{#if drawerOpen}
		<button
			type="button"
			class="android-drawer-backdrop"
			aria-label={$_('common.close')}
			onclick={closeDrawer}
		></button>
	{/if}

	<aside class="android-drawer" class:open={drawerOpen} aria-hidden={!drawerOpen}>
		<div class="android-drawer-head">
			<AppIcon size={36} />
			<div class="min-w-0">
				<p class="truncate text-base font-semibold">{$_('app.title')}</p>
			</div>
		</div>
		<nav class="android-drawer-nav" aria-label={$_('nav.menu')}>
			<a
				href={resolve(homeNav.href as '/')}
				class={linkClass(homeNav.isActive(path))}
				aria-current={homeNav.isActive(path) ? 'page' : undefined}
				onclick={closeDrawer}
			>
				{$_(homeNav.labelKey)}
			</a>
			{#each mainNav as item (item.href)}
				<a
					href={resolve(item.href as '/')}
					class={linkClass(item.isActive(path))}
					aria-current={item.isActive(path) ? 'page' : undefined}
					onclick={closeDrawer}
				>
					<span class="flex w-full items-center justify-between gap-2">
						<span>{$_(item.labelKey)}</span>
						{#if item.labelKey === 'nav.bankNotificationDrafts' && draftCount > 0}
							<span
								class="rounded-full px-2 py-0.5 text-xs font-semibold tabular-nums text-white"
								style:background="var(--primary)"
							>
								{draftCount}
							</span>
						{/if}
					</span>
				</a>
			{/each}
			<a
				href={settingsHref}
				class={linkClass(isAndroidSettingsGroupActive(path))}
				aria-current={isAndroidSettingsGroupActive(path) ? 'page' : undefined}
				onclick={closeDrawer}
			>
				{$_('nav.settings')}
			</a>
			{#if $user?.is_admin}
				<a
					href={adminHref}
					class={linkClass(isAndroidAdminGroupActive(path))}
					aria-current={isAndroidAdminGroupActive(path) ? 'page' : undefined}
					onclick={closeDrawer}
				>
					{$_('nav.admin')}
				</a>
			{/if}
		</nav>
		<div class="android-drawer-foot">
			{#if drawerOpen}
				<AndroidDrawerSync />
			{/if}
			<div class="android-drawer-foot-row">
				<button
					type="button"
					class="android-drawer-link min-w-0 flex-1 text-left"
					onclick={() => void onlogout()}
				>
					{$_('nav.logout')}
				</button>
				<AndroidDrawerVersion
					info={appVersionInfo}
					onshowUpdate={() => {
						closeDrawer();
						showUpdateModal = true;
					}}
				/>
			</div>
		</div>
	</aside>

	{#if showUpdateModal}
		<UpdateAvailableModal
			info={appVersionInfo}
			onclose={() => {
				showUpdateModal = false;
			}}
		/>
	{/if}

	<header class="android-shell-header">
		{#if chrome}
			<button
				type="button"
				class="btn-icon btn-ghost btn-nav"
				aria-label={$_('nav.back')}
				onclick={chrome.onBack}
			>
				<svg
					aria-hidden="true"
					class="h-5 w-5"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<path d="m15 6-6 6 6 6" />
				</svg>
			</button>
			<h1 class="min-w-0 flex-1 truncate text-base font-semibold">{chrome.title}</h1>
		{:else}
			<button
				type="button"
				class="btn-icon btn-ghost btn-nav"
				aria-expanded={drawerOpen}
				aria-controls="android-drawer"
				onclick={toggleDrawer}
			>
				<span class="sr-only">{$_('nav.menu')}</span>
				<svg
					aria-hidden="true"
					class="h-5 w-5"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<path d="M4 7h16M4 12h16M4 17h16" />
				</svg>
			</button>
			<h1 class="min-w-0 flex-1 truncate text-base font-semibold">{$_('app.title')}</h1>
		{/if}
	</header>

	<OfflineSyncBanner />

	<main class="android-shell-main" class:android-shell-main-flush={!!chrome}>
		{@render children()}
	</main>

	<ConnectionStatusBar />
</div>
