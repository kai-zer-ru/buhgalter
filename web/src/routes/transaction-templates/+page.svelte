<script lang="ts">
	import { _ } from 'svelte-i18n';
	import BackLink from '$lib/components/BackLink.svelte';
	import TransactionTemplatesTab from '$lib/settings/TransactionTemplatesTab.svelte';
</script>

<svelte:head>
	<title>{$_('settings.tab.templates')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/transaction-templates', label: $_('settings.tab.templates') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('settings.tab.templates')}</h1>
	<TransactionTemplatesTab />
</div>
