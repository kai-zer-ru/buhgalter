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
		{ path: '/settings', titleKey: 'settings.tab.profile', href: '/settings' },
		{ path: '/settings/password', titleKey: 'settings.tab.password', href: '/settings/password' },
		{ path: '/settings/tokens', titleKey: 'settings.tab.tokens', href: '/settings/tokens' },
		{
			path: '/settings/notifications',
			titleKey: 'settings.tab.notifications',
			href: '/settings/notifications'
		},
		{
			path: '/settings/categories',
			titleKey: 'settings.tab.categories',
			href: '/settings/categories'
		},
		{ path: '/settings/import', titleKey: 'settings.tab.import', href: '/settings/import' },
		{
			path: '/settings/recurring-operations',
			titleKey: 'nav.recurring',
			href: '/settings/recurring-operations',
			ownsHeader: true
		}
	];

	const current = $derived.by(() => {
		const pathname = $page.url.pathname;
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

<div class="space-y-6">
	<BackLink items={breadcrumbItems} />
	{#if !current.ownsHeader}
		<h1 class="text-2xl font-semibold">{$_(current.titleKey)}</h1>
	{/if}
	{@render children()}
</div>
