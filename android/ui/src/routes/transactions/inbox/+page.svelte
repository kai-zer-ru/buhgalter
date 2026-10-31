<script lang="ts">
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { createCategoryRule, listInboxTransactions, type Transaction } from '$lib/api/client';
	import { defaultPatternFromDescription } from '$lib/category-rule-pattern';
	import BackLink from '$lib/components/BackLink.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import TransactionList from '$lib/components/TransactionList.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let transactions = $state<Transaction[]>([]);
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	let total = $state(0);
	let page = $state(1);
	const limit = 20;
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let editTx = $state<Transaction | null>(null);
	let txOpen = $state(false);
	let saveAsRule = $state(false);
	let pendingRuleDesc = $state('');

	onMount(() => void load());

	async function load(opts: { silent?: boolean } = {}) {
		if (!opts?.silent) {
			loading = true;
			loadError = null;
		}
		try {
			const res = await listInboxTransactions({ page, limit });
			transactions = res.data ?? [];
			total = res.meta?.total ?? 0;
		} catch (e) {
			if (!opts?.silent) loadError = reportPageLoadFailure(e);
		} finally {
			if (!opts?.silent) loading = false;
		}
	}

	function openEdit(tx: Transaction) {
		editTx = tx;
		saveAsRule = false;
		pendingRuleDesc = defaultPatternFromDescription(tx.description ?? '');
		txOpen = true;
	}

	async function onSavedFromTx(tx: import('$lib/api/client').Transaction) {
		await load({ silent: true });
		if (saveAsRule && tx.category_id) {
			const pat = pendingRuleDesc.trim();
			if (pat) {
				try {
					await createCategoryRule({
						kind: 'text_category',
						category_id: tx.category_id,
						subcategory_id: tx.subcategory_id ?? undefined,
						pattern: pat
					});
					toast.success($_('categoryRules.savedFromInbox'));
				} catch (err) {
					toast.fromError(err);
				}
			}
		}
		editTx = null;
	}
</script>

<svelte:head>
	<title>{$_('inbox.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="space-y-6">
	<BackLink
		items={[
			{ href: '/', label: $_('nav.home') },
			{ href: '/transactions', label: $_('nav.transactions') },
			{ href: '/transactions/inbox', label: $_('inbox.title') }
		]}
	/>
	<h1 class="text-2xl font-semibold">{$_('inbox.title')}</h1>
	<p class="text-sm" style:color="var(--text-muted)">{$_('inbox.hint')}</p>

	<PageLoadGate {loading} error={loadError} onretry={() => void load()}>
		{#if transactions.length > 0}
			<div class="card overflow-x-auto p-0 sm:p-0">
				<TransactionList
					{transactions}
					{tz}
					emptyMessage={$_('inbox.empty')}
					showEdit
					showDescription
					showCategory={false}
					onedit={openEdit}
				/>
			</div>
		{:else}
			<TransactionList
				{transactions}
				{tz}
				emptyMessage={$_('inbox.empty')}
				showEdit
				showDescription
				showCategory={false}
				onedit={openEdit}
			/>
		{/if}
		<TransactionPagination
			{page}
			{limit}
			{total}
			onchange={(p) => {
				page = p;
				void load({ silent: true });
			}}
		/>
	</PageLoadGate>
</div>

{#if txOpen && editTx}
	<div class="mb-4 space-y-2 rounded-xl border p-3" style:border-color="var(--border)">
		<div class="flex items-center justify-between gap-4">
			<span class="min-w-0 text-sm">
				<span class="font-medium">{$_('inbox.saveAsRule')}</span>
			</span>
			<ToggleSwitch
				checked={saveAsRule}
				label={$_('inbox.saveAsRule')}
				onchange={() => (saveAsRule = !saveAsRule)}
			/>
		</div>
		{#if saveAsRule}
			<label class="block space-y-1 text-sm">
				<span>{$_('categoryRules.field.pattern')}</span>
				<input class="input w-full" bind:value={pendingRuleDesc} maxlength={200} />
			</label>
		{/if}
	</div>
{/if}

<TransactionForm
	bind:open={txOpen}
	transaction={editTx}
	onclose={() => (editTx = null)}
	onsaved={() => {}}
	onsavedTx={(tx) => void onSavedFromTx(tx)}
/>
