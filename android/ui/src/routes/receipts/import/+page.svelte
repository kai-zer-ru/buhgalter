<script lang="ts">
	import { page } from '$app/stores';
	import ReceiptScanForm from '$lib/components/ReceiptScanForm.svelte';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { requireOnline } from '$lib/offline/require-online';
	import { onMount } from 'svelte';

	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));
	const accountId = $derived($page.url.searchParams.get('account') ?? '');

	onMount(() => {
		if (!requireOnline()) {
			void leaveForm(returnTo);
		}
	});

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<ReceiptScanForm
	mode="import"
	backHref={returnTo}
	{accountId}
	onclose={() => void leaveForm(returnTo)}
	onsaved={finish}
/>
