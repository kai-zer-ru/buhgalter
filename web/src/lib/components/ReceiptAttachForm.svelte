<script lang="ts">
	import { onDestroy } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		attachReceipt,
		listCategories,
		listSubcategories,
		parseReceipt,
		ApiError,
		type Category,
		type ParsedReceipt,
		type Subcategory,
		type Transaction
	} from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import Select from '$lib/components/Select.svelte';
	import ReceiptInlineSubcategoryField from '$lib/components/ReceiptInlineSubcategoryField.svelte';
	import ReceiptItemCategoryButton from '$lib/components/ReceiptItemCategoryButton.svelte';
	import { confirm } from '$lib/confirm';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import {
		applyCategoryToUnassigned,
		formatReceiptItemSum,
		includedRowsMissingCategory,
		initReceiptRows,
		type ReceiptRowState
	} from '$lib/receipt/receipt-items';
	import { categorySelectOptions } from '$lib/select-options';
	import { refsForAccount } from '$lib/household/refs-for-account';
	import {
		decodeQrFromImageFile,
		startQrScanner,
		type QrScannerHandle
	} from '$lib/receipt/parse-qr';
	import { clearActiveReceiptScanJobId } from '$lib/receipt/scan-job-session';
	import { toast } from '$lib/toast';
	import { formatApiError } from '$lib/api/errors';

	type Props = {
		open: boolean;
		transaction: Transaction | null;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), transaction = null, onclose, onsaved }: Props = $props();

	let step = $state<'input' | 'confirm'>('input');
	let loading = $state(false);
	let loadingPhase = $state<'scan' | 'fns' | null>(null);
	let scanPollingToken = 0;
	let saving = $state(false);
	let scanning = $state(false);
	let error = $state('');
	let receipt = $state<ParsedReceipt | null>(null);
	let rows = $state<ReceiptRowState[]>([]);
	let categories = $state<Category[]>([]);
	let bulkCategoryId = $state('');
	let bulkSubcategoryId = $state('');
	let subcategoriesByRow = $state<Record<number, Subcategory[]>>({});
	let bulkSubcategories = $state<Subcategory[]>([]);
	let scannerHandle = $state<QrScannerHandle | null>(null);
	let decodeRun = 0;
	let wasOpen = false;

	const busy = $derived(loading || scanning);
	const txAmount = $derived(transaction?.amount ?? 0);
	const categoryOptions = $derived(
		categorySelectOptions(refsForAccount(transaction, categories).filter((c) => !c.is_system))
	);
	const missingCategory = $derived(includedRowsMissingCategory(rows));
	const receiptBlocked = $derived(Boolean(receipt?.already_in_ledger));
	const modalMaxWidth = $derived(receiptBlocked ? 'max-w-lg' : 'max-w-6xl');

	$effect(() => {
		if (!open) {
			if (wasOpen) {
				scanPollingToken++;
				void stopScanner();
				softResetForm();
				loading = false;
				loadingPhase = null;
				clearActiveReceiptScanJobId();
			}
			wasOpen = false;
			return;
		}
		if (!wasOpen) {
			wasOpen = true;
			initOnOpen();
		}
	});

	onDestroy(() => {
		scanPollingToken++;
		void stopScanner();
	});

	function softResetForm() {
		step = 'input';
		saving = false;
		scanning = false;
		error = '';
		receipt = null;
		rows = [];
		bulkCategoryId = '';
		bulkSubcategoryId = '';
		subcategoriesByRow = {};
		bulkSubcategories = [];
	}

	function fullResetForm() {
		scanPollingToken++;
		softResetForm();
		loading = false;
		loadingPhase = null;
	}

	async function initOnOpen() {
		softResetForm();
		loading = false;
		loadingPhase = null;
		await loadMeta();
	}

	async function loadMeta() {
		categories = await listCategories('expense');
	}

	async function ensureMetaLoaded() {
		if (categories.length === 0) {
			await loadMeta();
		}
	}

	async function stopScanner() {
		if (scannerHandle) {
			await scannerHandle.stop().catch(() => undefined);
			scannerHandle = null;
		}
		scanning = false;
	}

	async function applyParsedReceipt(parsed: ParsedReceipt) {
		receipt = parsed;
		step = 'confirm';
		error = '';
		if (parsed.already_in_ledger) {
			return;
		}
		await ensureMetaLoaded();
		rows = initReceiptRows(parsed.items.length);
		const txCat = transaction?.category_id ?? '';
		const txSub = transaction?.subcategory_id ?? '';
		if (txCat) {
			rows = rows.map((row) => ({
				...row,
				categoryId: txCat,
				subcategoryId: txSub
			}));
			const stubName = transaction?.subcategory_name?.trim() ?? '';
			subcategoriesByRow =
				txSub && stubName
					? Object.fromEntries(
							rows.map((row) => [
								row.index,
								[
									{
										id: txSub,
										category_id: txCat,
										name: stubName,
										icon: transaction?.subcategory_icon || 'default',
										sort_order: 0,
										created_at: ''
									}
								]
							])
						)
					: {};
			for (const row of rows) {
				void loadSubcategories(row.index, txCat);
			}
		}
		bulkCategoryId = '';
		bulkSubcategoryId = '';
		bulkSubcategories = [];
	}

	let subcategoryLoadGen: Record<number, number> = {};

	async function loadSubcategories(rowIndex: number, categoryId: string) {
		const gen = (subcategoryLoadGen[rowIndex] ?? 0) + 1;
		subcategoryLoadGen[rowIndex] = gen;
		if (!categoryId) {
			subcategoriesByRow = { ...subcategoriesByRow, [rowIndex]: [] };
			return;
		}
		try {
			const subs = await listSubcategories(categoryId);
			if (subcategoryLoadGen[rowIndex] !== gen) return;
			const selectedId = rows.find((item) => item.index === rowIndex)?.subcategoryId ?? '';
			const kept = (subcategoriesByRow[rowIndex] ?? []).find((sub) => sub.id === selectedId);
			const next =
				!selectedId || !kept || subs.some((sub) => sub.id === selectedId) ? subs : [...subs, kept];
			subcategoriesByRow = { ...subcategoriesByRow, [rowIndex]: next };
		} catch {
			// Сеть не стирает уже показанное имя подкатегории на кнопке.
		}
	}

	async function loadBulkSubcategories(categoryId: string) {
		if (!categoryId) {
			bulkSubcategories = [];
			return;
		}
		bulkSubcategories = await listSubcategories(categoryId);
	}

	function applyBulkToUnassigned() {
		rows = applyCategoryToUnassigned(rows, bulkCategoryId, bulkSubcategoryId);
		if (!bulkCategoryId) return;
		for (const row of rows) {
			if (row.include && row.categoryId === bulkCategoryId) {
				if (bulkSubcategories.length) {
					subcategoriesByRow = { ...subcategoriesByRow, [row.index]: bulkSubcategories };
				}
				void loadSubcategories(row.index, bulkCategoryId);
			}
		}
	}

	function attachItemsPayload() {
		return rows.map((row) => ({
			index: row.index,
			category_id: row.categoryId || undefined,
			subcategory_id: row.subcategoryId || null,
			include: true
		}));
	}

	function showScanError(err: unknown) {
		const message = formatApiError(err, 'receipt.parseError');
		error = message;
		if (!open) {
			toast.error(message);
		}
	}

	async function handleFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file || busy) return;
		await scanPhotoAndParse(file);
	}

	async function scanPhotoAndParse(file: File) {
		scanPollingToken++;
		const pollToken = scanPollingToken;
		const run = ++decodeRun;
		loading = true;
		loadingPhase = 'scan';
		error = '';
		try {
			const qr = await decodeQrFromImageFile(file);
			if (run !== decodeRun || pollToken !== scanPollingToken) return;
			if (qr) {
				loadingPhase = 'fns';
				try {
					const parsed = await parseReceipt(qr);
					if (run !== decodeRun || pollToken !== scanPollingToken) return;
					await applyParsedReceipt(parsed);
				} catch (e) {
					if (run !== decodeRun || pollToken !== scanPollingToken) return;
					showScanError(e);
				} finally {
					if (run === decodeRun && pollToken === scanPollingToken) {
						loading = false;
						loadingPhase = null;
					}
				}
				return;
			}
			error = $_('receipt.parseError');
			loading = false;
			loadingPhase = null;
		} catch (e) {
			if (run !== decodeRun || pollToken !== scanPollingToken) return;
			showScanError(e);
			loading = false;
			loadingPhase = null;
		}
	}

	async function toggleScanner() {
		if (busy) return;
		if (scanning) {
			await stopScanner();
			return;
		}
		error = '';
		scanning = true;
		try {
			scannerHandle = await startQrScanner('receipt-attach-qr-reader', async (text) => {
				await stopScanner();
				await decodeAndParse(async () => text);
			});
		} catch {
			error = $_('receipt.scanError');
			scanning = false;
		}
	}

	async function decodeAndParse(getQr: () => Promise<string>) {
		const run = ++decodeRun;
		loading = true;
		loadingPhase = 'fns';
		error = '';
		try {
			const qr = await getQr();
			if (run !== decodeRun) return;
			const parsed = await parseReceipt(qr);
			if (run !== decodeRun) return;
			await applyParsedReceipt(parsed);
		} catch (e) {
			if (run !== decodeRun) return;
			showScanError(e);
		} finally {
			if (run === decodeRun) {
				loading = false;
				loadingPhase = null;
			}
		}
	}

	async function confirmMismatchIfNeeded(): Promise<boolean> {
		if (!receipt || !transaction) return false;
		if (txAmount === receipt.total_sum) return true;
		return confirm({
			title: $_('receipt.attach.sumMismatch.title'),
			message: $_('receipt.attach.sumMismatch.message', {
				values: {
					txAmount: formatMoneyForDisplay({ cents: txAmount }),
					receiptAmount: formatMoneyForDisplay({ cents: receipt.total_sum })
				}
			}),
			confirmLabel: $_('receipt.attach.anyway'),
			cancelLabel: $_('common.cancel')
		});
	}

	async function save() {
		if (!receipt || !transaction || missingCategory) return;
		saving = true;
		error = '';
		try {
			let acknowledgeSumMismatch = false;
			if (txAmount !== receipt.total_sum) {
				const ok = await confirmMismatchIfNeeded();
				if (!ok) return;
				acknowledgeSumMismatch = true;
			}
			await attachReceipt({
				transaction_id: transaction.id,
				fiscal_drive_number: receipt.fiscal_drive_number,
				fiscal_document_number: receipt.fiscal_document_number,
				fiscal_sign: receipt.fiscal_sign,
				receipt,
				items: attachItemsPayload(),
				acknowledge_sum_mismatch: acknowledgeSumMismatch
			});
			toast.success($_('receipt.attach.success'));
			clearActiveReceiptScanJobId();
			fullResetForm();
			open = false;
			onclose();
			onsaved();
		} catch (e) {
			if (e instanceof ApiError && e.code === 'ERR_RECEIPT_SUM_MISMATCH') {
				const ok = await confirm({
					title: $_('receipt.attach.sumMismatch.title'),
					message: formatApiError(e, 'receipt.attach.sumMismatch.message'),
					confirmLabel: $_('receipt.attach.anyway'),
					cancelLabel: $_('common.cancel')
				});
				if (!ok) return;
				try {
					await attachReceipt({
						transaction_id: transaction.id,
						fiscal_drive_number: receipt.fiscal_drive_number,
						fiscal_document_number: receipt.fiscal_document_number,
						fiscal_sign: receipt.fiscal_sign,
						receipt,
						items: attachItemsPayload(),
						acknowledge_sum_mismatch: true
					});
					toast.success($_('receipt.attach.success'));
					clearActiveReceiptScanJobId();
					fullResetForm();
					open = false;
					onclose();
					onsaved();
					return;
				} catch (retryErr) {
					error = formatApiError(retryErr, 'receipt.attach.error');
					return;
				}
			}
			error = formatApiError(e, 'receipt.attach.error');
		} finally {
			saving = false;
		}
	}

	function handleClose() {
		scanPollingToken++;
		void stopScanner();
		open = false;
		onclose();
	}
