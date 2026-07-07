<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getReceipt, type ReceiptDetail } from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { formatReceiptItemSum } from '$lib/receipt/receipt-items';

	type Props = {
		open: boolean;
		receiptImportId: string;
		transactionId?: string;
		onclose: () => void;
	};

	let {
		open = $bindable(),
		receiptImportId,
		transactionId = '',
		onclose
	}: Props = $props();

	let loading = $state(false);
	let error = $state('');
	let detail = $state<ReceiptDetail | null>(null);

	$effect(() => {
		if (!open || !receiptImportId) return;
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		detail = null;
		try {
			detail = await getReceipt(receiptImportId, transactionId || undefined);
		} catch (e) {
			error = e instanceof Error ? e.message : $_('receipt.loadError');
		} finally {
			loading = false;
		}
	}
</script>

<ModalShell bind:open title={$_('receipt.view')} maxWidth="max-w-2xl" {onclose}>
	{#if loading}
		<p class="text-sm">{$_('receipt.loading')}</p>
	{:else if error}
		<p class="text-sm text-red-600">{error}</p>
	{:else if detail}
		<div class="space-y-4">
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
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<a
				class="btn-ghost inline-flex min-h-11 items-center"
				href={detail.fns_check_url}
				target="_blank"
				rel="noopener noreferrer"
			>
				{$_('receipt.openFns')}
			</a>
		</div>
	{/if}
</ModalShell>
