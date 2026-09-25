<script lang="ts">
	import { page } from '$app/stores';
	import TransferForm from '$lib/components/TransferForm.svelte';
	import { getAccount, getTransaction, type Account, type Transaction } from '$lib/api/client';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import {
		deleteInterceptDrafts,
		getInterceptDraft,
		setInterceptTransferPrefill,
		syncDraftNotifyToNative,
		takeInterceptTransferPrefill,
		transferPrefillFromDrafts
	} from '$lib/android/notification-intercept';
	import {
		hasTemplatePrefillWarnings,
		loadTemplateRepeatFrom
	} from '$lib/android/template-prefill';
	import { listIndexedTransferLegs, lookupServerTransaction } from '$lib/offline/transaction-index';
	import { user } from '$lib/stores/auth';
	import { toast } from '$lib/toast';
	import { _ } from 'svelte-i18n';

	import { dataRefreshTick } from '$lib/offline/sync';

	const accountId = $derived($page.url.searchParams.get('account') ?? '');
	const payCardId = $derived($page.url.searchParams.get('payCard'));
	const repeatId = $derived($page.url.searchParams.get('repeat'));
	const templateId = $derived($page.url.searchParams.get('template'));
	const returnTo = $derived(parseFormReturnPath($page.url.searchParams.get('from'), '/'));
	const interceptOnce =
		takeInterceptTransferPrefill() ??
		(() => {
			const raw =
				typeof window !== 'undefined'
					? new URLSearchParams(window.location.search).get('intercept_drafts')
					: null;
			if (!raw) return null;
			const ids = raw
				.split(',')
				.map((s) => s.trim())
				.filter(Boolean);
			if (ids.length < 2) return null;
			const a = getInterceptDraft(ids[0]);
			const b = getInterceptDraft(ids[1]);
			if (!a || !b) return null;
			const prefill = transferPrefillFromDrafts(a, b);
			if (prefill) setInterceptTransferPrefill(prefill);
			return prefill;
		})();
	const interceptDraftIds = interceptOnce?.draftIds ?? [];

	let creditCardPay = $state<Account | null>(null);
	let repeatFrom = $state<Transaction | null>(null);
	let siblings = $state<Transaction[]>([]);
	let ready = $state(false);

	$effect(() => {
		const pay = payCardId;
		const tpl = templateId;
		const repeat = repeatId;
		ready = false;
		void init(pay, tpl, repeat);
	});

	async function init(
		payCardId: string | null,
		templateId: string | null,
		repeatId: string | null
	) {
		creditCardPay = null;
		repeatFrom = null;
		siblings = [];

		if (payCardId) {
			try {
				creditCardPay = await getAccount(payCardId);
			} catch {
				creditCardPay = null;
			}
		}

		if (templateId) {
			const result = await loadTemplateRepeatFrom(templateId);
			if (result) {
				repeatFrom = result.tx;
				if (hasTemplatePrefillWarnings(result.warnings)) {
					toast($_('templates.prefill.missing'));
				}
			}
		} else if (repeatId) {
			try {
				repeatFrom = await getTransaction(repeatId);
			} catch {
				repeatFrom = lookupServerTransaction(repeatId);
			}
			if (repeatFrom?.transfer_group_id) {
				siblings = listIndexedTransferLegs(repeatFrom.transfer_group_id);
			}
		}

		ready = true;
	}

	function finish(saved: boolean) {
		if (saved && interceptDraftIds.length) {
			deleteInterceptDrafts(interceptDraftIds, $user?.id);
			void syncDraftNotifyToNative($user?.id);
		}
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready}
	<TransferForm
		variant="page"
		backHref={returnTo}
		{accountId}
		{creditCardPay}
		{repeatFrom}
		{siblings}
		createPrefill={interceptOnce}
		onclose={() => finish(false)}
		onsaved={() => finish(true)}
	/>
{/if}
