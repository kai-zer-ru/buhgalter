<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import { getTransaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { refreshMergeMeta, mergeMetaAccounts, mergeMetaCategories } from '$lib/offline/merge';
	import { lookupPendingTransaction } from '$lib/offline/pending-display';
	import { lookupServerTransaction } from '$lib/offline/transaction-index';
	import { outboxTick } from '$lib/offline/store';
	import { dataRefreshTick, localDataTick } from '$lib/offline/sync';
	import type { Transaction } from '$lib/api/client';

	const id = $derived($page.params.id);
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));

	let transaction = $state<Transaction | null>(null);
	let ready = $state(false);

	$effect(() => {
		void $outboxTick;
		void $localDataTick;
		if (!id) return;
		ready = false;
		void load(id);
	});

	async function load(txId: string) {
		await refreshMergeMeta().catch(() => {});
		try {
			transaction = await getTransaction(txId);
		} catch {
			const accounts = mergeMetaAccounts();
			const categories = mergeMetaCategories();
			transaction =
				lookupPendingTransaction(txId, accounts, categories) ?? lookupServerTransaction(txId);
		} finally {
			ready = true;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready && transaction}
	<TransactionForm
		variant="page"
		backHref={returnTo}
		{transaction}
		onclose={finish}
		onsaved={finish}
	/>
{:else if ready}
	<p class="p-4 text-sm" style:color="var(--text-muted)">{$_('common.notFound')}</p>
{/if}
