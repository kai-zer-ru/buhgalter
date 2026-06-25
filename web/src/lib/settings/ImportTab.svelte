<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		ApiError,
		createImportJob,
		exportCSVUrl,
		getImportJob,
		listAccounts,
		listBanks,
		listCategories,
		listSubcategories,
		peekImportHeaders,
		previewImport,
		type Account,
		type AccountMapEntry,
		type AccountMappingSuggestion,
		type Bank,
		type Category,
		type CategoryMapEntry,
		type CategoryMappingSuggestion,
		type ImportJob,
		type Subcategory,
		type SubcategoryMapEntry,
		type SubcategoryMappingSuggestion,
		type ImportReport
	} from '$lib/api/client';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';
	import {
		IMPORT_COLUMN_FIELDS,
		accountTypeLabel,
		guessColumnMap,
		isColumnMapValid,
		type ImportColumnField
	} from '$lib/import-columns';
	import { tick } from 'svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';

	type Step =
		| 'upload'
		| 'settings'
		| 'mapping'
		| 'accounts'
		| 'categories'
		| 'subcategories'
		| 'preview'
		| 'importing'
		| 'done';
	type Tab = 'import' | 'export';
	const ACTIVE_IMPORT_JOB_ID_KEY = 'buhgalter.active_import_job_id';

	let tab = $state<Tab>('import');
	let step = $state<Step>('upload');
	let file = $state<File | null>(null);
	let preset = $state<'cubux' | 'custom'>('cubux');
	let deduplicate = $state(true);
	let dragOver = $state(false);
	let loading = $state(false);
	let error = $state('');
	let report = $state<ImportReport | null>(null);
	let finalReport = $state<ImportReport | null>(null);
	let importJob = $state<ImportJob | null>(null);
	let importPollingToken = 0;
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let categories = $state<Category[]>([]);
	let subcategoriesByCategory = $state<Record<string, Subcategory[]>>({});
	let accountMap = $state<Record<string, AccountMapEntry>>({});
	let categoryMap = $state<Record<string, CategoryMapEntry>>({});
	let subcategoryMap = $state<Record<string, SubcategoryMapEntry>>({});
	let autoSubcategory = $state(true);
	let importAttemptKey = $state('');
	let fileHeaders = $state<string[]>([]);
	let columnMap = $state<Record<string, string>>({});

	let exportFrom = $state('2025-01-01');
	let exportTo = $state(new Date().toISOString().slice(0, 10));
	let exportAccountId = $state('');
	let exportCategoryId = $state('');

	onMount(async () => {
		try {
			[accounts, banks] = await Promise.all([listAccounts(), listBanks()]);
			const [expense, income] = await Promise.all([
				listCategories('expense'),
				listCategories('income')
			]);
			categories = [...expense, ...income].sort((a, b) => a.name.localeCompare(b.name, 'ru'));
		} catch {
			// optional for filters
		}
		void restoreActiveImportJob();
	});

	function saveActiveImportJobID(jobID: string) {
		localStorage.setItem(ACTIVE_IMPORT_JOB_ID_KEY, jobID);
	}

	function clearActiveImportJobID() {
		localStorage.removeItem(ACTIVE_IMPORT_JOB_ID_KEY);
	}

	function emptyReport(): ImportReport {
		return {
			total_rows: 0,
			processed_rows: 0,
			valid_rows: 0,
			skipped_duplicates: 0,
			errors: [],
			logs: [],
			preview: [],
			accounts_to_create: [],
			account_mappings: [],
			category_mappings: [],
			subcategory_mappings: [],
			categories_to_create: []
		};
	}

	async function restoreActiveImportJob() {
		const jobID = localStorage.getItem(ACTIVE_IMPORT_JOB_ID_KEY);
		if (!jobID) return;
		try {
			const current = await getImportJob(jobID);
			importJob = current;
			if (current.status === 'queued' || current.status === 'running') {
				step = 'importing';
				void pollImportJob(jobID);
				return;
			}
			if (current.status === 'done') {
				finalReport = normalizeImportReport(current.report ?? emptyReport());
				step = 'done';
			}
			if (current.status === 'failed') {
				error = current.error_message ?? $_('common.error');
			}
		} catch {
			// stale or unavailable job id, ignore and reset persisted state
		} finally {
			clearActiveImportJobID();
		}
	}

	function onFileSelect(f: File | null) {
		if (!f) return;
		const lower = f.name.toLowerCase();
		if (!lower.endsWith('.csv') && !lower.endsWith('.xlsx')) {
			error = $_('import.error.format');
			return;
		}
		file = f;
		error = '';
		fileHeaders = [];
		columnMap = {};
		subcategoryMap = {};
		autoSubcategory = true;
		step = 'settings';
	}

	function onDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const f = e.dataTransfer?.files?.[0];
		if (f) onFileSelect(f);
	}

	function importOpts() {
		return {
			file: file!,
			preset,
			deduplicate,
			column_map: preset === 'custom' ? columnMap : undefined,
			account_map: accountMap,
			category_map: categoryMap,
			subcategory_map: subcategoryMap,
			auto_subcategory: autoSubcategory
		};
	}

	async function goFromSettings() {
		if (!file) return;
		error = '';
		if (preset === 'custom') {
			loading = true;
			try {
				const res = await peekImportHeaders(file);
				fileHeaders = res.headers;
				columnMap = guessColumnMap(fileHeaders);
				step = 'mapping';
			} catch (err) {
				error = err instanceof ApiError ? err.message : $_('common.error');
			} finally {
				loading = false;
			}
			return;
		}
		await runPreview();
	}

	function goFromMapping() {
		if (!isColumnMapValid(columnMap)) {
			error = $_('import.mapping.required');
			return;
		}
		error = '';
		void runPreview();
	}

	function setColumnField(field: ImportColumnField, header: string) {
		if (!header) {
			const next = { ...columnMap };
			delete next[field];
			columnMap = next;
			return;
		}
		columnMap = { ...columnMap, [field]: header };
	}

	function normalizeImportReport(r: ImportReport): ImportReport {
		return {
			...r,
			processed_rows: r.processed_rows ?? 0,
			errors: r.errors ?? [],
			logs: r.logs ?? [],
			preview: r.preview ?? [],
			accounts_to_create: r.accounts_to_create ?? [],
			account_mappings: r.account_mappings ?? [],
			category_mappings: r.category_mappings ?? [],
			subcategory_mappings: r.subcategory_mappings ?? [],
			categories_to_create: r.categories_to_create ?? []
		};
	}

	function wizardStepAfterPreview(raw: ImportReport): Step {
		if (raw.account_mappings.length > 0) return 'accounts';
		if (raw.category_mappings.length > 0) return 'categories';
		if (!autoSubcategory && raw.subcategory_mappings.length > 0) return 'subcategories';
		return 'preview';
	}

	function newImportAttemptKey() {
		importAttemptKey = crypto.randomUUID();
	}

	async function runPreview() {
		if (!file) return;
		loading = true;
		error = '';
		try {
			const raw = normalizeImportReport(await previewImport(importOpts()));
			initAccountMapFromReport(raw.account_mappings);
			initCategoryMapFromReport(raw.category_mappings);
			initSubcategoryMapFromReport(raw.subcategory_mappings);
			loading = false;
			await tick();
			report = raw;
			step = wizardStepAfterPreview(raw);
			if (step === 'preview') {
				newImportAttemptKey();
			}
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
			loading = false;
		}
	}

	function initCategoryMapFromReport(mappings: CategoryMappingSuggestion[]) {
		const next: Record<string, CategoryMapEntry> = {};
		for (const m of mappings) {
			next[categoryMapKey(m)] = {
				mode: m.mode,
				category_id: m.category_id
			};
		}
		categoryMap = next;
	}

	function categoryMapKey(m: CategoryMappingSuggestion): string {
		return `${m.file_name}|${m.type}`;
	}

	function categoriesForType(type: 'expense' | 'income'): Category[] {
		return categories.filter((c) => c.type === type);
	}

	function categoryTypeLabel(type: 'expense' | 'income'): string {
		return type === 'income' ? $_('categories.tab.income') : $_('categories.tab.expense');
	}

	function initSubcategoryMapFromReport(mappings: SubcategoryMappingSuggestion[]) {
		const next: Record<string, SubcategoryMapEntry> = {};
		for (const m of mappings) {
			next[subcategoryMapKey(m)] = {
				mode: m.mode,
				subcategory_id: m.subcategory_id
			};
		}
		subcategoryMap = next;
	}

	function subcategoryMapKey(m: SubcategoryMappingSuggestion): string {
		return `${m.file_category}|${m.type}|${m.file_subcategory}`;
	}

	function categoryKeyByName(fileCategory: string, type: 'expense' | 'income'): string {
		return `${fileCategory}|${type}`;
	}

	function categoryMapEntryByName(
		fileCategory: string,
		type: 'expense' | 'income'
	): CategoryMapEntry {
		const key = categoryKeyByName(fileCategory, type);
		const mapped = categoryMap[key];
		if (mapped) return mapped;
		const suggested = report?.category_mappings?.find(
			(m) => m.file_name === fileCategory && m.type === type
		);
		return {
			mode: (suggested?.mode ?? 'create') as 'create' | 'existing',
			category_id: suggested?.category_id ?? ''
		};
	}

	async function ensureSubcategoriesLoaded(categoryId: string) {
		if (!categoryId || subcategoriesByCategory[categoryId]) return;
		try {
			const subs = await listSubcategories(categoryId);
			subcategoriesByCategory = { ...subcategoriesByCategory, [categoryId]: subs };
		} catch {
			// ignore; validation will catch missing selection if needed
		}
	}

	async function preloadSubcategoriesForMappings(mappings: SubcategoryMappingSuggestion[]) {
		const categoryIdSet: Record<string, true> = {};
		for (const m of mappings) {
			const categoryEntry = categoryMapEntryByName(m.file_category, m.type);
			if (categoryEntry.mode === 'existing' && categoryEntry.category_id) {
				categoryIdSet[categoryEntry.category_id] = true;
			}
		}
		await Promise.all(Object.keys(categoryIdSet).map((id) => ensureSubcategoriesLoaded(id)));
	}

	function initAccountMapFromReport(mappings: AccountMappingSuggestion[]) {
		const next: Record<string, AccountMapEntry> = {};
		for (const m of mappings) {
			next[m.file_name] = {
				mode: m.mode,
				account_id: m.account_id,
				account_type: m.account_type ?? 'cash',
				bank_id: m.bank_id
			};
		}
		accountMap = next;
	}

	function setAccountMapMode(fileName: string, mode: 'create' | 'existing') {
		const prev = accountMap[fileName];
		const suggestion = report?.account_mappings?.find((m) => m.file_name === fileName);
		if (mode === 'create') {
			accountMap = {
				...accountMap,
				[fileName]: {
					mode: 'create',
					account_type: prev?.account_type ?? suggestion?.account_type ?? 'cash',
					bank_id: prev?.bank_id ?? suggestion?.bank_id
				}
			};
			return;
		}
		const fallbackId = prev?.account_id ?? suggestion?.account_id ?? accounts[0]?.id ?? '';
		accountMap = { ...accountMap, [fileName]: { mode: 'existing', account_id: fallbackId } };
	}

	function setAccountMapExisting(fileName: string, accountId: string) {
		accountMap = { ...accountMap, [fileName]: { mode: 'existing', account_id: accountId } };
	}

	function setAccountMapType(fileName: string, accountType: 'cash' | 'bank') {
		const prev = accountMap[fileName];
		const suggestion = report?.account_mappings?.find((m) => m.file_name === fileName);
		accountMap = {
			...accountMap,
			[fileName]: {
				mode: 'create',
				account_type: accountType,
				bank_id:
					accountType === 'bank'
						? (prev?.bank_id ?? suggestion?.bank_id ?? banks[0]?.id ?? '')
						: undefined
			}
		};
	}

	function setAccountMapBank(fileName: string, bankId: string) {
		accountMap = {
			...accountMap,
			[fileName]: {
				mode: 'create',
				account_type: 'bank',
				bank_id: bankId
			}
		};
	}

	function mapEntry(m: AccountMappingSuggestion): AccountMapEntry {
		const entry = accountMap[m.file_name];
		return {
			mode: (entry?.mode ?? m.mode) as 'create' | 'existing',
			account_id: entry?.account_id ?? m.account_id ?? '',
			account_type: entry?.account_type ?? m.account_type ?? 'cash',
			bank_id: entry?.bank_id ?? m.bank_id ?? ''
		};
	}

	function syncAccountMapFromUI() {
		const next: Record<string, AccountMapEntry> = {};
		for (const m of report?.account_mappings ?? []) {
			const entry = mapEntry(m);
			next[m.file_name] =
				entry.mode === 'create'
					? {
							mode: 'create',
							account_type: entry.account_type ?? 'cash',
							bank_id: entry.account_type === 'bank' ? entry.bank_id : undefined
						}
					: { mode: 'existing', account_id: entry.account_id };
		}
		accountMap = next;
	}

	function validateAccountMap(): boolean {
		for (const m of report?.account_mappings ?? []) {
			const entry = mapEntry(m);
			if (entry.mode === 'existing' && !entry.account_id) {
				error = $_('import.accounts.pick_existing', { values: { name: m.file_name } });
				return false;
			}
			if (entry.mode === 'create' && entry.account_type === 'bank' && !entry.bank_id) {
				error = $_('import.accounts.pick_bank', { values: { name: m.file_name } });
				return false;
			}
		}
		return true;
	}

	function goFromAccounts() {
		error = '';
		syncAccountMapFromUI();
		if (!validateAccountMap()) return;
		step = (report?.category_mappings?.length ?? 0) > 0 ? 'categories' : 'preview';
		if (step === 'preview') {
			newImportAttemptKey();
		}
	}

	function setCategoryMapMode(
		key: string,
		m: CategoryMappingSuggestion,
		mode: 'create' | 'existing'
	) {
		const prev = categoryMap[key];
		if (mode === 'create') {
			categoryMap = { ...categoryMap, [key]: { mode: 'create' } };
			return;
		}
		const fallbackId = prev?.category_id ?? m.category_id ?? categoriesForType(m.type)[0]?.id ?? '';
		categoryMap = { ...categoryMap, [key]: { mode: 'existing', category_id: fallbackId } };
	}

	function setCategoryMapExisting(key: string, categoryId: string) {
		categoryMap = { ...categoryMap, [key]: { mode: 'existing', category_id: categoryId } };
	}

	function categoryMapEntry(m: CategoryMappingSuggestion): CategoryMapEntry {
		const entry = categoryMap[categoryMapKey(m)];
		return {
			mode: (entry?.mode ?? m.mode) as 'create' | 'existing',
			category_id: entry?.category_id ?? m.category_id ?? ''
		};
	}

	function syncCategoryMapFromUI() {
		const next: Record<string, CategoryMapEntry> = {};
		for (const m of report?.category_mappings ?? []) {
			const entry = categoryMapEntry(m);
			const key = categoryMapKey(m);
			next[key] =
				entry.mode === 'create'
					? { mode: 'create' }
					: { mode: 'existing', category_id: entry.category_id };
		}
		categoryMap = next;
	}

	function validateCategoryMap(): boolean {
		for (const m of report?.category_mappings ?? []) {
			const entry = categoryMapEntry(m);
			if (entry.mode === 'existing' && !entry.category_id) {
				error = $_('import.categories.pick_existing', { values: { name: m.file_name } });
				return false;
			}
		}
		return true;
	}

	function subcategoryMapEntry(m: SubcategoryMappingSuggestion): SubcategoryMapEntry {
		const entry = subcategoryMap[subcategoryMapKey(m)];
		return {
			mode: (entry?.mode ?? m.mode) as 'create' | 'existing',
			subcategory_id: entry?.subcategory_id ?? m.subcategory_id ?? ''
		};
	}

	function subcategoryOptionsFor(m: SubcategoryMappingSuggestion): Subcategory[] {
		const catEntry = categoryMapEntryByName(m.file_category, m.type);
		if (catEntry.mode !== 'existing' || !catEntry.category_id) return [];
		return subcategoriesByCategory[catEntry.category_id] ?? [];
	}

	function setSubcategoryMapMode(
		key: string,
		m: SubcategoryMappingSuggestion,
		mode: 'create' | 'existing'
	) {
		if (mode === 'create') {
			subcategoryMap = { ...subcategoryMap, [key]: { mode: 'create' } };
			return;
		}
		const options = subcategoryOptionsFor(m);
		const fallbackId =
			subcategoryMap[key]?.subcategory_id ?? m.subcategory_id ?? options[0]?.id ?? '';
		subcategoryMap = { ...subcategoryMap, [key]: { mode: 'existing', subcategory_id: fallbackId } };
	}

	function setSubcategoryMapExisting(key: string, subcategoryId: string) {
		subcategoryMap = {
			...subcategoryMap,
			[key]: { mode: 'existing', subcategory_id: subcategoryId }
		};
	}

	function syncSubcategoryMapFromUI() {
		const next: Record<string, SubcategoryMapEntry> = {};
		for (const m of report?.subcategory_mappings ?? []) {
			const key = subcategoryMapKey(m);
			const entry = subcategoryMapEntry(m);
			next[key] =
				entry.mode === 'create'
					? { mode: 'create' }
					: { mode: 'existing', subcategory_id: entry.subcategory_id };
		}
		subcategoryMap = next;
	}

	function validateSubcategoryMap(): boolean {
		for (const m of report?.subcategory_mappings ?? []) {
			const catEntry = categoryMapEntryByName(m.file_category, m.type);
			if (catEntry.mode !== 'existing') continue;
			const entry = subcategoryMapEntry(m);
			if (entry.mode === 'existing' && !entry.subcategory_id) {
				error = $_('import.subcategories.pick_existing', { values: { name: m.file_subcategory } });
				return false;
			}
		}
		return true;
	}

	async function goFromCategories() {
		error = '';
		syncCategoryMapFromUI();
		if (!validateCategoryMap()) return;
		if (autoSubcategory) {
			newImportAttemptKey();
			step = 'preview';
			return;
		}
		if (!file) return;
		loading = true;
		try {
			const raw = normalizeImportReport(await previewImport(importOpts()));
			report = raw;
			initSubcategoryMapFromReport(raw.subcategory_mappings);
			await preloadSubcategoriesForMappings(raw.subcategory_mappings);
			newImportAttemptKey();
			step = raw.subcategory_mappings.length > 0 ? 'subcategories' : 'preview';
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}

	function goFromSubcategories() {
		error = '';
		syncSubcategoryMapFromUI();
		if (!validateSubcategoryMap()) return;
		newImportAttemptKey();
		step = 'preview';
	}

	function sleep(ms: number) {
		return new Promise<void>((resolve) => setTimeout(resolve, ms));
	}

	async function pollImportJob(jobId: string) {
		const token = ++importPollingToken;
		while (token === importPollingToken) {
			try {
				const current = await getImportJob(jobId);
				importJob = current;
				if (current.status === 'done') {
					finalReport = normalizeImportReport(current.report ?? emptyReport());
					clearActiveImportJobID();
					step = 'done';
					toast($_('import.done.title'));
					return;
				}
				if (current.status === 'failed') {
					clearActiveImportJobID();
					error = current.error_message ?? $_('common.error');
					step = 'preview';
					return;
				}
			} catch (err) {
				clearActiveImportJobID();
				error = err instanceof ApiError ? err.message : $_('common.error');
				step = 'preview';
				return;
			}
			await sleep(1200);
		}
	}

	async function runImport() {
		if (!file) return;
		const ok = await confirm({
			title: $_('import.confirm.title'),
			message: $_('import.confirm.message', {
				values: { count: report?.valid_rows ?? 0 }
			}),
			confirmLabel: $_('import.confirm.apply')
		});
		if (!ok) return;

		loading = true;
		error = '';
		try {
			if (!importAttemptKey) {
				newImportAttemptKey();
			}
			importJob = await createImportJob({
				...importOpts(),
				idempotencyKey: importAttemptKey
			});
			saveActiveImportJobID(importJob.id);
			step = 'importing';
			loading = false;
			void pollImportJob(importJob.id);
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
			loading = false;
		}
	}

	function reset() {
		file = null;
		step = 'upload';
		report = null;
		finalReport = null;
		importJob = null;
		importPollingToken++;
		clearActiveImportJobID();
		fileHeaders = [];
		columnMap = {};
		accountMap = {};
		categoryMap = {};
		subcategoryMap = {};
		subcategoriesByCategory = {};
		autoSubcategory = true;
		importAttemptKey = '';
		error = '';
	}

	function accountOptionLabel(acc: Account): string {
		return `${acc.name} (${accountTypeLabel(acc.type, $_)})`;
	}

	function progressPercent(total: number, processed: number): number {
		if (total <= 0) return 0;
		const percent = Math.floor((Math.min(processed, total) / total) * 100);
		if (processed > 0) return Math.max(1, percent);
		return percent;
	}

	function etaSeconds(
		total: number,
		processed: number,
		startedAt: string | undefined
	): number | null {
		if (!startedAt || total <= 0 || processed <= 0 || processed >= total) return null;
		const started = Date.parse(startedAt);
		if (!Number.isFinite(started)) return null;
		const elapsedSec = Math.max(1, Math.floor((Date.now() - started) / 1000));
		const rate = processed / elapsedSec;
		if (!Number.isFinite(rate) || rate <= 0) return null;
		return Math.max(1, Math.ceil((total - processed) / rate));
	}

	function formatEta(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		if (mins < 60) return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
		const hours = Math.floor(mins / 60);
		const remMins = mins % 60;
		return remMins > 0 ? `${hours}h ${remMins}m` : `${hours}h`;
	}

	const exportAccountOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.export.all_accounts') },
			...accounts.map((acc) => ({ value: acc.id, label: accountOptionLabel(acc) }))
		];
	});
	const exportCategoryOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.export.all_categories') },
			...categories.map((cat) => ({
				value: cat.id,
				label: `${cat.name} (${cat.type === 'income' ? tr('categories.tab.income') : tr('categories.tab.expense')})`
			}))
		];
	});
	const headerOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.mapping.none') },
			...fileHeaders.map((header) => ({ value: header, label: header }))
		];
	});
	const accountModeOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'create', label: tr('import.accounts.create') },
			{ value: 'existing', label: tr('import.accounts.map_existing') }
		];
	});
	const bankSelectOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('import.accounts.pick_bank_short') },
			...banks.map((bank) => ({ value: bank.id, label: bank.name }))
		];
	});
	const categoryModeOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'create', label: tr('import.accounts.create') },
			{ value: 'existing', label: tr('import.accounts.map_existing') }
		];
	});
	const subcategoryModeOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'create', label: tr('import.accounts.create') },
			{ value: 'existing', label: tr('import.accounts.map_existing') }
		];
	});
