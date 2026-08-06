<script lang="ts">
	import { page } from '$app/stores';
	import TransactionForm from '$lib/components/TransactionForm.svelte';
	import { getTransaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { takeSharePrefill } from '$lib/android/share-target';
	import { deleteInterceptDraft, takeInterceptPrefill } from '$lib/android/notification-intercept';
	import { lookupServerTransaction } from '$lib/offline/transaction-index';
	import type { Transaction } from '$lib/api/client';
	import { user } from '$lib/stores/auth';

	import { dataRefreshTick } from '$lib/offline/sync';

	const type = $derived($page.url.searchParams.get('type') === 'income' ? 'income' : 'expense');
	const accountId = $derived($page.url.searchParams.get('account') ?? '');
	const repeatId = $derived($page.url.searchParams.get('repeat'));
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));
	const descriptionParam = $derived($page.url.searchParams.get('description') ?? '');
	const shareOnce = takeSharePrefill();
	const interceptOnce = takeInterceptPrefill();
	let initialDescription = $state(interceptOnce?.description ?? shareOnce?.description ?? '');
	let createPrefill = $state(interceptOnce);
	const interceptDraftId = interceptOnce?.draftId;
	let repeatFrom = $state<Transaction | null>(null);
	let ready = $state(true);

	$effect(() => {
		if (descriptionParam) {
			initialDescription = descriptionParam;
		}
	});

	$effect(() => {
		if (!repeatId) {
			repeatFrom = null;
			ready = true;
			return;
		}
		ready = false;
		void loadRepeat(repeatId);
	});

	async function loadRepeat(id: string) {
		try {
			repeatFrom = await getTransaction(id);
		} catch {
			repeatFrom = lookupServerTransaction(id);
		} finally {
			ready = true;
		}
	}

	function finish(saved: boolean) {
		if (saved && interceptDraftId) {
			deleteInterceptDraft(interceptDraftId, $user?.id);
		}
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready}
	<TransactionForm
		variant="page"
		backHref={returnTo}
		accountId={createPrefill?.accountId || accountId}
		defaultType={type}
		{repeatFrom}
		{initialDescription}
		{createPrefill}
		onclose={() => finish(false)}
		onsaved={() => finish(true)}
	/>
{/if}
