<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { completeCredit, getCredit, type Credit } from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import { fromDatetimeLocalValue, todayDateLocal } from '$lib/dates';
	import { formatMoneyForDisplay } from '$lib/money-display';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const creditId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/credits/${creditId}`)
	);
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const currency = $derived($user?.currency ?? 'RUB');

	let credit = $state<Credit | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let completeDateLocal = $state('');
	let completeMode = $state<'account' | 'skip'>('account');

	$effect(() => {
		if (!creditId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			const c = await getCredit(creditId);
			credit = c;
			completeDateLocal = todayDateLocal(tz);
			completeMode = c.remaining_amount > 0 ? 'account' : 'skip';
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!credit });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}

	async function save() {
		if (!credit) return;
		saving = true;
		try {
			await completeCredit(credit.id, {
				affects_balance: completeMode === 'account',
				payment_date: fromDatetimeLocalValue(completeDateLocal, tz)
			});
			toast($_('common.saved'));
			finish();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if credit}
		{@const activeCredit = credit}
		<FormPageShell title={$_('credits.complete.title')} backHref={returnTo} onback={finish}>
			<div class="space-y-4">
				{#if activeCredit.remaining_amount > 0}
					<div class="space-y-2">
						<label class="flex cursor-pointer items-start gap-2">
							<input
								type="radio"
								class="mt-1"
								name="complete-mode"
								value="account"
								bind:group={completeMode}
							/>
							<span class="text-sm">
								{$_('credits.complete.payFromAccount', {
									values: {
										amount: formatMoneyForDisplay({
											value: activeCredit.remaining_amount_display,
											currency
										}),
										account: activeCredit.debit_account_name
									}
								})}
							</span>
						</label>
						<label class="flex cursor-pointer items-start gap-2">
							<input
								type="radio"
								class="mt-1"
								name="complete-mode"
								value="skip"
								bind:group={completeMode}
							/>
							<span class="text-sm">{$_('credits.complete.skipBalance')}</span>
						</label>
						<FieldHint text={$_('credits.complete.skipBalanceHint')} />
					</div>
				{/if}
				<DateTimePicker
					label={$_('credits.complete.date')}
					bind:value={completeDateLocal}
					{...dateOnlyPicker}
				/>
			</div>
			{#snippet footer()}
				<button type="button" class="btn-ghost" onclick={finish}>{$_('common.cancel')}</button>
				<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
					{saving ? $_('common.loading') : $_('credits.action.complete')}
				</button>
			{/snippet}
		</FormPageShell>
	{/if}
</PageLoadGate>