</script>

<div class="space-y-4">
	<PageTabs
		active={tab}
		tabs={[
			{ id: 'import', label: $_('import.tab.import') },
			{ id: 'export', label: $_('import.tab.export') }
		]}
		onchange={(next) => (tab = next as Tab)}
	/>

	{#if tab === 'export'}
		<div class="card space-y-4">
			<h2 class="text-lg font-medium">{$_('import.export.title')}</h2>
			<div class="grid gap-4 sm:grid-cols-2">
				<DateTimePicker
					label={$_('import.export.from')}
					bind:value={exportFrom}
					timeMode="hidden"
				/>
				<DateTimePicker label={$_('import.export.to')} bind:value={exportTo} timeMode="hidden" />
			</div>
			<Select
				label={$_('import.export.account')}
				bind:value={exportAccountId}
				options={exportAccountOptions}
			/>
			<Select
				label={$_('import.export.category')}
				bind:value={exportCategoryId}
				options={exportCategoryOptions}
			/>
			<button
				type="button"
				class="btn-primary inline-flex"
				onclick={() => {
					window.location.href = exportCSVUrl({
						from: exportFrom.split('T')[0],
						to: exportTo.split('T')[0],
						account_id: exportAccountId || undefined,
						category_id: exportCategoryId || undefined
					});
				}}
			>
				{$_('import.export.download')}
			</button>
		</div>
	{:else}
		{#if error}
			<p class="text-sm" style:color="var(--danger)">{error}</p>
		{/if}

		{#if step === 'upload'}
			<div
				class="card flex min-h-48 flex-col items-center justify-center gap-3 border-2 border-dashed p-8 text-center transition-colors"
				class:border-primary={dragOver}
				style:border-color={dragOver ? 'var(--primary)' : 'var(--border)'}
				role="button"
				tabindex="0"
				ondragover={(e) => {
					e.preventDefault();
					dragOver = true;
				}}
				ondragleave={() => (dragOver = false)}
				ondrop={onDrop}
			>
				<p style:color="var(--text-muted)">{$_('import.upload.hint')}</p>
				<label class="btn-primary cursor-pointer">
					{$_('import.upload.choose')}
					<input
						type="file"
						class="sr-only"
						accept=".csv,.xlsx"
						onchange={(e) => onFileSelect((e.currentTarget as HTMLInputElement).files?.[0] ?? null)}
					/>
				</label>
			</div>
		{:else if step === 'settings'}
			<div class="card space-y-4">
				<p class="text-sm" style:color="var(--text-muted)">{file?.name}</p>
				<label class="flex items-center gap-2">
					<input type="radio" bind:group={preset} value="cubux" />
					<span>{$_('import.settings.preset_cubux')}</span>
				</label>
				<label class="flex items-center gap-2">
					<input type="radio" bind:group={preset} value="custom" />
					<span>{$_('import.settings.preset_custom')}</span>
				</label>
				<div class="flex items-center justify-between gap-4">
					<span class="text-sm">{$_('import.settings.deduplicate')}</span>
					<ToggleSwitch
						checked={deduplicate}
						label={$_('import.settings.deduplicate')}
						onchange={() => (deduplicate = !deduplicate)}
					/>
				</div>
				<div class="flex gap-2">
					<button type="button" class="btn-ghost" onclick={reset}>{$_('common.cancel')}</button>
					<button
						type="button"
						class="btn-primary"
						disabled={loading}
						onclick={() => void goFromSettings()}
					>
						{loading ? $_('common.loading') : $_('import.settings.next')}
					</button>
				</div>
			</div>
		{:else if step === 'mapping'}
			<div class="card space-y-4">
				<h2 class="text-lg font-medium">{$_('import.mapping.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('import.mapping.hint')}</p>
				<div class="hidden overflow-x-auto md:block">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b" style:border-color="var(--border)">
								<th class="py-2 text-left">{$_('import.mapping.field')}</th>
								<th class="py-2 text-left">{$_('import.mapping.column')}</th>
							</tr>
						</thead>
						<tbody>
							{#each IMPORT_COLUMN_FIELDS as field (field.id)}
								<tr class="border-b" style:border-color="var(--border)">
									<td class="py-2">
										{$_(`import.mapping.fields.${field.id}`)}
										{#if field.required}
											<span style:color="var(--danger)">*</span>
										{/if}
									</td>
									<td class="py-2">
										<Select
											controlled
											value={columnMap[field.id] ?? ''}
											onchange={(next) => setColumnField(field.id, next)}
											options={headerOptions}
											usePortal
										/>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
				<div class="space-y-3 md:hidden">
					{#each IMPORT_COLUMN_FIELDS as field (field.id)}
						<article class="rounded-xl border p-3" style:border-color="var(--border)">
							<p class="text-sm font-medium">
								{$_(`import.mapping.fields.${field.id}`)}
								{#if field.required}
									<span style:color="var(--danger)">*</span>
								{/if}
							</p>
							<div class="mt-2">
								<Select
									controlled
									value={columnMap[field.id] ?? ''}
									onchange={(next) => setColumnField(field.id, next)}
									options={headerOptions}
									usePortal
								/>
							</div>
						</article>
					{/each}
				</div>
				<div class="flex gap-2">
					<button type="button" class="btn-ghost" onclick={() => (step = 'settings')}>
						{$_('import.mapping.back')}
					</button>
					<button type="button" class="btn-primary" disabled={loading} onclick={goFromMapping}>
						{loading ? $_('common.loading') : $_('import.settings.next')}
					</button>
				</div>
			</div>
		{:else if step === 'accounts' && report}
			<div class="card space-y-4">
				<h2 class="text-lg font-medium">{$_('import.accounts.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('import.accounts.hint')}</p>
				<p class="text-xs" style:color="var(--text-muted)">
					{$_('import.accounts.unique_note', {
						values: { count: report.account_mappings?.length ?? 0 }
					})}
				</p>
				<div
					class="hidden gap-x-4 px-1 text-xs font-medium tracking-wide sm:grid sm:grid-cols-[minmax(5rem,0.75fr)_minmax(9rem,1fr)_minmax(12rem,2fr)]"
					style:color="var(--text-muted)"
				>
					<span>{$_('import.accounts.file_name')}</span>
					<span>{$_('import.accounts.action')}</span>
					<span>{$_('import.accounts.account')}</span>
				</div>
				<div class="divide-y rounded-xl border" style:border-color="var(--border)">
					{#each report.account_mappings ?? [] as m (m.file_name)}
						{@const entry = mapEntry(m)}
						<div
							class="grid grid-cols-1 gap-3 p-3 sm:grid-cols-[minmax(5rem,0.75fr)_minmax(9rem,1fr)_minmax(12rem,2fr)] sm:items-center sm:gap-x-4 sm:py-3"
						>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.accounts.file_name')}
								</span>
								<span class="block truncate text-sm font-medium">{m.file_name}</span>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.accounts.action')}
								</span>
								<Select
									controlled
									value={entry.mode}
									onchange={(next) => setAccountMapMode(m.file_name, next as 'create' | 'existing')}
									options={accountModeOptions}
									usePortal
								/>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.accounts.account')}
								</span>
								{#if entry.mode === 'existing'}
									<Select
										controlled
										value={entry.account_id}
										onchange={(next) => setAccountMapExisting(m.file_name, next)}
										options={[
											{ value: '', label: '—' },
											...accounts.map((acc) => ({
												value: acc.id,
												label: accountOptionLabel(acc)
											}))
										]}
										usePortal
									/>
								{:else}
									<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
										<div
											class="flex shrink-0 rounded-xl border p-0.5"
											style:border-color="var(--border)"
										>
											<button
												type="button"
												class={(entry.account_type ?? 'cash') === 'cash'
													? 'tab tab-active !px-3 !py-1.5 text-xs'
													: 'tab !px-3 !py-1.5 text-xs'}
												onclick={() => setAccountMapType(m.file_name, 'cash')}
											>
												{$_('accounts.type.cash')}
											</button>
											<button
												type="button"
												class={(entry.account_type ?? 'cash') === 'bank'
													? 'tab tab-active !px-3 !py-1.5 text-xs'
													: 'tab !px-3 !py-1.5 text-xs'}
												onclick={() => setAccountMapType(m.file_name, 'bank')}
											>
												{$_('accounts.type.bank')}
											</button>
										</div>
										{#if (entry.account_type ?? 'cash') === 'bank'}
											<Select
												controlled
												value={entry.bank_id}
												onchange={(next) => setAccountMapBank(m.file_name, next)}
												options={bankSelectOptions}
												usePortal
											/>
										{/if}
									</div>
								{/if}
							</div>
						</div>
					{/each}
				</div>
				{#if error}
					<p class="text-sm" style:color="var(--danger)">{error}</p>
				{/if}
				<button type="button" class="btn-primary" onclick={goFromAccounts}>
					{$_('import.settings.next')}
				</button>
			</div>
		{:else if step === 'categories' && report}
			<div class="card space-y-4">
				<h2 class="text-lg font-medium">{$_('import.categories.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('import.categories.hint')}</p>
				<p class="text-xs" style:color="var(--text-muted)">
					{$_('import.categories.unique_note', {
						values: { count: report.category_mappings?.length ?? 0 }
					})}
				</p>
				<div
					class="hidden gap-x-4 px-1 text-xs font-medium tracking-wide sm:grid sm:grid-cols-[minmax(6rem,0.9fr)_minmax(9rem,1fr)_minmax(12rem,2fr)]"
					style:color="var(--text-muted)"
				>
					<span>{$_('import.categories.file_name')}</span>
					<span>{$_('import.categories.action')}</span>
					<span>{$_('import.categories.category')}</span>
				</div>
				<div
					class="max-h-[28rem] divide-y overflow-y-auto rounded-xl border"
					style:border-color="var(--border)"
				>
					{#each report.category_mappings ?? [] as m (categoryMapKey(m))}
						{@const key = categoryMapKey(m)}
						{@const entry = categoryMapEntry(m)}
						<div
							class="grid grid-cols-1 gap-3 p-3 sm:grid-cols-[minmax(6rem,0.9fr)_minmax(9rem,1fr)_minmax(12rem,2fr)] sm:items-center sm:gap-x-4 sm:py-3"
						>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.categories.file_name')}
								</span>
								<div class="flex min-w-0 flex-wrap items-center gap-2">
									<span class="truncate text-sm font-medium">{m.file_name}</span>
									<span class="badge shrink-0">{categoryTypeLabel(m.type)}</span>
								</div>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.categories.action')}
								</span>
								<Select
									controlled
									value={entry.mode}
									onchange={(next) => setCategoryMapMode(key, m, next as 'create' | 'existing')}
									options={categoryModeOptions}
									usePortal
								/>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.categories.category')}
								</span>
								{#if entry.mode === 'existing'}
									<Select
										controlled
										value={entry.category_id}
										onchange={(next) => setCategoryMapExisting(key, next)}
										options={[
											{ value: '', label: '—' },
											...categoriesForType(m.type).map((cat) => ({
												value: cat.id,
												label: cat.name
											}))
										]}
										usePortal
									/>
								{:else}
									<span class="text-sm" style:color="var(--text-muted)">
										{$_('import.categories.new_category', {
											values: { name: m.file_name, type: categoryTypeLabel(m.type) }
										})}
									</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
				<div
					class="flex items-center justify-between gap-4 text-sm"
					style:color="var(--text-muted)"
				>
					<span>{$_('import.subcategories.auto_toggle')}</span>
					<ToggleSwitch
						checked={autoSubcategory}
						label={$_('import.subcategories.auto_toggle')}
						onchange={() => (autoSubcategory = !autoSubcategory)}
					/>
				</div>
				{#if error}
					<p class="text-sm" style:color="var(--danger)">{error}</p>
				{/if}
				<div class="flex gap-2">
					<button
						type="button"
						class="btn-ghost"
						onclick={() =>
							(step = (report?.account_mappings?.length ?? 0) > 0 ? 'accounts' : 'settings')}
					>
						{$_('import.mapping.back')}
					</button>
					<button type="button" class="btn-primary" onclick={goFromCategories}>
						{$_('import.settings.next')}
					</button>
				</div>
			</div>
		{:else if step === 'subcategories' && report}
			<div class="card space-y-4">
				<h2 class="text-lg font-medium">{$_('import.subcategories.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('import.subcategories.hint')}</p>
				<p class="text-xs" style:color="var(--text-muted)">
					{$_('import.subcategories.unique_note', {
						values: { count: report.subcategory_mappings?.length ?? 0 }
					})}
				</p>
				<div
					class="hidden gap-x-4 px-1 text-xs font-medium tracking-wide sm:grid sm:grid-cols-[minmax(7rem,1fr)_minmax(9rem,1fr)_minmax(12rem,2fr)]"
					style:color="var(--text-muted)"
				>
					<span>{$_('import.subcategories.file_name')}</span>
					<span>{$_('import.subcategories.action')}</span>
					<span>{$_('import.subcategories.subcategory')}</span>
				</div>
				<div
					class="max-h-[28rem] divide-y overflow-y-auto rounded-xl border"
					style:border-color="var(--border)"
				>
					{#each report.subcategory_mappings ?? [] as m (subcategoryMapKey(m))}
						{@const key = subcategoryMapKey(m)}
						{@const entry = subcategoryMapEntry(m)}
						<div
							class="grid grid-cols-1 gap-3 p-3 sm:grid-cols-[minmax(7rem,1fr)_minmax(9rem,1fr)_minmax(12rem,2fr)] sm:items-center sm:gap-x-4 sm:py-3"
						>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.subcategories.file_name')}
								</span>
								<div class="flex min-w-0 flex-wrap items-center gap-2">
									<span class="truncate text-sm font-medium">{m.file_subcategory}</span>
									<span class="badge shrink-0">{m.file_category}</span>
									<span class="badge shrink-0">{categoryTypeLabel(m.type)}</span>
								</div>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.subcategories.action')}
								</span>
								<Select
									controlled
									value={entry.mode}
									onchange={(next) => setSubcategoryMapMode(key, m, next as 'create' | 'existing')}
									options={subcategoryModeOptions}
									usePortal
								/>
							</div>
							<div class="min-w-0">
								<span
									class="mb-1 block text-xs font-medium sm:hidden"
									style:color="var(--text-muted)"
								>
									{$_('import.subcategories.subcategory')}
								</span>
								{#if entry.mode === 'existing'}
									{#if subcategoryOptionsFor(m).length > 0}
										<Select
											controlled
											value={entry.subcategory_id}
											onchange={(next) => setSubcategoryMapExisting(key, next)}
											options={[
												{ value: '', label: '—' },
												...subcategoryOptionsFor(m).map((sub) => ({
													value: sub.id,
													label: sub.name
												}))
											]}
											usePortal
										/>
									{:else}
										<span class="text-sm" style:color="var(--text-muted)">
											{$_('import.subcategories.no_options')}
										</span>
									{/if}
								{:else}
									<span class="text-sm" style:color="var(--text-muted)">
										{$_('import.subcategories.new_subcategory', {
											values: { name: m.file_subcategory }
										})}
									</span>
								{/if}
							</div>
						</div>
					{/each}
				</div>
				{#if error}
					<p class="text-sm" style:color="var(--danger)">{error}</p>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-ghost" onclick={() => (step = 'categories')}>
						{$_('import.mapping.back')}
					</button>
					<button type="button" class="btn-primary" onclick={goFromSubcategories}>
						{$_('import.settings.next')}
					</button>
				</div>
			</div>
		{:else if step === 'preview' && report}
			<div class="card space-y-4">
				<h2 class="text-lg font-medium">{$_('import.preview.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('import.preview.hint')}</p>
				<div
					class="grid gap-3 rounded-xl border p-4 text-sm sm:grid-cols-2"
					style:border-color="var(--border)"
				>
					<span>{$_('import.preview.total')}: {report.total_rows}</span>
					<span>{$_('import.preview.valid')}: {report.valid_rows}</span>
					<span>{$_('import.preview.duplicates')}: {report.skipped_duplicates}</span>
					<span>{$_('import.preview.errors')}: {(report.errors ?? []).length}</span>
				</div>
				{#if (report.errors ?? []).length > 0}
					<details>
						<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
							{$_('import.preview.error_list')}
						</summary>
						<ul class="mt-2 space-y-1 text-sm" style:color="var(--danger)">
							{#each (report.errors ?? []).slice(0, 20) as err (`${err.row}-${err.message}`)}
								<li>
									{$_('import.preview.error_row', {
										values: { row: err.row, message: err.message }
									})}
								</li>
							{/each}
						</ul>
					</details>
				{/if}
				<div class="flex gap-2">
					<button
						type="button"
						class="btn-ghost"
						onclick={() => {
							if (!autoSubcategory && (report?.subcategory_mappings?.length ?? 0) > 0)
								step = 'subcategories';
							else if ((report?.category_mappings?.length ?? 0) > 0) step = 'categories';
							else if ((report?.account_mappings?.length ?? 0) > 0) step = 'accounts';
							else step = 'settings';
						}}
					>
						{$_('import.mapping.back')}
					</button>
					<button type="button" class="btn-ghost" onclick={reset}>{$_('common.cancel')}</button>
					<button
						type="button"
						class="btn-primary"
						disabled={loading}
						onclick={() => void runImport()}
					>
						{loading ? $_('common.loading') : $_('import.preview.import')}
					</button>
				</div>
			</div>
		{:else if step === 'done' && finalReport}
			<div class="card space-y-3">
				<h2 class="text-lg font-medium">{$_('import.done.title')}</h2>
				<div
					class="grid gap-3 rounded-xl border p-4 text-sm sm:grid-cols-2"
					style:border-color="var(--border)"
				>
					<span>{$_('import.preview.total')}: {finalReport.total_rows}</span>
					<span>{$_('import.preview.valid')}: {finalReport.valid_rows}</span>
					<span>{$_('import.preview.duplicates')}: {finalReport.skipped_duplicates}</span>
					<span>{$_('import.preview.errors')}: {(finalReport.errors ?? []).length}</span>
				</div>
				<p>
					{$_('import.done.created', { values: { count: finalReport.created_transactions ?? 0 } })}
				</p>
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('import.done.skipped', { values: { count: finalReport.skipped_duplicates } })}
				</p>
				{#if (finalReport.errors ?? []).length > 0}
					<details>
						<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
							{$_('import.preview.error_list')}
						</summary>
						<ul class="mt-2 space-y-1 text-sm" style:color="var(--danger)">
							{#each finalReport.errors ?? [] as err (`done-${err.row}-${err.message}`)}
								<li>
									{$_('import.preview.error_row', {
										values: { row: err.row, message: err.message }
									})}
								</li>
							{/each}
						</ul>
					</details>
				{/if}
				{#if (finalReport.logs ?? []).length > 0}
					<details>
						<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
							{$_('import.job.full_log', { values: { count: (finalReport.logs ?? []).length } })}
						</summary>
						<div
							class="mt-2 max-h-72 overflow-auto rounded-lg border p-2 font-mono text-xs"
							style:border-color="var(--border)"
						>
							{#each finalReport.logs ?? [] as line, i (`done-log-${i}-${line}`)}
								<div>{line}</div>
							{/each}
						</div>
					</details>
				{/if}
				<a href={resolve('/transactions')} class="btn-primary inline-flex"
					>{$_('import.done.view_tx')}</a
				>
				<button type="button" class="btn-ghost" onclick={reset}>{$_('import.done.another')}</button>
			</div>
		{:else if step === 'importing' && importJob}
			{@const jobReport = normalizeImportReport(importJob.report ?? emptyReport())}
			{@const processedRows = Math.min(jobReport.processed_rows ?? 0, jobReport.total_rows ?? 0)}
			{@const progress = progressPercent(jobReport.total_rows, processedRows)}
			{@const eta = etaSeconds(jobReport.total_rows, processedRows, importJob.started_at)}
			<div class="card space-y-3">
				<h2 class="text-lg font-medium">{$_('import.job.title')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('import.job.hint', { values: { file: importJob.filename } })}
				</p>
				<div
					class="rounded-xl border px-4 py-3 text-sm"
					style:border-color="var(--border); color: var(--text-muted)"
				>
					{#if importJob.status === 'queued'}
						{$_('import.job.status.queued')}
					{:else if importJob.status === 'running'}
						{$_('import.job.status.running')}
					{:else if importJob.status === 'done'}
						{$_('import.job.status.done')}
					{:else}
						{$_('import.job.status.failed')}
					{/if}
				</div>
				{#if jobReport.total_rows > 0}
					<div class="space-y-2">
						<div class="flex items-center justify-between text-sm">
							<span>{$_('import.job.progress')}</span>
							<span>{processedRows} / {jobReport.total_rows} ({progress}%)</span>
						</div>
						<p class="text-sm" style:color="var(--text-muted)">
							{#if eta !== null}
								{$_('import.job.eta', { values: { value: formatEta(eta) } })}
							{:else}
								{$_('import.job.eta_unknown')}
							{/if}
						</p>
						<div class="h-2 rounded-full" style:background-color="var(--bg-muted)">
							<div
								class="h-2 rounded-full transition-all"
								style:width={`${progress}%`}
								style:background-color="var(--primary)"
							></div>
						</div>
						<div
							class="grid gap-3 rounded-xl border p-3 text-sm sm:grid-cols-3"
							style:border-color="var(--border)"
						>
							<span
								>{$_('import.done.created', {
									values: { count: jobReport.created_transactions ?? 0 }
								})}</span
							>
							<span>{$_('import.preview.duplicates')}: {jobReport.skipped_duplicates}</span>
							<span>{$_('import.preview.errors')}: {(jobReport.errors ?? []).length}</span>
						</div>
					</div>
				{/if}
				{#if (jobReport.logs ?? []).length > 0}
					<details>
						<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
							{$_('import.job.full_log', { values: { count: (jobReport.logs ?? []).length } })}
						</summary>
						<div
							class="mt-2 max-h-72 overflow-auto rounded-lg border p-2 font-mono text-xs"
							style:border-color="var(--border)"
						>
							{#each jobReport.logs ?? [] as line, i (`run-log-${i}-${line}`)}
								<div>{line}</div>
							{/each}
						</div>
					</details>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-ghost" onclick={reset}>{$_('common.cancel')}</button>
				</div>
			</div>
		{/if}
	{/if}
</div>