</script>

<ModalShell
	bind:open
	title={$_('receipt.attach.title')}
	maxWidth={modalMaxWidth}
	onclose={handleClose}
>
	{#if step === 'input'}
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">{$_('receipt.intro')}</p>
			<div class="flex flex-wrap items-center gap-2">
				<button
					type="button"
					class="btn-primary btn-inline inline-flex min-h-11 items-center justify-center"
					disabled={busy}
					onclick={toggleScanner}
				>
					{scanning ? $_('receipt.stopScan') : $_('receipt.scan')}
				</button>
				<label
					class="btn-ghost btn-inline btn-file inline-flex min-h-11 cursor-pointer items-center justify-center"
					class:opacity-50={busy}
					class:pointer-events-none={busy}
					aria-disabled={busy}
				>
					{$_('receipt.upload')}
					<input type="file" accept="image/*" disabled={busy} onchange={handleFileChange} />
				</label>
			</div>
			{#if scanning}
				<div
					id="receipt-attach-qr-reader"
					class="overflow-hidden rounded-lg border"
					style:border-color="var(--border)"
				></div>
			{/if}
			{#if loading}
				<p class="text-sm" role="status" aria-live="polite">
					{loadingPhase === 'scan' ? $_('receipt.decodingQr') : $_('receipt.loading')}
				</p>
			{/if}
			{#if error}
				<p class="text-sm text-red-600">{error}</p>
			{/if}
		</div>
	{:else if receipt}
		{#if receiptBlocked}
			<div class="space-y-4">
				<p class="text-sm text-amber-700" role="alert">{$_('receipt.alreadyInLedger')}</p>
				<div class="flex justify-end">
					<button type="button" class="btn-ghost min-h-11" onclick={() => (step = 'input')}>
						{$_('common.back')}
					</button>
				</div>
			</div>
		{:else}
			<div class="space-y-4">
				<div class="rounded-lg border p-3 text-sm" style:border-color="var(--border)">
					<p class="font-medium">{receipt.store_name || '—'}</p>
					<p style:color="var(--text-muted)">
						{$_('receipt.docNumber')}: {receipt.fiscal_document_number} ·
						{formatMoneyForDisplay({ cents: receipt.total_sum })}
					</p>
					{#if transaction && txAmount !== receipt.total_sum}
						<p class="mt-2 text-amber-700">
							{$_('receipt.attach.sumHint', {
								values: {
									txAmount: formatMoneyForDisplay({ cents: txAmount }),
									receiptAmount: formatMoneyForDisplay({ cents: receipt.total_sum })
								}
							})}
						</p>
					{/if}
				</div>

				<div class="space-y-2 rounded-lg border p-3" style:border-color="var(--border)">
					<p class="text-sm font-medium">{$_('receipt.bulkCategory')}</p>
					<p class="text-xs" style:color="var(--text-muted)">{$_('receipt.bulkCategoryHint')}</p>
					<div class="flex flex-wrap items-end gap-2">
						<div class="min-w-[12rem] flex-1">
							<Select
								id="receipt-attach-bulk-category"
								label={$_('transactions.field.category')}
								placeholder={$_('receipt.selectCategory')}
								bind:value={bulkCategoryId}
								options={categoryOptions}
								usePortal
								onchange={() => {
									bulkSubcategoryId = '';
									void loadBulkSubcategories(bulkCategoryId);
								}}
							/>
						</div>
						<div class="min-w-[14rem] flex-1">
							<span class="mb-1 block text-sm font-medium"
								>{$_('transactions.field.subcategory')}</span
							>
							<ReceiptInlineSubcategoryField
								id="receipt-attach-bulk-subcategory"
								categoryId={bulkCategoryId}
								bind:value={bulkSubcategoryId}
								subcategories={bulkSubcategories}
								onSubcategoriesChange={(subs) => {
									bulkSubcategories = subs;
								}}
							/>
						</div>
						<button
							type="button"
							class="btn-primary min-h-11"
							disabled={!bulkCategoryId}
							onclick={applyBulkToUnassigned}
						>
							{$_('receipt.applyUnassigned')}
						</button>
					</div>
				</div>

				<table class="w-full table-fixed text-sm">
					<thead>
						<tr style:color="var(--text-muted)">
							<th class="p-2 text-left">{$_('receipt.col.item')}</th>
							<th class="w-[22%] p-2 text-right">{$_('receipt.col.sum')}</th>
							<th class="w-[38%] p-2 text-left"
								><span class="sr-only">{$_('transactions.field.category')}</span></th
							>
						</tr>
					</thead>
					<tbody>
						{#each rows as row (row.index)}
							<tr class="border-t" style:border-color="var(--border)">
								<td class="min-w-0 p-2 break-words">{receipt.items[row.index]?.name ?? ''}</td>
								<td class="p-2 text-right tabular-nums">
									<MoneyDisplay value={formatReceiptItemSum(receipt.items[row.index]?.sum ?? 0)} />
								</td>
								<td class="min-w-0 p-2">
									<ReceiptItemCategoryButton
										buttonId={`receipt-attach-item-cat-${row.index}`}
										categorySelectId={`receipt-attach-cat-${row.index}`}
										subcategoryFieldId={`receipt-attach-subcat-${row.index}`}
										itemName={receipt.items[row.index]?.name ?? ''}
										bind:categoryId={row.categoryId}
										bind:subcategoryId={row.subcategoryId}
										{categoryOptions}
										subcategories={subcategoriesByRow[row.index] ?? []}
										onCategoryChange={(categoryId) => {
											row.subcategoryId = '';
											void loadSubcategories(row.index, categoryId);
										}}
										onSubcategoriesChange={(subs) => {
											subcategoriesByRow = { ...subcategoriesByRow, [row.index]: subs };
										}}
									/>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>

				{#if missingCategory}
					<p class="text-sm text-amber-600">{$_('receipt.categoryRequired')}</p>
				{/if}
				{#if error}
					<p class="text-sm text-red-600">{error}</p>
				{/if}

				<div class="flex justify-end gap-2">
					<button type="button" class="btn-ghost min-h-11" onclick={() => (step = 'input')}>
						{$_('common.back')}
					</button>
					<button
						type="button"
						class="btn-primary min-h-11"
						disabled={saving || missingCategory}
						onclick={save}
					>
						{$_('receipt.attach.confirm')}
					</button>
				</div>
			</div>
		{/if}
	{/if}
</ModalShell>
