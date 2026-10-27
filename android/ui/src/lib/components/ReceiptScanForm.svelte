<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		attachReceipt,
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
		type Subcategory,
		type Transaction
	} from '$lib/api/client';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
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
	import { ReceiptPhotoError } from '$lib/receipt/compress-photo';
	import { receiptPhotoSendErrorKey } from '$lib/receipt/photo-send-error';
	import { isServerUnavailableForUser } from '$lib/offline/server-connectivity';
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
	import { confirm } from '$lib/confirm';

	type Props = {
		mode: 'import' | 'attach';
		backHref: string;
		onclose: () => void;
		onsaved: () => void;
		accountId?: string;
		transaction?: Transaction | null;
	};

	let { mode, backHref, onclose, onsaved, accountId = '', transaction = null }: Props = $props();

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

	const busy = $derived(loading || scanning);
	const txAmount = $derived(transaction?.amount ?? 0);
	const selectedAcc = $derived(accounts.find((acc) => acc.id === selectedAccount));
	const catalogAccount = $derived(mode === 'attach' ? transaction : selectedAcc);
	const scopedCategories = $derived(
		refsForAccount(catalogAccount, categories).filter((c) => !c.is_system)
	);
	const categoryOptions = $derived(categorySelectOptions(scopedCategories));
	const accountOptions = $derived(accountSelectOptions(accounts));
	const includedCount = $derived(countIncludedRows(rows));
	const missingCategory = $derived(includedRowsMissingCategory(rows));
	const receiptBlocked = $derived(
		Boolean(receipt && (mode === 'import' ? receipt.already_imported : receipt.already_in_ledger))
	);
	const pageTitle = $derived(mode === 'import' ? $_('receipt.title') : $_('receipt.attach.title'));
	const qrReaderId = $derived(mode === 'import' ? 'receipt-qr-reader' : 'receipt-attach-qr-reader');

	onMount(() => {
		void initOnOpen();
	});

	onDestroy(() => {
		scanPollingToken++;
		void stopScanner();
	});

	async function initOnOpen() {
		await loadMeta();
		if (mode === 'import') {
			const storedJobId = getActiveReceiptScanJobId();
			if (storedJobId) {
				await restoreActiveScanJob(storedJobId);
			}
		}
	}

	async function loadMeta() {
		if (mode === 'import') {
			const [accs, cats] = await Promise.all([listAccounts('active'), listCategories('expense')]);
			accounts = accs;
			categories = cats;
			selectedAccount = defaultAccountId(accs, accountId);
		} else {
			categories = await listCategories('expense');
		}
	}

	async function ensureMetaLoaded() {
		if (mode === 'import' && (accounts.length === 0 || categories.length === 0)) {
			await loadMeta();
		}
		if (mode === 'attach' && categories.length === 0) {
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
		if (mode === 'import' && parsed.already_imported) return;
		if (mode === 'attach' && parsed.already_in_ledger) return;
		await ensureMetaLoaded();
		rows = initReceiptRows(parsed.items.length);
		subcategoriesByRow = {};
		bulkCategoryId = '';
		bulkSubcategoryId = '';
		bulkSubcategories = [];
		if (mode === 'attach' && transaction) {
			const txCat = transaction.category_id ?? '';
			const txSub = transaction.subcategory_id ?? '';
			if (txCat) {
				rows = rows.map((row) => ({
					...row,
					categoryId: txCat,
					subcategoryId: txSub
				}));
				const stubName = transaction.subcategory_name?.trim() ?? '';
				if (txSub && stubName) {
					subcategoriesByRow = Object.fromEntries(
						rows.map((row) => [
							row.index,
							[
								{
									id: txSub,
									category_id: txCat,
									name: stubName,
									icon: transaction.subcategory_icon || 'default',
									sort_order: 0,
									created_at: ''
								}
							]
						])
					);
				}
				for (const row of rows) {
					void loadSubcategories(row.index, txCat);
				}
			}
		}
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
		if (err instanceof ReceiptPhotoError) {
			error = $_('receipt.photoTooLarge');
			return;
		}
		if (isServerUnavailableForUser(err)) {
			error = $_('receipt.requestNotSent');
			return;
		}
		error = formatApiError(err, 'receipt.parseError');
	}

	/** Upload/poll failed on the device. Do not reuse «session saved». */
	function showPhotoSendError(err: unknown) {
		const key = receiptPhotoSendErrorKey(err);
		if (key === 'receipt.photoTooLarge') {
			error = $_('receipt.photoTooLarge');
			return;
		}
		if (key === 'receipt.photoNotSent') {
			error = $_('receipt.photoNotSent');
			return;
		}
		showScanError(err);
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
			if (mode === 'import') clearActiveReceiptScanJobId();
		} catch (e) {
			if (pollToken !== scanPollingToken) return;
			if (e instanceof ApiError && e.code === 'ERR_ABORTED') return;
			showPhotoSendError(e);
			if (mode === 'import') clearActiveReceiptScanJobId();
		} finally {
			if (pollToken === scanPollingToken) {
				loading = false;
				loadingPhase = null;
			}
			if (pollingJobId === jobId) pollingJobId = null;
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
					// Gallery return resumes the activity in the same turn and the first POST
					// dies on the bridge. Wait, then retry once.
					await new Promise((resolve) => setTimeout(resolve, 350));
					if (run !== decodeRun || pollToken !== scanPollingToken) return;
					let parsed;
					try {
						parsed = await parseReceipt(qr);
					} catch (e) {
						if (!isServerUnavailableForUser(e)) throw e;
						await new Promise((resolve) => setTimeout(resolve, 400));
						if (run !== decodeRun || pollToken !== scanPollingToken) return;
						parsed = await parseReceipt(qr);
					}
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
			showPhotoSendError(e);
			if (mode === 'import') clearActiveReceiptScanJobId();
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
			scannerHandle = await startQrScanner(qrReaderId, async (text) => {
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

	function attachItemsPayload() {
		return rows.map((row) => ({
			index: row.index,
			category_id: row.categoryId || undefined,
			subcategory_id: row.subcategoryId || null,
			include: mode === 'attach' ? true : row.include
		}));
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

	async function saveImport() {
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
				items: attachItemsPayload()
			});
			toast.success($_('receipt.importSuccess', { values: { count: includedCount } }));
			clearActiveReceiptScanJobId();
			onsaved();
		} catch (e) {
			error = formatApiError(e, 'receipt.importError');
		} finally {
			saving = false;
		}
	}

	async function saveAttach() {
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
						transaction_id: transaction!.id,
						fiscal_drive_number: receipt.fiscal_drive_number,
						fiscal_document_number: receipt.fiscal_document_number,
						fiscal_sign: receipt.fiscal_sign,
						receipt,
						items: attachItemsPayload(),
						acknowledge_sum_mismatch: true
					});
					toast.success($_('receipt.attach.success'));
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

	function save() {
		if (mode === 'import') void saveImport();
		else void saveAttach();
	}

	function goBackStep() {
		if (step === 'confirm') {
			step = 'input';
			receipt = null;
			rows = [];
			error = '';
			return;
		}
		scanPollingToken++;
		void stopScanner();
		onclose();
	}

	function handleClose() {
		scanPollingToken++;
		void stopScanner();
		onclose();
	}
</script>

{#snippet confirmFooter()}
	<button type="button" class="btn-ghost" onclick={goBackStep}>
		{$_('common.back')}
	</button>
	<button
		type="button"
		class="btn-primary"
		disabled={saving ||
			missingCategory ||
			(mode === 'import' && includedCount === 0) ||
			(mode === 'attach' && !transaction)}
		onclick={save}
	>
		{mode === 'import' ? $_('common.save') : $_('receipt.attach.confirm')}
	</button>
{/snippet}

<FormPageShell
	title={pageTitle}
	{backHref}
	onback={handleClose}
	footer={step === 'confirm' && receipt && !receiptBlocked ? confirmFooter : undefined}
>
	<div class="space-y-4 p-4">
		{#if step === 'input'}
			<p class="text-sm" style:color="var(--text-muted)">{$_('receipt.intro')}</p>
			<div class="flex flex-wrap items-center justify-center gap-2">
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
					id={qrReaderId}
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
		{:else if receipt}
			{#if receiptBlocked}
				<p class="text-sm text-amber-700" role="alert">
					{mode === 'import' ? $_('receipt.alreadyImported') : $_('receipt.alreadyInLedger')}
				</p>
				<button type="button" class="btn-ghost min-h-11" onclick={goBackStep}>
					{$_('common.back')}
				</button>
			{:else}
				<div class="rounded-lg border p-3 text-sm" style:border-color="var(--border)">
					<p class="font-medium">{receipt.store_name || '—'}</p>
					<p style:color="var(--text-muted)">
						{$_('receipt.docNumber')}: {receipt.fiscal_document_number} ·
						{formatMoneyForDisplay({ cents: receipt.total_sum })}
					</p>
					{#if mode === 'attach' && transaction && txAmount !== receipt.total_sum}
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

				{#if mode === 'import'}
					<Select
						id="receipt-account"
						label={$_('transactions.field.account')}
						bind:value={selectedAccount}
						options={accountOptions}
						onchange={() => onReceiptAccountChange()}
					/>
				{/if}

				<details class="rounded-lg border p-3" style:border-color="var(--border)">
					<summary class="cursor-pointer text-sm font-medium">
						{$_('receipt.bulkCategory')}
					</summary>
					<div class="mt-2 space-y-2">
						<p class="text-xs" style:color="var(--text-muted)">{$_('receipt.bulkCategoryHint')}</p>
						<Select
							id="receipt-bulk-category"
							label={$_('transactions.field.category')}
							placeholder={$_('receipt.selectCategory')}
							bind:value={bulkCategoryId}
							options={categoryOptions}
							onchange={() => {
								bulkSubcategoryId = '';
								void loadBulkSubcategories(bulkCategoryId);
							}}
						/>
						<span class="block text-sm font-medium">{$_('transactions.field.subcategory')}</span>
						<ReceiptInlineSubcategoryField
							id="receipt-bulk-subcategory"
							categoryId={bulkCategoryId}
							bind:value={bulkSubcategoryId}
							subcategories={bulkSubcategories}
							onSubcategoriesChange={(subs) => {
								bulkSubcategories = subs;
							}}
						/>
						<button
							type="button"
							class="btn-primary min-h-11 w-full"
							disabled={!bulkCategoryId}
							onclick={applyBulkToUnassigned}
						>
							{$_('receipt.applyUnassigned')}
						</button>
					</div>
				</details>

				<table class="w-full table-fixed text-sm">
					<thead>
						<tr style:color="var(--text-muted)">
							{#if mode === 'import'}
								<th class="w-[12%] p-2 text-left"></th>
							{/if}
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
								{#if mode === 'import'}
									<td class="p-2">
										<input
											type="checkbox"
											bind:checked={row.include}
											aria-label={$_('receipt.include')}
										/>
									</td>
								{/if}
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
			{/if}
		{/if}
	</div>
</FormPageShell>
