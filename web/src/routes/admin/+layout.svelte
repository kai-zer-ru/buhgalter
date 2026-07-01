<script lang="ts">
	import { page } from '$app/stores';
	import { _, locale } from 'svelte-i18n';
	import BackLink, { type BreadcrumbItem } from '$lib/components/BackLink.svelte';
	import AdminSupportLinks from '$lib/components/AdminSupportLinks.svelte';

	let { children } = $props();

	const pages = [
		{ path: '/admin', titleKey: 'admin.tab.system' },
		{ path: '/admin/users', titleKey: 'admin.tab.users' },
		{ path: '/admin/backups', titleKey: 'admin.tab.backups' },
		{ path: '/admin/diagnostics', titleKey: 'admin.tab.diagnostics' }
	] as const;

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
		const home: BreadcrumbItem = { href: '/', label: $_('nav.home') };
		const admin: BreadcrumbItem = { href: '/admin', label: $_('admin.title') };
		if (current.path === '/admin') {
			return [home, admin];
		}
		return [
			home,
			admin,
			{ href: current.path as BreadcrumbItem['href'], label: $_(current.titleKey) }
		];
	});
</script>

<svelte:head>
	<title>{$_(current.titleKey)} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink items={breadcrumbItems} />
	<h1 class="text-2xl font-semibold">{$_(current.titleKey)}</h1>
	<div class="space-y-4">
		{@render children()}
		<AdminSupportLinks />
	</div>
</div>
