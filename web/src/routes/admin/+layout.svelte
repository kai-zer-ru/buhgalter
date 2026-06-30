<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _, locale } from 'svelte-i18n';
	import BackLink, { type BreadcrumbItem } from '$lib/components/BackLink.svelte';
	import AdminSupportLinks from '$lib/components/AdminSupportLinks.svelte';

	let { children } = $props();

	const tabs = [
		{ href: resolve('/admin'), path: '/admin', label: 'admin.tab.system', exact: true },
		{ href: resolve('/admin/users'), path: '/admin/users', label: 'admin.tab.users', exact: false },
		{
			href: resolve('/admin/backups'),
			path: '/admin/backups',
			label: 'admin.tab.backups',
			exact: false
		},
		{
			href: resolve('/admin/diagnostics'),
			path: '/admin/diagnostics',
			label: 'admin.tab.diagnostics',
			exact: false
		}
	] as const;

	function isActive(path: string, exact: boolean) {
		if (exact) return $page.url.pathname === path;
		return $page.url.pathname.startsWith(path);
	}

	const breadcrumbItems = $derived.by((): BreadcrumbItem[] => {
		void $locale;
		const items: BreadcrumbItem[] = [
			{ href: '/', label: $_('nav.home') },
			{ href: '/settings', label: $_('settings.title'), search: 'tab=admin' },
			{ href: '/admin', label: $_('admin.title') }
		];
		const active = tabs.find((t) => isActive(t.path, t.exact));
		if (active && active.path !== '/admin') {
			items.push({
				href: active.path as BreadcrumbItem['href'],
				label: $_(active.label)
			});
		}
		return items;
	});
</script>

<svelte:head>
	<title>{$_('admin.title')} — {$_('app.title')}</title>
</svelte:head>

<BackLink items={breadcrumbItems} />

<h1 class="mb-6 text-2xl font-semibold">{$_('admin.title')}</h1>

<div class="mb-6 flex flex-wrap gap-2">
	{#each tabs as t (t.path)}
		<a href={t.href} class="tab {isActive(t.path, t.exact) ? 'tab-active' : ''}">
			{$_(t.label)}
		</a>
	{/each}
</div>

<div class="space-y-4">
	{@render children()}
	<AdminSupportLinks />
</div>
