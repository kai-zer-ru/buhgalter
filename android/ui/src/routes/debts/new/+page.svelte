<script lang="ts">
	import { page } from '$app/stores';
	import DebtForm from '$lib/components/DebtForm.svelte';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { dataRefreshTick } from '$lib/offline/sync';

	const direction = $derived(
		$page.url.searchParams.get('direction') === 'borrowed' ? 'borrowed' : 'lent'
	);
	const debtorId = $derived($page.url.searchParams.get('debtor') ?? '');
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/debts'));

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

<DebtForm
	variant="page"
	backHref={returnTo}
	defaultDirection={direction}
	debtorId={debtorId || undefined}
	onclose={finish}
	onsaved={finish}
/>
