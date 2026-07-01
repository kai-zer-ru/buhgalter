<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { afterNavigate, goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { ApiError, getSetupStatus } from '$lib/api/client';
	import { loadUser, logout, user, hasRecentSession } from '$lib/stores/auth';
	import { registerServiceWorker } from '$lib/pwa';
	import { initTheme, syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import UpdateAvailableModal from '$lib/components/UpdateAvailableModal.svelte';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import IconButton from '$lib/components/IconButton.svelte';
	import NavDropdown, { type NavDropdownItem } from '$lib/components/NavDropdown.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';
	import AdminPasswordResetBanner from '$lib/components/AdminPasswordResetBanner.svelte';
	import { checkForVersionUpdate, type PendingVersionUpdate } from '$lib/version-check';
	import './layout.css';

	let { children } = $props();
	let ready = $state(false);
	let bootError = $state<string | null>(null);
	let navOpen = $state(false);
	let pendingUpdate = $state<PendingVersionUpdate | null>(null);

	afterNavigate(() => {
		navOpen = false;
	});

	function closeNav() {
		navOpen = false;
	}

	const path = $derived($page.url.pathname);
	const isSetup = $derived(path === '/setup');

	type NavItem = {
		href: string;
		labelKey: string;
		mobileOnly?: boolean;
		isActive: (pathname: string, search: URLSearchParams) => boolean;
	};

	const flatNavItems: NavItem[] = [
		{
			href: resolve('/accounts'),
			labelKey: 'nav.accounts',
			isActive: (p) => p.startsWith('/accounts')
		},
		{
			href: resolve('/debts'),
			labelKey: 'nav.debts',
			isActive: (p) => p.startsWith('/debts') || p.startsWith('/debtors')
		},
		{
			href: resolve('/credits'),
			labelKey: 'nav.credits',
			isActive: (p) => p.startsWith('/credits')
		},
		{
			href: resolve('/stats'),
			labelKey: 'nav.stats',
			isActive: (p) => p.startsWith('/stats')
		}
	];

	const settingsNavItems: NavDropdownItem[] = [
		{
			path: '/settings',
			labelKey: 'settings.tab.profile',
			isActive: (p) => p === '/settings'
		},
		{
			path: '/settings/password',
			labelKey: 'settings.tab.password',
			isActive: (p) => p === '/settings/password'
		},
		{
			path: '/settings/tokens',
			labelKey: 'settings.tab.tokens',
			isActive: (p) => p === '/settings/tokens'
		},
		{
			path: '/settings/notifications',
			labelKey: 'settings.tab.notifications',
			isActive: (p) => p === '/settings/notifications'
		},
		{
			path: '/settings/categories',
			labelKey: 'settings.tab.categories',
			isActive: (p) => p === '/settings/categories'
		},
		{
			path: '/settings/import',
			labelKey: 'settings.tab.import',
			isActive: (p) => p === '/settings/import'
		},
		{
			path: '/settings/recurring-operations',
			labelKey: 'nav.recurring',
			isActive: (p) => p.startsWith('/settings/recurring-operations')
		}
	];

	const adminNavItems: NavDropdownItem[] = [
		{
			path: '/admin',
			labelKey: 'admin.tab.system',
			isActive: (p) => p === '/admin'
		},
		{
			path: '/admin/users',
			labelKey: 'admin.tab.users',
			isActive: (p) => p.startsWith('/admin/users')
		},
		{
			path: '/admin/backups',
			labelKey: 'admin.tab.backups',
			isActive: (p) => p.startsWith('/admin/backups')
		},
		{
			path: '/admin/diagnostics',
			labelKey: 'admin.tab.diagnostics',
			isActive: (p) => p.startsWith('/admin/diagnostics')
		}
	];

	function isSettingsGroupActive(p: string) {
		return p === '/settings' || p.startsWith('/settings/');
	}

	function isAdminGroupActive(p: string) {
		return p === '/admin' || p.startsWith('/admin/');
	}

	function navLinkClass(active: boolean, base: string) {
		return active ? `${base} nav-link-active` : base;
	}

	function isNavItemActive(item: NavItem) {
		return item.isActive(path, $page.url.searchParams);
	}

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
					if (currentUser.is_admin) {
						void checkForVersionUpdate().then((update) => {
							if (update) pendingUpdate = update;
						});
					}
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
		{#if navOpen}
			<button
				type="button"
				class="nav-mobile-backdrop fixed inset-0 z-40 sm:hidden"
				aria-label={$_('common.close')}
				onclick={closeNav}
			></button>
		{/if}
		<header
			class="sticky top-0 z-50 border-b px-4 py-3 backdrop-blur-sm sm:px-6 sm:py-4"
			style:border-color="var(--border)"
			style:background-color="color-mix(in srgb, var(--bg-elevated) 85%, transparent)"
		>
			<div class="mx-auto flex max-w-5xl items-center justify-between gap-2">
				<a
					href={resolve('/')}
					class="flex min-w-0 items-center gap-2 text-lg font-semibold tracking-tight"
				>
					<AppIcon size={32} />
					<span class="truncate">{$_('app.title')}</span>
				</a>
				{#if $user}
					<div class="flex shrink-0 items-center gap-1">
						<div class="relative sm:hidden">
							<button
								type="button"
								class="btn-icon btn-ghost btn-nav"
								aria-expanded={navOpen}
								aria-haspopup="true"
								onclick={() => (navOpen = !navOpen)}
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
							{#if navOpen}
								<div
									class="popover-panel nav-mobile-panel absolute right-0 z-[60] mt-2 max-h-[min(70dvh,24rem)] min-w-[12rem] overflow-y-auto p-2"
								>
									{#each flatNavItems as item (item.labelKey)}
										<a
											href={item.href}
											class={navLinkClass(isNavItemActive(item), 'nav-mobile-link')}
											aria-current={isNavItemActive(item) ? 'page' : undefined}
											onclick={closeNav}>{$_(item.labelKey)}</a
										>
									{/each}
									<NavDropdown
										labelKey="nav.settings"
										items={settingsNavItems}
										isGroupActive={isSettingsGroupActive}
										mobile
										onNavigate={closeNav}
									/>
									{#if $user?.is_admin}
										<NavDropdown
											labelKey="nav.admin"
											items={adminNavItems}
											isGroupActive={isAdminGroupActive}
											mobile
											onNavigate={closeNav}
										/>
									{/if}
								</div>
							{/if}
						</div>
						<nav class="hidden items-center gap-2 sm:flex">
							{#each flatNavItems as item (item.labelKey)}
								{#if !item.mobileOnly}
									<a
										href={item.href}
										class={navLinkClass(isNavItemActive(item), 'btn-ghost btn-nav')}
										aria-current={isNavItemActive(item) ? 'page' : undefined}>{$_(item.labelKey)}</a
									>
								{/if}
							{/each}
							<NavDropdown
								labelKey="nav.settings"
								items={settingsNavItems}
								isGroupActive={isSettingsGroupActive}
							/>
							{#if $user?.is_admin}
								<NavDropdown
									labelKey="nav.admin"
									items={adminNavItems}
									isGroupActive={isAdminGroupActive}
								/>
							{/if}
						</nav>
						<IconButton
							icon="logout"
							label={$_('nav.logout')}
							class="btn-nav"
							onclick={handleLogout}
						/>
					</div>
				{/if}
			</div>
		</header>
		<main class="mx-auto max-w-5xl px-4 py-6 sm:px-6 sm:py-8">
			<AdminPasswordResetBanner />
			{@render children()}
		</main>
	</div>
{/if}

<ConfirmDialog />
{#if pendingUpdate && $user?.is_admin}
	<UpdateAvailableModal update={pendingUpdate} onclose={() => (pendingUpdate = null)} />
{/if}
<ToastContainer />
