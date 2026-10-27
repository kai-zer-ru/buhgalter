<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { gotoReplace } from '$lib/android/form-nav';

	onMount(() => {
		const code = $page.params.code ?? '';
		void gotoReplace(`/settings/household/accept?code=${encodeURIComponent(code)}`);
	});
</script>
