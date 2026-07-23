<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import { createDebt } from '$lib/offline/debts-api';
	import {
		getDebtor,
		listAccounts,
		listDebtors,
		listDebts,
		type Account,
		type Debt,
		type Debtor
	} from '$lib/api/client';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker, operationDatetimePickerCreate } from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import {
		fromDateLocalEnd,
		fromDatetimeLocalValue,
		nowDatetimeLocal,
		todayDateLocal
	} from '$lib/dates';
	import { toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';
	import { refCacheUpdate } from '$lib/offline/ref-cache';
	import { refCachePathMatches } from '$lib/offline/ref-cache-watch';

	type Props = {
		variant?: 'modal' | 'page';
		open?: boolean;
		backHref?: string;
		onclose: () => void;
		onsaved: () => void;
		/** Фиксированный должник — скрывает выбор должника */
		debtorId?: string;
		debtorName?: string;
		/** Фиксированное направление — скрывает выбор направления */
		defaultDirection?: 'lent' | 'borrowed';
	};

	let {
		variant = 'modal',
		open = $bindable(false),
		backHref = '/debts',
		onclose,
		onsaved,
		debtorId: fixedDebtorId,
		debtorName: fixedDebtorName,
		defaultDirection
	}: Props = $props();

	const compact = $derived(Boolean(fixedDebtorId));
	const fixedDirection = $derived(defaultDirection !== undefined);

	let direction = $state<'lent' | 'borrowed'>('lent');
	let amount = $state('');
	let debtorId = $state('');
	let debtorName = $state('');
	let newDebtorName = $state('');
	let debtDateLocal = $state('');
	let dueDateLocal = $state('');
	let skipBalance = $state(false);
	let accountId = $state('');
	let description = $state('');
	let debtors = $state<Debtor[]>([]);
	let accounts = $state<Account[]>([]);
	let activeDebts = $state<Debt[]>([]);
	let saving = $state(false);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');

	const directionOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: 'lent', label: tr('debts.direction.lent') },
			{ value: 'borrowed', label: tr('debts.direction.borrowed') }
		];
	});
	const debtorOptions = $derived.by(() => {
		void $locale;
		return [
			{ value: '', label: tr('debts.field.newDebtor') },
			...debtors.map((debtor) => ({ value: debtor.id, label: debtor.name }))
		];
	});
	const accountOptions = $derived(accountSelectOptions(accounts));

	const pageTitle = $derived.by(() => {
		void $locale;
		const name = fixedDebtorName || debtorName;
		return compact && name
			? `${direction === 'lent' ? tr('debts.new.lentTo') : tr('debts.new.borrowFrom')} ${name}`
			: defaultDirection === 'lent'
				? tr('debts.action.lend')
				: defaultDirection === 'borrowed'
					? tr('debts.action.borrow')
					: tr('debts.new');
	});

	$effect(() => {
		if (variant === 'page') {
			void init();
			return;
		}
		if (!open) return;
		void init();
	});

	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || compact) return;
		if (!refCachePathMatches(update.path, '/api/v1/debtors')) return;
		void reloadDebtors();
	});

	async function reloadDebtors() {
		try {
			debtors = await listDebtors();
		} catch {
			// keep current list on transient errors
		}
	}

	const directionConflict = $derived.by(() => {
		if (!debtorId) return '';
		const dirs = new Set(
			activeDebts.filter((d) => d.debtor_id === debtorId).map((d) => d.direction)
		);
		if (direction === 'borrowed' && dirs.has('lent')) {
			return $_('debts.error.cannotBorrow');
		}
		if (direction === 'lent' && dirs.has('borrowed')) {
			return $_('debts.error.cannotLend');
		}
		return '';
	});

	async function init() {
		direction = defaultDirection ?? 'lent';
		amount = '';
		debtorId = fixedDebtorId ?? '';
		debtorName = fixedDebtorName ?? '';
		newDebtorName = '';
		debtDateLocal = nowDatetimeLocal(tz);
		dueDateLocal = todayDateLocal(tz);
		skipBalance = false;
		description = '';
		const [accountsData, activeList] = await Promise.all([
			listAccounts('active'),
			listDebts({ settled: 'false' })
		]);
		accounts = accountsData;
		activeDebts = activeList;
		if (!compact) {
			debtors = await listDebtors();
		} else if (fixedDebtorId && !fixedDebtorName) {
			try {
				const debtor = await getDebtor(fixedDebtorId);
				debtorName = debtor.name;
			} catch {
				// title falls back without name
			}
		}
		accountId = defaultAccountId(accounts);
	}

	function close() {
		if (variant === 'page') {
			onclose();
			return;
		}
		open = false;
		onclose();
	}

	async function save(event: Event) {
		event.preventDefault();
		saving = true;
		try {
			if (directionConflict) {
				throw new Error(directionConflict);
			}
			if (!debtorId && !newDebtorName.trim()) {
				throw new Error($_('debts.field.debtor'));
			}
			if (!debtDateLocal) {
				throw new Error($_('debts.field.debtDate'));
			}
			if (!dueDateLocal) {
				throw new Error($_('debts.field.dueDate'));
			}
			const debt_date = fromDatetimeLocalValue(debtDateLocal, tz);
			const due_date = fromDateLocalEnd(dueDateLocal, tz);
			await createDebt({
				...(debtorId ? { debtor_id: debtorId } : { debtor_name: newDebtorName.trim() }),
				direction,
				amount: toAPIAmount(amount),
				debt_date,
				due_date,
				affects_balance: !skipBalance,
				description: description.trim() || undefined,
				account_id: !skipBalance ? accountId : undefined
			});
			if (variant !== 'page') {
				open = false;
			}
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

{#snippet formBody()}
	<form id="debt-form" class="space-y-4" onsubmit={(e) => void save(e)}>
		{#if !fixedDirection}
			<Select
				label={$_('debts.field.direction')}
				bind:value={direction}
				options={directionOptions}
				usePortal
			/>
		{/if}

		{#if !compact}
			<Select
				label={$_('debts.field.debtor')}
				bind:value={debtorId}
				options={debtorOptions}
				usePortal
			/>

			{#if !debtorId}
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('debts.field.debtorName')}</span
					>
					<input class="input w-full" bind:value={newDebtorName} />
				</label>
			{/if}
		{/if}

		<label class="block space-y-1">
			<span class="text-sm" style:color="var(--text-muted)">{$_('transactions.field.amount')}</span>
			<MoneyInput bind:value={amount} />
		</label>

		<DateTimePicker
			label={$_('debts.field.debtDate')}
			bind:value={debtDateLocal}
			{...operationDatetimePickerCreate}
			usePortal
			required
		/>

		<DateTimePicker
			label={$_('debts.field.dueDate')}
			bind:value={dueDateLocal}
			{...dateOnlyPicker}
			usePortal
			required
		/>

		<div class="space-y-1">
			<div class="flex items-center justify-between gap-4">
				<div>
					<p class="text-sm">{$_('debts.field.noBalance')}</p>
					<FieldHint text={$_('debts.field.noBalanceHint')} />
				</div>
				<ToggleSwitch
					checked={skipBalance}
					label={$_('debts.field.noBalance')}
					onchange={() => (skipBalance = !skipBalance)}
				/>
			</div>
		</div>

		{#if !skipBalance}
			<Select
				label={$_('transactions.field.account')}
				bind:value={accountId}
				options={accountOptions}
				usePortal
			/>
		{/if}

		<label class="block space-y-1">
			<span class="text-sm" style:color="var(--text-muted)"
				>{$_('transactions.field.description')}</span
			>
			<input class="input w-full" bind:value={description} />
		</label>

		{#if directionConflict}
			<p class="text-sm" style:color="var(--danger)" role="alert">{directionConflict}</p>
		{/if}
	</form>
{/snippet}

{#snippet formFooter()}
	<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
	<button
		type="submit"
		form="debt-form"
		class="btn-primary"
		disabled={saving || Boolean(directionConflict)}
	>
		{saving ? $_('common.loading') : $_('common.create')}
	</button>
{/snippet}

{#if variant === 'page'}
	<FormPageShell title={pageTitle} {backHref} onback={close}>
		{@render formBody()}
		{#snippet footer()}
			{@render formFooter()}
		{/snippet}
	</FormPageShell>
{:else}
	<ModalShell bind:open title={pageTitle} onclose={close}>
		{@render formBody()}
		{#snippet footer()}
			{@render formFooter()}
		{/snippet}
	</ModalShell>
{/if}
