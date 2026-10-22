<script lang="ts">
	import { _ } from 'svelte-i18n';
	import BackLink from '$lib/components/BackLink.svelte';
	import CategoryRulesTab from '$lib/settings/CategoryRulesTab.svelte';
</script>

<svelte:head>
	<title>{$_('categoryRules.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/category-rules', label: $_('categoryRules.title') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('categoryRules.title')}</h1>
	<CategoryRulesTab />
</div>
