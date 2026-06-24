<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';

	onMount(() => {
		const url = new URL($page.url);
		const type = url.searchParams.get('type');
		const target = new URL(resolve('/settings'), url.origin);
		target.searchParams.set('tab', 'categories');
		if (type === 'income') {
			target.searchParams.set('type', 'income');
		}
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		void goto(`${target.pathname}${target.search}`, { replaceState: true });
	});
</script>
