<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { user } from '$lib/stores/auth';
	import { beginCreditCreate } from '$lib/credits/create-draft';
	import { goCreditCreateStep } from '$lib/credits/create-nav';

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromRaw = $derived($page.url.searchParams.get('from'));

	onMount(() => {
		beginCreditCreate(tz);
		goCreditCreateStep('basics', fromRaw);
	});
</script>
