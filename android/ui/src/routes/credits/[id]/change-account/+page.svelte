<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		getCredit,
		listAccounts,
		updateCredit,
		type Account,
		type Credit
	} from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import { accountSelectOptions } from '$lib/select-options';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const creditId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/credits/${creditId}`)
	);

	let credit = $state<Credit | null>(null);
	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let newAccountId = $state('');

	const accountOptions = $derived(accountSelectOptions(accounts));

	$effect(() => {
		if (!creditId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			const [c, accs] = await Promise.all([getCredit(creditId), listAccounts()]);
			credit = c;
			accounts = accs.filter((a) => a.status === 'active');
			newAccountId = c.debit_account_id;
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
			await updateCredit(credit.id, { debit_account_id: newAccountId });
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
		<FormPageShell title={$_('credits.action.changeAccount')} backHref={returnTo} onback={finish}>
			<Select
				label={$_('transactions.field.account')}
				bind:value={newAccountId}
				options={accountOptions}
			/>
			{#snippet footer()}
				<button type="button" class="btn-ghost" onclick={finish}>{$_('common.cancel')}</button>
				<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
					{saving ? $_('common.loading') : $_('common.save')}
				</button>
			{/snippet}
		</FormPageShell>
	{/if}
</PageLoadGate>
