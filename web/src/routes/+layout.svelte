<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { ApiError, getSetupStatus } from '$lib/api/client';
	import { loadUser, logout, user, hasRecentSession } from '$lib/stores/auth';
	import { registerServiceWorker } from '$lib/pwa';
	import { initTheme, syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';
	import './layout.css';

	let { children } = $props();
	let ready = $state(false);
	let bootError = $state<string | null>(null);
	let navOpen = $state(false);

	const path = $derived($page.url.pathname);
	const isSetup = $derived(path === '/setup');

	async function goReady(route: '/' | '/login' | '/setup') {
		ready = true;
		await goto(resolve(route));
	}

	async function bootstrap() {
		bootError = null;
		const softBoot = get(user) !== null && hasRecentSession();
		if (!softBoot) {
			ready = false;
		}

		initTheme();

		const currentPath = $page.url.pathname;
		try {
			const status = await getSetupStatus();
			if (!status.configured && currentPath !== '/setup') {
				await goReady('/setup');
				return;
			}
			if (status.configured && currentPath === '/setup') {
				await goReady('/login');
				return;
			}

			if (status.configured && !['/setup', '/login', '/register'].includes(currentPath)) {
				const auth = await loadUser();
				if (auth === 'unauthorized') {
					await goReady('/login');
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
					if (currentPath === '/login' || currentPath === '/register') {
						await goReady('/');
						return;
					}
				}
			}
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				await goReady('/login');
				return;
			}
			if (currentPath !== '/setup' && currentPath !== '/login' && currentPath !== '/register') {
				ready = true;
				bootError = 'server_unavailable';
				return;
			}
		}
		ready = true;
	}

	onMount(() => {
		registerServiceWorker();
		if (get(user) !== null && hasRecentSession()) {
			ready = true;
		}
		void bootstrap();
	});

	async function handleLogout() {
		await logout();
		await goto(resolve('/login'));
	}
</script>

{#if bootError}
	<div class="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
		<p style:color="var(--text-muted)">{$_('common.server_unavailable')}</p>
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
{:else if isSetup || path === '/login' || path === '/register'}
	{@render children()}
{:else}
	<div class="min-h-screen">
		<header
			class="border-b px-6 py-4 backdrop-blur-sm"
			style:border-color="var(--border)"
			style:background-color="color-mix(in srgb, var(--bg-elevated) 85%, transparent)"
		>
			<div class="mx-auto flex max-w-5xl items-center justify-between gap-4">
				<a href={resolve('/')} class="flex items-center gap-2 text-lg font-semibold tracking-tight">
					<AppIcon size={32} />
					{$_('app.title')}
				</a>
				{#if $user}
					<details
						class="relative sm:hidden"
						open={navOpen}
						ontoggle={(e) => (navOpen = (e.currentTarget as HTMLDetailsElement).open)}
					>
						<summary
							class="btn-ghost cursor-pointer list-none select-none [&::-webkit-details-marker]:hidden"
						>
							{$_('nav.menu')}
						</summary>
						<div
							class="popover-panel nav-mobile-panel absolute right-0 z-50 mt-2 min-w-[12rem] p-2"
						>
							<a href={resolve('/')} class="nav-mobile-link" onclick={() => (navOpen = false)}
								>{$_('nav.home')}</a
							>
							<a href={resolve('/debts')} class="nav-mobile-link" onclick={() => (navOpen = false)}
								>{$_('nav.debts')}</a
							>
							<a
								href={resolve('/credits')}
								class="nav-mobile-link"
								onclick={() => (navOpen = false)}>{$_('nav.credits')}</a
							>
							<a href={resolve('/stats')} class="nav-mobile-link" onclick={() => (navOpen = false)}
								>{$_('nav.stats')}</a
							>
							<a
								href={resolve('/settings')}
								class="nav-mobile-link"
								onclick={() => (navOpen = false)}>{$_('nav.settings')}</a
							>
							{#if $user.is_admin}
								<a
									href={resolve('/admin')}
									class="nav-mobile-link"
									onclick={() => (navOpen = false)}>{$_('nav.admin')}</a
								>
							{/if}
						</div>
					</details>
				{/if}
				<nav class="flex items-center gap-2">
					{#if $user}
						<div class="hidden items-center gap-2 sm:flex">
							<a href={resolve('/')} class="btn-ghost">{$_('nav.home')}</a>
							<a href={resolve('/debts')} class="btn-ghost">{$_('nav.debts')}</a>
							<a href={resolve('/credits')} class="btn-ghost">{$_('nav.credits')}</a>
							<a href={resolve('/stats')} class="btn-ghost">{$_('nav.stats')}</a>
							<a href={resolve('/settings')} class="btn-ghost">{$_('nav.settings')}</a>
							{#if $user.is_admin}
								<a href={resolve('/admin')} class="btn-ghost">{$_('nav.admin')}</a>
							{/if}
						</div>
						<span class="hidden text-sm sm:inline" style:color="var(--text-muted)">
							{#if $user.display_name && $user.display_name !== $user.login}
								{$user.display_name}
								<span class="opacity-70">(@{$user.login})</span>
							{:else}
								@{$user.login}
							{/if}
						</span>
						<button type="button" class="btn-ghost" onclick={handleLogout}>
							{$_('nav.logout')}
						</button>
					{/if}
				</nav>
			</div>
		</header>
		<main class="mx-auto max-w-5xl px-6 py-8">
			{@render children()}
		</main>
	</div>
{/if}

<ConfirmDialog />
<ToastContainer />
