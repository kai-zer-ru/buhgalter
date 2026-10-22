<script lang="ts">
	import { _ } from 'svelte-i18n';
	import BackLink from '$lib/components/BackLink.svelte';
	import TagsTab from '$lib/settings/TagsTab.svelte';
</script>

<svelte:head>
	<title>{$_('nav.tags')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/tags', label: $_('nav.tags') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('nav.tags')}</h1>
	<TagsTab />
</div>
