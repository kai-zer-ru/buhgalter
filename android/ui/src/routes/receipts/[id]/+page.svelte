<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { detachReceipt, getReceipt, type ReceiptDetail } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { formatReceiptItemSum } from '$lib/receipt/receipt-items';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import { formatApiError } from '$lib/api/errors';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { onMount } from 'svelte';

	const receiptId = $derived($page.params.id);
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), '/transactions')
	);
	const transactionId = $derived($page.url.searchParams.get('transaction_id') ?? '');

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let detaching = $state(false);
	let error = $state('');
	let detail = $state<ReceiptDetail | null>(null);

	onMount(() => {
		void load();
	});

	async function load() {
		if (!receiptId) {
			loadError = $_('common.notFound');
			loading = false;
			return;
		}
		loading = true;
		loadError = null;
		error = '';
		detail = null;
		try {
			detail = await getReceipt(receiptId, transactionId || undefined);
		} catch (e) {
			loadError = reportPageLoadFailure(e);
		} finally {
			loading = false;
		}
	}

	async function handleDetach() {
		if (!transactionId || detaching) return;
		const ok = await confirm({
			title: $_('receipt.detach.title'),
			message: $_('receipt.detach.message'),
			confirmLabel: $_('receipt.detach.confirm'),
			cancelLabel: $_('common.cancel'),
			danger: true
		});
		if (!ok) return;
		detaching = true;
		error = '';
		try {
			await detachReceipt(transactionId);
			toast.success($_('receipt.detach.success'));
			dataRefreshTick.update((n) => n + 1);
			void leaveForm(returnTo);
		} catch (e) {
			error = formatApiError(e, 'receipt.detach.error');
		} finally {
			detaching = false;
		}
	}

	function handleClose() {
		void leaveForm(returnTo);
	}
</script>

<FormPageShell title={$_('receipt.view')} backHref={returnTo} onback={handleClose}>
	<PageLoadGate {loading} error={loadError} onretry={() => void load()}>
		{#if detail}
			<div class="space-y-4 p-4">
				<div class="text-sm">
					<p class="font-medium">{detail.store_name || '—'}</p>
					<p style:color="var(--text-muted)">
						{$_('receipt.docNumber')}: {detail.fiscal_document_number} ·
						<MoneyDisplay value={detail.total_sum_display} />
					</p>
					{#if detail.user_inn}
						<p style:color="var(--text-muted)">{$_('receipt.inn')}: {detail.user_inn}</p>
					{/if}
				</div>

				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr style:color="var(--text-muted)">
								<th class="p-2 text-left">{$_('receipt.col.item')}</th>
								<th class="p-2 text-right">{$_('receipt.col.qty')}</th>
								<th class="p-2 text-right">{$_('receipt.col.sum')}</th>
								<th class="p-2 text-left">{$_('transactions.col.category')}</th>
							</tr>
						</thead>
						<tbody>
							{#each detail.items as item, index (index)}
								<tr
									class="border-t"
									style:border-color="var(--border)"
									style:background={detail.highlight_item_index === index
										? 'var(--surface-muted, rgba(0,0,0,0.04))'
										: undefined}
									style:font-weight={detail.highlight_item_index === index ? '600' : undefined}
								>
									<td class="p-2">{item.name}</td>
									<td class="p-2 text-right tabular-nums">{item.quantity}</td>
									<td class="p-2 text-right tabular-nums">
										<MoneyDisplay value={formatReceiptItemSum(item.sum)} />
									</td>
									<td class="p-2 text-sm" style:color="var(--text-muted)">
										{#if item.category_name}
											{item.category_name}{item.subcategory_name
												? ` · ${item.subcategory_name}`
												: ''}
										{:else}
											—
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<div class="flex flex-wrap gap-2">
					<!-- eslint-disable svelte/no-navigation-without-resolve -- external FNS check URL -->
					<a
						class="btn-ghost inline-flex min-h-11 items-center"
						href={detail.fns_check_url}
						target="_blank"
						rel="noopener noreferrer"
					>
						{$_('receipt.openFns')}
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
					{#if transactionId}
						<button
							type="button"
							class="btn-ghost min-h-11 text-red-600"
							disabled={detaching}
							onclick={handleDetach}
						>
							{$_('receipt.detach.confirm')}
						</button>
					{/if}
				</div>
				{#if error}
					<p class="text-sm text-red-600">{error}</p>
				{/if}
			</div>
		{/if}
	</PageLoadGate>
</FormPageShell>
