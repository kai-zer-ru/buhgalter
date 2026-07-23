<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import TransactionPagination from '$lib/components/TransactionPagination.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { gotoReplace } from '$lib/android/form-nav';
	import {
		canToggleRetroDebit,
		creditCreateDraft,
		ensureManualScheduleRows,
		isManualInterval,
		isRetroDebited,
		patchCreditCreate,
		refreshCreditCreateSchedule,
		retroRowIndices,
		rowStatus,
		schedulePageSize,
		scheduleRowsComplete,
		submitCreditCreate,
		toggleRetroDebit,
		validateReadyToSave,
		type ScheduleRow
	} from '$lib/credits/create-draft';
	import {
		creditCreateReturnTo,
		ensureCreditCreateDraft,
		goCreditCreateStep,
		prevCreditCreateStep
	} from '$lib/credits/create-nav';

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const fromRaw = $derived($page.url.searchParams.get('from'));
	const returnTo = $derived(creditCreateReturnTo(fromRaw));

	let ready = $state(false);
	let saving = $state(false);
	let rows = $state<ScheduleRow[]>([]);
	let scheduleLoading = $state(false);
	let scheduleError = $state('');
	let schedulePage = $state(1);
	let retroactive = $state(false);
	let manual = $state(false);

	const totalPages = $derived(Math.max(1, Math.ceil(rows.length / schedulePageSize)));
	const pageSafe = $derived(Math.min(Math.max(1, schedulePage), totalPages));
	const visible = $derived(
		rows
			.slice((pageSafe - 1) * schedulePageSize, pageSafe * schedulePageSize)
			.map((row, offset) => ({ row, index: (pageSafe - 1) * schedulePageSize + offset }))
	);

	onMount(() => {
		ensureCreditCreateDraft(tz);
		let d = get(creditCreateDraft);
		if (!d) {
			goCreditCreateStep('basics', fromRaw);
			return;
		}
		if (isManualInterval(d)) {
			d = ensureManualScheduleRows(d);
			creditCreateDraft.set(d);
		}
		syncFromDraft();
		ready = true;
		if (!manual) void refreshCreditCreateSchedule(tz).then(syncFromDraft);
	});

	function syncFromDraft() {
		const d = get(creditCreateDraft);
		if (!d) return;
		rows = d.scheduleRows.map((r) => ({ ...r }));
		scheduleLoading = d.scheduleLoading;
		scheduleError = d.scheduleError;
		schedulePage = d.schedulePage;
		retroactive = d.retroactive;
		manual = isManualInterval(d);
	}

	function persistRows() {
		let d = get(creditCreateDraft);
		if (!d) return;
		d = { ...d, scheduleRows: rows.map((r) => ({ ...r })), schedulePage };
		if (isManualInterval(d)) d = ensureManualScheduleRows(d);
		creditCreateDraft.set(d);
	}

	function draftForStatus() {
		return { ...get(creditCreateDraft)!, scheduleRows: rows, retroactive };
	}

	function onToggleRetro(index: number) {
		const d = draftForStatus();
		const next = toggleRetroDebit(d, index, tz);
		patchCreditCreate({ retroactiveDebitCount: next.retroactiveDebitCount });
		syncFromDraft();
	}

	async function save() {
		persistRows();
		const d = get(creditCreateDraft);
		if (!d) return;
		if (!scheduleRowsComplete(d)) {
			toast.error(manual ? $_('credits.error.manualIncomplete') : $_('credits.schedule.empty'));
			return;
		}
		const err = validateReadyToSave(d);
		if (err) {
			toast.error($_(err));
			return;
		}
		saving = true;
		try {
			const created = await submitCreditCreate(tz);
			dataRefreshTick.update((n) => n + 1);
			toast($_('common.saved'));
			void gotoReplace(created.id ? `/credits/${created.id}` : returnTo);
		} catch (e) {
			const key = (e as Error & { i18nKey?: string }).i18nKey;
			if (key) toast.error($_(key));
			else toast.fromError(e);
		} finally {
			saving = false;
		}
	}
</script>

{#if ready}
	<FormPageShell
		title={$_('credits.create.step.schedule')}
		onback={() => {
			persistRows();
			prevCreditCreateStep('schedule', fromRaw, returnTo);
		}}
	>
		<div class="space-y-3">
			<div class="flex items-center gap-2">
				<p class="text-sm font-medium">{$_('credits.schedule.title')}</p>
				{#if scheduleLoading}
					<span class="text-xs" style:color="var(--text-muted)"
						>{$_('credits.schedule.loading')}</span
					>
				{/if}
			</div>
			{#if scheduleError}
				<p class="text-sm text-red-600">{scheduleError}</p>
			{/if}
			{#if retroactive && retroRowIndices(draftForStatus(), tz).length > 0}
				<FieldHint text={$_('credits.field.retroactiveDebitHint')} />
			{/if}

			{#if rows.length === 0}
				<p class="text-sm" style:color="var(--text-muted)">
					{scheduleLoading ? $_('credits.schedule.loading') : $_('credits.schedule.empty')}
				</p>
			{:else}
				<div class="space-y-3" class:opacity-50={scheduleLoading}>
					{#each visible as item (item.index)}
						{@const status = rowStatus(draftForStatus(), rows[item.index], tz)}
						<article class="rounded-xl border p-3" style:border-color="var(--border)">
							<p class="mb-2 text-xs font-medium" style:color="var(--text-muted)">
								{$_('credits.schedule.paymentNumber', { values: { n: item.index + 1 } })}
							</p>
							<div class="space-y-3 text-sm">
								<DateTimePicker
									id={`credit-create-schedule-${item.index}`}
									bind:value={rows[item.index].date}
									label={$_('transactions.col.date')}
									{...dateOnlyPicker}
								/>
								<label class="block space-y-1">
									<span style:color="var(--text-muted)">{$_('transactions.col.amount')}</span>
									<MoneyInput bind:value={rows[item.index].amount} />
								</label>
								{#if status === 'retroactive'}
									<div class="flex items-center justify-between gap-2">
										<span style:color="var(--text-muted)"
											>{$_('credits.field.retroactiveDebit')}</span
										>
										<ToggleSwitch
											checked={isRetroDebited(draftForStatus(), item.index, tz)}
											disabled={!canToggleRetroDebit(draftForStatus(), item.index, tz)}
											label={$_('credits.field.retroactiveDebit')}
											onchange={() => onToggleRetro(item.index)}
										/>
									</div>
									<span class="badge">{$_('credits.payment.status.retroactive')}</span>
								{:else if status === 'pending'}
									<span style:color="var(--text-muted)">{$_('credits.payment.status.pending')}</span
									>
								{/if}
							</div>
						</article>
					{/each}
				</div>
				<TransactionPagination
					page={pageSafe}
					limit={schedulePageSize}
					total={rows.length}
					onchange={(p) => (schedulePage = p)}
					class="text-sm"
				/>
			{/if}
		</div>

		{#snippet footer()}
			<button
				type="button"
				class="btn-ghost"
				onclick={() => {
					persistRows();
					prevCreditCreateStep('schedule', fromRaw, returnTo);
				}}
			>
				{$_('common.back')}
			</button>
			<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
				{$_('common.save')}
			</button>
		{/snippet}
	</FormPageShell>
{/if}
