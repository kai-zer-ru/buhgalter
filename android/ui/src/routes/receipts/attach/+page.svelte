<script lang="ts">
	import { page } from '$app/stores';
	import ReceiptScanForm from '$lib/components/ReceiptScanForm.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { getTransaction, type Transaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { requireOnline } from '$lib/offline/require-online';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';

	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), '/transactions')
	);
	const attachTxId = $derived($page.url.searchParams.get('attach_tx') ?? '');

	let transaction = $state<Transaction | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	onMount(() => {
		if (!requireOnline()) {
			void leaveForm(returnTo);
			return;
		}
		if (!attachTxId) {
			loadError = $_('common.notFound');
			loading = false;
			return;
		}
		void loadTx();
	});

	async function loadTx() {
		loading = true;
		loadError = null;
		try {
			transaction = await getTransaction(attachTxId);
		} catch (e) {
			loadError = reportPageLoadFailure(e);
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void loadTx()}>
	{#if transaction}
		<ReceiptScanForm
			mode="attach"
			backHref={returnTo}
			{transaction}
			onclose={() => void leaveForm(returnTo)}
			onsaved={finish}
		/>
	{/if}
</PageLoadGate>
