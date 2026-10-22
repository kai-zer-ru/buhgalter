<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { user } from '$lib/stores/auth';
	import { beginCreditCreate } from '$lib/credits/create-draft';
	import { goCreditCreateStep } from '$lib/credits/create-nav';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { requireOnline } from '$lib/offline/require-online';

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromRaw = $derived($page.url.searchParams.get('from'));
	const returnTo = $derived(parseFormReturnPath(fromRaw, '/credits'));

	onMount(() => {
		if (!requireOnline('offline.onlineOnly.creditCreate')) {
			void leaveForm(returnTo);
			return;
		}
		beginCreditCreate(tz);
		goCreditCreateStep('basics', fromRaw);
	});
</script>
