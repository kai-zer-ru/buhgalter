<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	onMount(() => {
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		void goto(`${resolve('/settings')}?tab=accounts`, { replaceState: true });
	});
</script>
