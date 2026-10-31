<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { detachReceipt, getReceipt, type ReceiptDetail } from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { confirm } from '$lib/confirm';
	import { formatReceiptItemSum } from '$lib/receipt/receipt-items';
	import { toast } from '$lib/toast';
	import { formatApiError } from '$lib/api/errors';

	type Props = {
		open: boolean;
		receiptImportId: string;
		transactionId?: string;
		canDetach?: boolean;
		onclose: () => void;
		ondetached?: () => void;
	};

	let {
		open = $bindable(),
		receiptImportId,
		transactionId = '',
		canDetach = false,
		onclose,
		ondetached
	}: Props = $props();

	let loading = $state(false);
	let detaching = $state(false);
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
			open = false;
			onclose();
			ondetached?.();
		} catch (e) {
			error = formatApiError(e, 'receipt.detach.error');
		} finally {
			detaching = false;
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
				<table class="data-table w-full text-sm">
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
										{item.category_name}{item.subcategory_name ? ` · ${item.subcategory_name}` : ''}
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
				{#if canDetach && transactionId}
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
		</div>
	{/if}
</ModalShell>
