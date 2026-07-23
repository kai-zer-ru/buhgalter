<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { getCredit, listBanks, updateCredit, type Bank, type Credit } from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const creditId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/credits/${creditId}`)
	);

	let credit = $state<Credit | null>(null);
	let banks = $state<Bank[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let newBankId = $state('');

	const bankOptions = $derived([
		{ value: '', label: $_('credits.field.bankNotSelected') },
		...banks.map((bank) => ({ value: bank.id, label: bank.name }))
	]);

	$effect(() => {
		if (!creditId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			const [c, bankList] = await Promise.all([getCredit(creditId), listBanks()]);
			credit = c;
			banks = bankList;
			newBankId = c.bank_id ?? '';
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
			await updateCredit(credit.id, { bank_id: newBankId || null });
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
		<FormPageShell title={$_('credits.action.changeBank')} backHref={returnTo} onback={finish}>
			<Select label={$_('credits.field.bank')} bind:value={newBankId} options={bankOptions} />
			{#snippet footer()}
				<button type="button" class="btn-ghost" onclick={finish}>{$_('common.cancel')}</button>
				<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
					{saving ? $_('common.loading') : $_('common.save')}
				</button>
			{/snippet}
		</FormPageShell>
	{/if}
</PageLoadGate>
