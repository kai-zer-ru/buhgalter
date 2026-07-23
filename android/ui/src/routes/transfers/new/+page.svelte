<script lang="ts">
	import { page } from '$app/stores';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { getAccount, getTransaction, type Account, type Transaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { listIndexedTransferLegs, lookupServerTransaction } from '$lib/offline/transaction-index';

	import { dataRefreshTick } from '$lib/offline/sync';

	const accountId = $derived($page.url.searchParams.get('account') ?? '');
	const payCardId = $derived($page.url.searchParams.get('payCard'));
	const repeatId = $derived($page.url.searchParams.get('repeat'));
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));

	let creditCardPay = $state<Account | null>(null);
	let repeatFrom = $state<Transaction | null>(null);
	let siblings = $state<Transaction[]>([]);
	let ready = $state(false);

	$effect(() => {
		ready = false;
		void init();
	});

	async function init() {
		creditCardPay = null;
		repeatFrom = null;
		siblings = [];

		if (payCardId) {
			try {
				creditCardPay = await getAccount(payCardId);
			} catch {
				creditCardPay = null;
			}
		}

		if (repeatId) {
			try {
				repeatFrom = await getTransaction(repeatId);
			} catch {
				repeatFrom = lookupServerTransaction(repeatId);
			}
			if (repeatFrom?.transfer_group_id) {
				siblings = listIndexedTransferLegs(repeatFrom.transfer_group_id);
			}
		}

		ready = true;
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready}
	<TransferForm
		variant="page"
		backHref={returnTo}
		{accountId}
		{creditCardPay}
		{repeatFrom}
		{siblings}
		onclose={finish}
		onsaved={finish}
	/>
{/if}
