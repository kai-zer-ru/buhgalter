<script lang="ts">
	import { page } from '$app/stores';
	import { getDebt, type Debt } from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import SettleDebtForm from '$lib/components/SettleDebtForm.svelte';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const debtId = $derived($page.params.id ?? '');
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/debts'));

	let debt = $state<Debt | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	$effect(() => {
		if (!debtId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			debt = await getDebt(debtId);
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!debt });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if debt}
		<SettleDebtForm variant="page" {debt} backHref={returnTo} onclose={finish} onsaved={finish} />
	{/if}
</PageLoadGate>
