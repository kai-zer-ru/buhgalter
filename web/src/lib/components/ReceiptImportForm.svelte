<script lang="ts">
	import { onDestroy } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		importReceipt,
		listAccounts,
		listCategories,
		listSubcategories,
		parseReceipt,
		createReceiptScanJob,
		getReceiptScanJob,
		pollReceiptScanJob,
		ApiError,
		type Account,
		type Category,
		type ParsedReceipt,
		type Subcategory
	} from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import Select from '$lib/components/Select.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import {
		applyCategoryToSelected,
		countIncludedRows,
		formatReceiptItemSum,
		includedRowsMissingCategory,
		initReceiptRows,
		type ReceiptRowState
	} from '$lib/receipt/receipt-items';
	import { startQrScanner, type QrScannerHandle } from '$lib/receipt/parse-qr';
	import {
		clearActiveReceiptScanJobId,
		getActiveReceiptScanJobId,
		saveActiveReceiptScanJobId
	} from '$lib/receipt/scan-job-session';
	import {
		accountSelectOptions,
		categorySelectOptions,
		subcategorySelectOptions
	} from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { formatApiError } from '$lib/api/errors';

	type Props = {
		open: boolean;
		accountId?: string;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), accountId = '', onclose, onsaved }: Props = $props();

	let step = $state<'input' | 'confirm'>('input');
	let loading = $state(false);
	let loadingPhase = $state<'scan' | 'fns' | null>(null);
	let scanPollingToken = 0;
	let pollingJobId = $state<string | null>(null);
	let saving = $state(false);
	let scanning = $state(false);
	let error = $state('');
	let receipt = $state<ParsedReceipt | null>(null);
	let rows = $state<ReceiptRowState[]>([]);
	let selectedAccount = $state('');
	let bulkCategoryId = $state('');
	let bulkSubcategoryId = $state('');
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let subcategoriesByRow = $state<Record<number, Subcategory[]>>({});
	let scannerHandle = $state<QrScannerHandle | null>(null);
	let decodeRun = 0;

	const busy = $derived(loading || scanning);
	const categoryOptions = $derived(categorySelectOptions(categories.filter((c) => !c.is_system)));
	const accountOptions = $derived(accountSelectOptions(accounts));
	const includedCount = $derived(countIncludedRows(rows));
	const missingCategory = $derived(includedRowsMissingCategory(rows));

	$effect(() => {
		if (!open) {
			void stopScanner();
			return;
		}
		void initOnOpen();
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
	}

	function fullResetForm() {
		scanPollingToken++;
		softResetForm();
		loading = false;
		loadingPhase = null;
		pollingJobId = null;
	}

	async function initOnOpen() {
		await loadMeta();
		const storedJobId = getActiveReceiptScanJobId();
		if (storedJobId) {
			await restoreActiveScanJob(storedJobId);
			return;
		}
		if (step === 'input' && !loading && !pollingJobId && !error && !receipt) {
			softResetForm();
		}
	}

	async function loadMeta() {
		const [accs, cats] = await Promise.all([listAccounts('active'), listCategories('expense')]);
		accounts = accs;
		categories = cats;
		selectedAccount = defaultAccountId(accs, accountId);
	}

	async function stopScanner() {
		if (scannerHandle) {
			await scannerHandle.stop().catch(() => undefined);
			scannerHandle = null;
		}
		scanning = false;
	}

	function applyParsedReceipt(parsed: ParsedReceipt) {
		receipt = parsed;
		rows = initReceiptRows(parsed.items.length);
		step = 'confirm';
		error = '';
	}

	function setLoadingPhase(status: string) {
		loadingPhase = status === 'running' ? 'fns' : 'scan';
	}

	function showScanError(err: unknown) {
		const message = formatApiError(err, 'receipt.parseError');
		error = message;
		if (!open) {
			toast.error(message);
		}
	}

	async function pollScanJob(jobId: string, pollToken: number) {
		if (pollingJobId === jobId) {
			return;
		}
		pollingJobId = jobId;
		loading = true;
		error = '';
		try {
			const parsed = await pollReceiptScanJob(jobId, {
				shouldAbort: () => pollToken !== scanPollingToken,
				onStatus: (current) => {
					if (pollToken !== scanPollingToken) return;
					setLoadingPhase(current.status);
				}
			});
			if (pollToken !== scanPollingToken) return;
			applyParsedReceipt(parsed);
			clearActiveReceiptScanJobId();
		} catch (e) {
			if (pollToken !== scanPollingToken) return;
			if (e instanceof ApiError && e.code === 'ERR_ABORTED') return;
			showScanError(e);
			clearActiveReceiptScanJobId();
		} finally {
			if (pollToken === scanPollingToken) {
				loading = false;
				loadingPhase = null;
				pollingJobId = null;
			}
		}
	}

	async function restoreActiveScanJob(jobId: string) {
		try {
			const job = await getReceiptScanJob(jobId);
			if (job.status === 'queued' || job.status === 'running') {
				loading = true;
				setLoadingPhase(job.status);
				const pollToken = scanPollingToken;
				void pollScanJob(jobId, pollToken);
				return;
			}
			if (job.status === 'done' && job.result) {
				applyParsedReceipt(job.result);
				clearActiveReceiptScanJobId();
				return;
			}
			if (job.status === 'failed') {
				showScanError(
					new ApiError(job.error_message ?? 'ERR_RECEIPT_SCAN', job.error_message ?? 'ERR_RECEIPT_SCAN', 400)
				);
				clearActiveReceiptScanJobId();
			}
		} catch {
			clearActiveReceiptScanJobId();
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
			const job = await createReceiptScanJob(file);
			if (run !== decodeRun || pollToken !== scanPollingToken) return;
			saveActiveReceiptScanJobId(job.id);
			await pollScanJob(job.id, pollToken);
		} catch (e) {
			if (run !== decodeRun || pollToken !== scanPollingToken) return;
			showScanError(e);
			clearActiveReceiptScanJobId();
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
			scannerHandle = await startQrScanner('receipt-qr-reader', async (text) => {
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
			applyParsedReceipt(parsed);
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

	async function loadSubcategories(rowIndex: number, categoryId: string) {
		if (!categoryId) {
			subcategoriesByRow = { ...subcategoriesByRow, [rowIndex]: [] };
			return;
		}
		const subs = await listSubcategories(categoryId);
		subcategoriesByRow = { ...subcategoriesByRow, [rowIndex]: subs };
	}

	function applyBulkCategory() {
		rows = applyCategoryToSelected(rows, bulkCategoryId, bulkSubcategoryId);
	}

	async function save() {
		if (!receipt || missingCategory || !selectedAccount) return;
		saving = true;
		error = '';
		try {
			await importReceipt({
				account_id: selectedAccount,
				fiscal_drive_number: receipt.fiscal_drive_number,
				fiscal_document_number: receipt.fiscal_document_number,
				fiscal_sign: receipt.fiscal_sign,
				receipt,
				items: rows.map((row) => ({
					index: row.index,
					category_id: row.categoryId || undefined,
					subcategory_id: row.subcategoryId || null,
					include: row.include
				}))
			});
			toast.success($_('receipt.importSuccess', { values: { count: includedCount } }));
			clearActiveReceiptScanJobId();
			fullResetForm();
			open = false;
			onclose();
			onsaved();
		} catch (e) {
			error = formatApiError(e, 'receipt.importError');
		} finally {
			saving = false;
		}
	}

	function handleClose() {
		void stopScanner();
		onclose();
	}
</script>

<ModalShell bind:open title={$_('receipt.title')} maxWidth="max-w-3xl" onclose={handleClose}>
	{#if step === 'input'}
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">{$_('receipt.intro')}</p>
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn-primary min-h-11" disabled={busy} onclick={toggleScanner}>
					{scanning ? $_('receipt.stopScan') : $_('receipt.scan')}
				</button>
				<label
					class="btn-ghost min-h-11 cursor-pointer"
					class:opacity-50={busy}
					class:pointer-events-none={busy}
					aria-disabled={busy}
				>
					{$_('receipt.upload')}
					<input
						type="file"
						accept="image/*"
						class="hidden"
						disabled={busy}
						onchange={handleFileChange}
					/>
				</label>
			</div>
			{#if scanning}
				<div id="receipt-qr-reader" class="overflow-hidden rounded-lg border" style:border-color="var(--border)"></div>
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
		<div class="space-y-4">
			<div class="rounded-lg border p-3 text-sm" style:border-color="var(--border)">
				<p class="font-medium">{receipt.store_name || '—'}</p>
				<p style:color="var(--text-muted)">
					{$_('receipt.docNumber')}: {receipt.fiscal_document_number} ·
					{formatMoneyForDisplay(receipt.total_sum / 100)}
				</p>
			</div>

			<Select
				id="receipt-account"
				label={$_('transactions.field.account')}
				bind:value={selectedAccount}
				options={accountOptions}
				usePortal
			/>

			<div class="flex flex-wrap items-end gap-2 rounded-lg border p-3" style:border-color="var(--border)">
				<div class="min-w-[12rem] flex-1">
					<Select
						id="receipt-bulk-category"
						label={$_('receipt.applyCategory')}
						bind:value={bulkCategoryId}
						options={categoryOptions}
						usePortal
					/>
				</div>
				<button type="button" class="btn-primary min-h-11" disabled={!bulkCategoryId} onclick={applyBulkCategory}>
					{$_('receipt.apply')}
				</button>
			</div>

			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr style:color="var(--text-muted)">
							<th class="p-2 text-left"></th>
							<th class="p-2 text-left"></th>
							<th class="p-2 text-left">{$_('receipt.col.item')}</th>
							<th class="p-2 text-right">{$_('receipt.col.sum')}</th>
							<th class="p-2 text-left">{$_('transactions.col.category')}</th>
						</tr>
					</thead>
					<tbody>
						{#each rows as row (row.index)}
							<tr class="border-t" style:border-color="var(--border)">
								<td class="p-2">
									<input type="checkbox" bind:checked={row.include} aria-label={$_('receipt.include')} />
								</td>
								<td class="p-2">
									<input type="checkbox" bind:checked={row.selected} aria-label={$_('receipt.select')} />
								</td>
								<td class="p-2">{receipt.items[row.index]?.name ?? ''}</td>
								<td class="p-2 text-right tabular-nums">
									<MoneyDisplay value={formatReceiptItemSum(receipt.items[row.index]?.sum ?? 0)} />
								</td>
								<td class="p-2">
									<Select
										id={`receipt-cat-${row.index}`}
										bind:value={row.categoryId}
										options={categoryOptions}
										usePortal
										onchange={() => {
											row.subcategoryId = '';
											void loadSubcategories(row.index, row.categoryId);
										}}
									/>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

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
					disabled={saving || missingCategory || includedCount === 0}
					onclick={save}
				>
					{$_('receipt.create', { values: { count: includedCount } })}
				</button>
			</div>
		</div>
	{/if}
</ModalShell>
