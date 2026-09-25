<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import { transactionNewPath, transferNewPath } from '$lib/android/form-routes';
	import { formatMoneyDisplay } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { listAccounts, listBanks, type Account, type Bank } from '$lib/api/client';
	import {
		bankOrWalletLabelKey,
		deleteInterceptDrafts,
		draftTxType,
		getCurrentInterceptSettings,
		getNotificationListenerState,
		interceptDraftsTick,
		listInterceptDrafts,
		listUniqueTransferPairs,
		pairedDraftIds,
		prefillFromDraft,
		processPendingBankNotifications,
		processPendingBankNotificationsDetailed,
		reconnectNotificationListener,
		scanActiveNotifications,
		setInterceptPrefill,
		setInterceptTransferPrefill,
		syncDraftNotifyToNative,
		syncInterceptNativeFromSettings,
		transferPrefillFromDrafts,
		type InterceptDraft,
		type InterceptTransferPair,
		type TransferCreatePrefill
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';

	let scanning = $state(false);
	let listenerConnected = $state<boolean | null>(null);
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let splitPairIds = $state<string[]>([]);

	const drafts = $derived.by(() => {
		void $interceptDraftsTick;
		return listInterceptDrafts($user?.id);
	});
	const enabled = $derived.by(() => {
		void $interceptDraftsTick;
		return getCurrentInterceptSettings().enabled;
	});
	const allPairs = $derived(listUniqueTransferPairs(drafts));
	const splitSet = $derived(new Set(splitPairIds));
	const transferPairs = $derived(
		allPairs.filter((pair) => !splitSet.has(pair.from.id) && !splitSet.has(pair.to.id))
	);
	const hiddenPairedIds = $derived(pairedDraftIds(transferPairs));
	const visibleDrafts = $derived(drafts.filter((d) => !hiddenPairedIds.has(d.id)));

	async function refreshListenerState() {
		if (!isNativeApp()) {
			listenerConnected = null;
			return;
		}
		const st = await getNotificationListenerState();
		listenerConnected = st ? Boolean(st.listenerConnected) : false;
	}

	onMount(() => {
		void loadCatalogs();
		void (async () => {
			if (isNativeApp() && $user?.id) {
				await syncInterceptNativeFromSettings($user.id);
				const re = await reconnectNotificationListener();
				listenerConnected = re.listenerConnected;
			}
			await processPendingBankNotifications();
			await refreshListenerState();
		})();
	});

	async function loadCatalogs() {
		try {
			const [acc, bk] = await Promise.all([listAccounts('active'), listBanks()]);
			accounts = acc;
			banks = bk;
		} catch {
			accounts = [];
			banks = [];
		}
	}

	function sideLabel(draft: InterceptDraft): string {
		const acc = draft.accountId ? accounts.find((a) => a.id === draft.accountId) : undefined;
		if (acc?.name) return acc.name;
		const key = bankOrWalletLabelKey(draft.parsed.bankId);
		if (key) return $_(key);
		return banks.find((b) => b.id === draft.parsed.bankId)?.name || draft.parsed.bankId;
	}

	function openDraft(draft: InterceptDraft) {
		const prefill = prefillFromDraft(draft);
		setInterceptPrefill(prefill);
		void goto(
			resolve(
				transactionNewPath({
					type: prefill.type ?? draftTxType(draft),
					from: '/settings/bank-notifications/drafts'
				})
			)
		);
	}

	function openTransfer(prefill: TransferCreatePrefill | null) {
		if (!prefill) {
			toast.error($_('bankNotifications.drafts.pairInvalid'));
			return;
		}
		setInterceptTransferPrefill(prefill);
		void goto(resolve(transferNewPath({ from: '/settings/bank-notifications/drafts' })));
	}

	function openPairTransfer(pair: InterceptTransferPair) {
		openTransfer(transferPrefillFromDrafts(pair.from, pair.to));
	}

	function removeDrafts(ids: string[]) {
		if (!ids.length) return;
		const n = deleteInterceptDrafts(ids, $user?.id);
		if (n === 0) return;
		splitPairIds = splitPairIds.filter((id) => !ids.includes(id));
		void syncDraftNotifyToNative($user?.id);
		toast(
			n === 1 ? $_('bankNotifications.drafts.deleted') : $_('bankNotifications.drafts.deletedMany')
		);
	}

	function showAsSeparate(pair: InterceptTransferPair) {
		splitPairIds = [...splitPairIds, pair.from.id, pair.to.id];
	}

	function formatWhen(iso: string): string {
		try {
			return new Date(iso).toLocaleString();
		} catch {
			return iso;
		}
	}

	async function scanActive() {
		scanning = true;
		try {
			let result = await scanActiveNotifications();
			if (!result.listenerConnected) {
				const re = await reconnectNotificationListener();
				listenerConnected = re.listenerConnected;
				if (!re.listenerConnected) {
					toast.error($_('bankNotifications.history.scanNoListener'));
					return;
				}
				result = await scanActiveNotifications();
			}
			listenerConnected = result.listenerConnected;
			if (!result.listenerConnected) {
				toast.error($_('bankNotifications.history.scanNoListener'));
				return;
			}
			const { added, cancelled } = await processPendingBankNotificationsDetailed();
			const parts = [$_('bankNotifications.history.scanResult', { values: { n: result.scanned } })];
			if (added > 0) {
				parts.push($_('bankNotifications.drafts.toastNew', { values: { n: added } }));
			}
			if (cancelled > 0) {
				parts.push($_('bankNotifications.drafts.toastCancelled', { values: { n: cancelled } }));
			}
			toast(parts.join(' · '));
		} finally {
			scanning = false;
		}
	}
</script>

{#if !enabled}
	<p class="text-sm" style:color="var(--text-muted)">{$_('bankNotifications.drafts.disabled')}</p>
{:else}
	{#if isNativeApp() && listenerConnected === false}
		<p class="mb-3 text-sm" style:color="var(--danger)">
			{$_('bankNotifications.drafts.listenerOff')}
		</p>
	{/if}
	<div class="mb-4 space-y-2">
		<button
			type="button"
			class="btn-primary w-full"
			disabled={scanning}
			onclick={() => void scanActive()}
		>
			{$_('bankNotifications.history.scanActive')}
		</button>
		<p class="text-xs" style:color="var(--text-muted)">
			{$_('bankNotifications.history.scanActiveHint')}
		</p>
	</div>
	{#if drafts.length === 0}
		<EmptyStateCard message={$_('bankNotifications.drafts.empty')} />
	{:else}
		<p class="mb-3 text-xs" style:color="var(--text-muted)">
			{$_('bankNotifications.drafts.transferHint')}
		</p>
		{#if transferPairs.length > 0}
			<ul class="mb-3 space-y-3">
				{#each transferPairs as pair (`${pair.from.id}:${pair.to.id}`)}
					<li class="card space-y-2">
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('bankNotifications.drafts.kindTransferPair')}
						</p>
						<p class="text-lg font-semibold">{formatMoneyDisplay(pair.from.parsed.amount)} ₽</p>
						<p class="font-medium">
							{$_('bankNotifications.drafts.fromTo', {
								values: { from: sideLabel(pair.from), to: sideLabel(pair.to) }
							})}
						</p>
						<p class="text-sm" style:color="var(--text-muted)">
							{formatWhen(pair.from.parsed.occurredAt)}
						</p>
						<div class="space-y-2">
							<button
								type="button"
								class="btn-primary w-full"
								onclick={() => openPairTransfer(pair)}
							>
								{$_('bankNotifications.drafts.createTransfer')}
							</button>
							<button type="button" class="btn-ghost w-full" onclick={() => showAsSeparate(pair)}>
								{$_('bankNotifications.drafts.showSeparate')}
							</button>
							<button
								type="button"
								class="btn-ghost w-full"
								onclick={() => removeDrafts([pair.from.id, pair.to.id])}
							>
								{$_('bankNotifications.drafts.delete')}
							</button>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
		{#if visibleDrafts.length > 0}
			<ul class="space-y-3">
				{#each visibleDrafts as draft (draft.id)}
					<li class="card space-y-2">
						<div class="min-w-0">
							<p class="text-sm" style:color="var(--text-muted)">
								{draftTxType(draft) === 'income'
									? $_('bankNotifications.drafts.kindIncome')
									: $_('bankNotifications.drafts.kindExpense')}
							</p>
							<p class="text-lg font-semibold">
								{formatMoneyDisplay(draft.parsed.amount)} ₽
							</p>
							<p class="truncate font-medium">
								{draft.merchantName ||
									draft.parsed.merchantText ||
									$_('bankNotifications.drafts.noMerchant')}
							</p>
							<p class="text-sm" style:color="var(--text-muted)">
								{sideLabel(draft)}
								{#if draft.parsed.last4}
									· *{draft.parsed.last4}
								{/if}
								· {formatWhen(draft.parsed.occurredAt)}
							</p>
						</div>
						<div class="space-y-2">
							<div class="btn-pair-row">
								<button type="button" class="btn-primary" onclick={() => openDraft(draft)}>
									{$_('bankNotifications.drafts.create')}
								</button>
								<button
									type="button"
									class="btn-ghost"
									onclick={() => openTransfer(transferPrefillFromDrafts(draft))}
								>
									{$_('bankNotifications.drafts.createTransfer')}
								</button>
							</div>
							<button
								type="button"
								class="btn-ghost w-full"
								onclick={() => removeDrafts([draft.id])}
							>
								{$_('bankNotifications.drafts.delete')}
							</button>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
{/if}

{#if isNativeApp()}
	<div class="mt-4">
		<button
			type="button"
			class="btn-ghost w-full"
			onclick={() => void goto(resolve('/settings/bank-notifications/history'))}
		>
			{$_('bankNotifications.history.open')}
		</button>
	</div>
{/if}
