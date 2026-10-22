<script lang="ts">
	import { _ } from 'svelte-i18n';
	import BackLink from '$lib/components/BackLink.svelte';
	import CategoriesTab from '$lib/settings/CategoriesTab.svelte';
</script>

<svelte:head>
	<title>{$_('nav.categories')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/categories', label: $_('nav.categories') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('nav.categories')}</h1>
	<CategoriesTab />
</div>
