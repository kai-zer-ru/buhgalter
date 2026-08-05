<script lang="ts">
	import { page } from '$app/stores';
	import SubscriptionForm from '$lib/components/SubscriptionForm.svelte';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { dataRefreshTick } from '$lib/offline/sync';

	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), '/subscriptions')
	);
	const fromTxId = $derived($page.url.searchParams.get('from_tx') ?? '');

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<SubscriptionForm
	backHref={returnTo}
	fromTxId={fromTxId || undefined}
	onclose={finish}
	onsaved={finish}
/>
