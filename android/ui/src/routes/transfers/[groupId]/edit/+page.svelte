<script lang="ts">
	import { page } from '$app/stores';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { listTransactions, type Transaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { mergeMetaAccounts, mergeMetaCategories, refreshMergeMeta } from '$lib/offline/merge';
	import {
		lookupPendingTransferLegs,
		lookupPendingTransaction
	} from '$lib/offline/pending-display';
	import { listIndexedTransferLegs, lookupServerTransaction } from '$lib/offline/transaction-index';
	import { outboxTick } from '$lib/offline/store';
	import { isLocalEntityKey } from '$lib/offline/types';

	import { _ } from 'svelte-i18n';
	import { dataRefreshTick, localDataTick } from '$lib/offline/sync';

	const groupId = $derived($page.params.groupId);
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));

	let editTx = $state<Transaction | null>(null);
	let siblings = $state<Transaction[]>([]);
	let ready = $state(false);

	$effect(() => {
		void $outboxTick;
		void $localDataTick;
		if (!groupId) return;
		ready = false;
		void load(groupId);
	});

	async function load(id: string) {
		await refreshMergeMeta().catch(() => {});
		const accounts = mergeMetaAccounts();
		const categories = mergeMetaCategories();

		if (isLocalEntityKey(id)) {
			siblings = lookupPendingTransferLegs(id, accounts, categories);
			editTx = lookupPendingTransaction(id, accounts, categories);
			ready = true;
			return;
		}

		editTx = lookupServerTransaction(id);
		siblings = listIndexedTransferLegs(id);

		if (!editTx || siblings.length < 2) {
			try {
				const res = await listTransactions({ type: 'transfer', limit: '100' });
				siblings = res.data.filter((tx) => tx.transfer_group_id === id);
				editTx =
					siblings.find((tx) => tx.transfer_is_out) ??
					siblings.find((tx) => tx.type === 'transfer') ??
					siblings[0] ??
					null;
			} catch {
				siblings = lookupPendingTransferLegs(id, accounts, categories);
				editTx = lookupPendingTransaction(id, accounts, categories);
			}
		}

		ready = true;
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready && editTx}
	<TransferForm
		variant="page"
		backHref={returnTo}
		{editTx}
		{siblings}
		onclose={finish}
		onsaved={finish}
	/>
{:else if ready}
	<p class="p-4 text-sm" style:color="var(--text-muted)">{$_('common.notFound')}</p>
{/if}
