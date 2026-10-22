<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import SectionNavHub from '$lib/components/SectionNavHub.svelte';
	import { androidAdminNavItems } from '$lib/android/nav-items';
	import { user } from '$lib/stores/auth';

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
		}
	});
</script>

<SectionNavHub items={androidAdminNavItems()} />
