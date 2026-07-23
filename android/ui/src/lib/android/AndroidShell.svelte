<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import OfflineSyncBanner from '$lib/components/OfflineSyncBanner.svelte';
	import ConnectionStatusBar from '$lib/android/ConnectionStatusBar.svelte';
	import { initAndroidBackHandler } from '$lib/android/back-handler';
	import { shellHeader } from '$lib/android/shell-header';
	import { user } from '$lib/stores/auth';
	import {
		androidHomeNavItem,
		androidMainNavItemsAfterHome,
		isAndroidAdminGroupActive,
		isAndroidSettingsGroupActive
	} from '$lib/android/nav-items';
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
	const mainNav = $derived(androidMainNavItemsAfterHome());
	const settingsHref = resolve('/settings');
	const adminHref = resolve('/admin');

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
		void refreshAppVersionInfo();
		let cleanup: (() => void) | undefined;
		void initAndroidBackHandler({
			isDrawerOpen: () => drawerOpen,
			closeDrawer
		}).then((fn) => {
			cleanup = fn;
		});
		return () => cleanup?.();
	});

	$effect(() => {
		if (!drawerOpen) return;
		void refreshAppVersionInfo();
	});
</script>

<div class="android-shell-layout">
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
					{$_(item.labelKey)}
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
			<AndroidDrawerSync />
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
