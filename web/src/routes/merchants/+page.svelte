<script lang="ts">
	import { _ } from 'svelte-i18n';
	import BackLink from '$lib/components/BackLink.svelte';
	import MerchantsTab from '$lib/settings/MerchantsTab.svelte';
</script>

<svelte:head>
	<title>{$_('nav.merchants')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/merchants', label: $_('nav.merchants') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('nav.merchants')}</h1>
	<MerchantsTab />
</div>
