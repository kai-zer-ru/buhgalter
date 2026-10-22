<script lang="ts">
	import { page } from '$app/stores';
	import { getAccount, type Account } from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import CreditCardFeeForm from '$lib/components/CreditCardFeeForm.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const accountId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/accounts/${accountId}`)
	);

	let account = $state<Account | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	$effect(() => {
		if (!accountId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			account = await getAccount(accountId);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!account });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if account}
		<CreditCardFeeForm
			variant="page"
			{account}
			backHref={returnTo}
			onclose={finish}
			onsaved={finish}
		/>
	{/if}
</PageLoadGate>
