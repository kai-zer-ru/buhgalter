<script lang="ts">
	import { page } from '$app/stores';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import BackLink, {
		type BackLinkHref,
		type BreadcrumbItem
	} from '$lib/components/BackLink.svelte';

	let { children } = $props();

	type SettingsPage = {
		path: string;
		titleKey: string;
		href: BackLinkHref;
		/** Page renders its own SectionHeader (e.g. with create CTA). */
		ownsHeader?: boolean;
	};

	const pages: SettingsPage[] = [
		{ path: '/settings', titleKey: 'settings.title', href: '/settings' },
		{ path: '/settings/profile', titleKey: 'settings.tab.profile', href: '/settings/profile' },
		{ path: '/settings/password', titleKey: 'settings.tab.password', href: '/settings/password' },
		{ path: '/settings/security', titleKey: 'settings.tab.security', href: '/settings/security' },
		{ path: '/settings/server', titleKey: 'settings.tab.server', href: '/settings/server' },
		{ path: '/settings/tokens', titleKey: 'settings.tab.tokens', href: '/settings/tokens' },
		{
			path: '/settings/notifications',
			titleKey: 'settings.tab.notifications',
			href: '/settings/notifications'
		},
		{ path: '/settings/import', titleKey: 'settings.tab.import', href: '/settings/import' }
	];

	const pathname = $derived($page.url.pathname);
	const isLegacyRedirect = $derived(
		pathname === '/settings/categories' || pathname === '/settings/recurring-operations'
	);

	const current = $derived.by(() => {
		return (
			pages.find((p) => p.path === pathname) ??
			pages.find((p) => pathname.startsWith(`${p.path}/`)) ??
			pages[0]
		);
	});

	const breadcrumbItems = $derived.by((): BreadcrumbItem[] => {
		void $locale;
		const home: BreadcrumbItem = { href: '/', label: tr('nav.home') };
		const settings: BreadcrumbItem = { href: '/settings', label: tr('settings.title') };
		const page = current;
		if (page.path === '/settings') {
			return [home, settings];
		}
		return [home, settings, { href: page.href, label: tr(page.titleKey) }];
	});
</script>

<svelte:head>
	<title>{$_(current.titleKey)} — {$_('app.title')}</title>
</svelte:head>

{#if isLegacyRedirect}
	{@render children()}
{:else}
	<div class="space-y-6">
		<BackLink items={breadcrumbItems} />
		{#if !current.ownsHeader}
			<h1 class="text-2xl font-semibold">{$_(current.titleKey)}</h1>
		{/if}
		{@render children()}
	</div>
{/if}
