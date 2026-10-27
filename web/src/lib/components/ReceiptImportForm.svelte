<script lang="ts">
	import { onDestroy } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		importReceipt,
		listAccounts,
		listCategories,
		listSubcategories,
		parseReceipt,
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
	import ReceiptInlineSubcategoryField from '$lib/components/ReceiptInlineSubcategoryField.svelte';
	import ReceiptItemCategoryButton from '$lib/components/ReceiptItemCategoryButton.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import {
		applyCategoryToUnassigned,
		countIncludedRows,
		formatReceiptItemSum,
		includedRowsMissingCategory,
		initReceiptRows,
		type ReceiptRowState
	} from '$lib/receipt/receipt-items';
	import {
		decodeQrFromImageFile,
		startQrScanner,
		type QrScannerHandle
	} from '$lib/receipt/parse-qr';
	import {
		clearActiveReceiptScanJobId,
		getActiveReceiptScanJobId
	} from '$lib/receipt/scan-job-session';
	import { accountSelectOptions, categorySelectOptions } from '$lib/select-options';
	import { matchRefByName, refsForAccount } from '$lib/household/refs-for-account';
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
	let bulkSubcategories = $state<Subcategory[]>([]);
	let scannerHandle = $state<QrScannerHandle | null>(null);
	let decodeRun = 0;
	let wasOpen = false;

	const busy = $derived(loading || scanning);
	const selectedAcc = $derived(accounts.find((acc) => acc.id === selectedAccount));
	const scopedCategories = $derived(
		refsForAccount(selectedAcc, categories).filter((c) => !c.is_system)
	);
	const categoryOptions = $derived(categorySelectOptions(scopedCategories));
	const accountOptions = $derived(accountSelectOptions(accounts));
	const includedCount = $derived(countIncludedRows(rows));
	const missingCategory = $derived(includedRowsMissingCategory(rows));
	const receiptBlocked = $derived(Boolean(receipt?.already_imported));
	const modalMaxWidth = $derived(receiptBlocked ? 'max-w-lg' : 'max-w-6xl');

	$effect(() => {
		if (!open) {
			if (wasOpen) {
				scanPollingToken++;
				void stopScanner();
				loading = false;
				loadingPhase = null;
				pollingJobId = null;
			}
			wasOpen = false;
			return;
		}
		if (!wasOpen) {
			wasOpen = true;
			void initOnOpen();
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

	async function ensureMetaLoaded() {
		if (accounts.length === 0 || categories.length === 0) {
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
		if (parsed.already_imported) {
			return;
		}
		await ensureMetaLoaded();
		rows = initReceiptRows(parsed.items.length);
		subcategoriesByRow = {};
		bulkCategoryId = '';
		bulkSubcategoryId = '';
		bulkSubcategories = [];
	}

	function remapReceiptCategory(id: string): { id: string; dropped: boolean } {
		const current = categories.find((c) => c.id === id);
		const matched = matchRefByName(current, scopedCategories);
		if (!id) return { id: '', dropped: false };
		if (current && !matched) return { id: '', dropped: true };
		return { id: matched?.id ?? '', dropped: false };
	}

	function onReceiptAccountChange() {
		const bulk = remapReceiptCategory(bulkCategoryId);
		if (bulk.dropped) {
			bulkCategoryId = '';
			bulkSubcategoryId = '';
			bulkSubcategories = [];
			toast($_('refs.reset.category'));
		} else if (bulk.id !== bulkCategoryId) {
			bulkCategoryId = bulk.id;
		}
		let droppedRow = false;
		rows = rows.map((row) => {
			const next = remapReceiptCategory(row.categoryId);
			if (next.dropped) droppedRow = true;
			if (next.id === row.categoryId) return row;
			return { ...row, categoryId: next.id, subcategoryId: '' };
		});
		if (droppedRow) toast($_('refs.reset.category'));
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
			await applyParsedReceipt(parsed);
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
			}
			if (pollingJobId === jobId) {
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
				await applyParsedReceipt(job.result);
				clearActiveReceiptScanJobId();
				return;
			}
			if (job.status === 'failed') {
				showScanError(
					new ApiError(
						job.error_message ?? 'ERR_RECEIPT_SCAN',
						job.error_message ?? 'ERR_RECEIPT_SCAN',
						400
					)
				);
				clearActiveReceiptScanJobId();
			}
		} catch {
			clearActiveReceiptScanJobId();
			loading = false;
			loadingPhase = null;
			pollingJobId = null;
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

<ModalShell bind:open title={$_('receipt.title')} maxWidth={modalMaxWidth} onclose={handleClose}>
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
					id="receipt-qr-reader"
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
				<p class="text-sm text-amber-700" role="alert">{$_('receipt.alreadyImported')}</p>
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
				</div>

				<Select
					id="receipt-account"
					label={$_('transactions.field.account')}
					bind:value={selectedAccount}
					options={accountOptions}
					onchange={() => onReceiptAccountChange()}
					usePortal
				/>

				<div class="space-y-2 rounded-lg border p-3" style:border-color="var(--border)">
					<p class="text-sm font-medium">{$_('receipt.bulkCategory')}</p>
					<p class="text-xs" style:color="var(--text-muted)">{$_('receipt.bulkCategoryHint')}</p>
					<div class="flex flex-wrap items-end gap-2">
						<div class="min-w-[12rem] flex-1">
							<Select
								id="receipt-bulk-category"
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
								id="receipt-bulk-subcategory"
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
							<th class="w-[12%] p-2 text-left"></th>
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
								<td class="p-2">
									<input
										type="checkbox"
										bind:checked={row.include}
										aria-label={$_('receipt.include')}
									/>
								</td>
								<td class="min-w-0 p-2 break-words">{receipt.items[row.index]?.name ?? ''}</td>
								<td class="p-2 text-right tabular-nums">
									<MoneyDisplay value={formatReceiptItemSum(receipt.items[row.index]?.sum ?? 0)} />
								</td>
								<td class="min-w-0 p-2">
									<ReceiptItemCategoryButton
										buttonId={`receipt-item-cat-${row.index}`}
										categorySelectId={`receipt-cat-${row.index}`}
										subcategoryFieldId={`receipt-subcat-${row.index}`}
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
						disabled={saving || missingCategory || includedCount === 0}
						onclick={save}
					>
						{$_('common.save')}
					</button>
				</div>
			</div>
		{/if}
	{/if}
</ModalShell>
