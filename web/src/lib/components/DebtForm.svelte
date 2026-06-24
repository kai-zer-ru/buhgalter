<script lang="ts">
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		createDebt,
		listAccounts,
		listDebtors,
		listDebts,
		type Account,
		type Debt,
		type Debtor
	} from '$lib/api/client';
	import { ApiError } from '$lib/api/client';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import { defaultAccountId } from '$lib/accounts';
	import { toast } from '$lib/toast';
	import { fromDatetimeLocalValue, nowDatetimeLocal } from '$lib/dates';
	import { toAPIAmount } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		onclose: () => void;
		onsaved: () => void;
		/** Фиксированный должник — скрывает выбор должника */
		debtorId?: string;
		debtorName?: string;
		/** Фиксированное направление — скрывает выбор направления */
		defaultDirection?: 'lent' | 'borrowed';
	};

	let {
		open = $bindable(),
		onclose,
		onsaved,
		debtorId: fixedDebtorId,
		debtorName,
		defaultDirection
	}: Props = $props();

	const compact = $derived(Boolean(fixedDebtorId));
	const fixedDirection = $derived(defaultDirection !== undefined);

	let direction = $state<'lent' | 'borrowed'>('lent');
	let amount = $state('');
	let debtorId = $state('');
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
	let error = $state('');

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
	const accountOptions = $derived(accounts.map((acc) => ({ value: acc.id, label: acc.name })));

	const modalTitle = $derived.by(() => {
		void $locale;
		return compact && debtorName
			? `${direction === 'lent' ? tr('debts.new.lentTo') : tr('debts.new.borrowFrom')} ${debtorName}`
			: defaultDirection === 'lent'
				? tr('debts.action.lend')
				: defaultDirection === 'borrowed'
					? tr('debts.action.borrow')
					: tr('debts.new');
	});

	$effect(() => {
		if (!open) return;
		void init();
	});

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
		error = '';
		direction = defaultDirection ?? 'lent';
		amount = '';
		debtorId = fixedDebtorId ?? '';
		newDebtorName = '';
		const now = nowDatetimeLocal(tz);
		debtDateLocal = now;
		dueDateLocal = now;
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
		}
		accountId = defaultAccountId(accounts);
	}

	async function save() {
		saving = true;
		error = '';
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
			const due_date = fromDatetimeLocalValue(dueDateLocal, tz);
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
			open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError
					? err.message
					: err instanceof Error
						? err.message
						: $_('common.error');
		} finally {
			saving = false;
		}
	}
</script>

<ModalShell bind:open title={modalTitle} {onclose}>
	<div class="space-y-4">
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
			usePortal
			required
		/>

		<DateTimePicker
			label={$_('debts.field.dueDate')}
			bind:value={dueDateLocal}
			usePortal
			required
		/>

		<div class="space-y-1">
			<label class="flex items-start gap-2">
				<input type="checkbox" bind:checked={skipBalance} class="mt-1" />
				<span class="text-sm">{$_('debts.field.noBalance')}</span>
			</label>
			<FieldHint text={$_('debts.field.noBalanceHint')} />
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

		<FormFeedback error={directionConflict || error} />
	</div>
	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={onclose}>{$_('common.cancel')}</button>
		<button
			type="button"
			class="btn-primary"
			disabled={saving || Boolean(directionConflict)}
			onclick={() => void save()}
		>
			{saving ? $_('common.loading') : $_('common.create')}
		</button>
	{/snippet}
</ModalShell>
