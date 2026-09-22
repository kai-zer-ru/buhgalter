<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import { resolveAppPath } from '$lib/android/form-nav';
	import { formatMoneyDisplay } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';
	import { listAccounts, listBanks, type Account, type Bank } from '$lib/api/client';
	import {
		bankOrWalletLabelKey,
		deleteInterceptDraft,
		draftTxType,
		findUniqueTransferComplement,
		getCurrentInterceptSettings,
		getNotificationListenerState,
		interceptCreateRoute,
		interceptDraftsTick,
		interceptTransferRoute,
		listInterceptDrafts,
		listUniqueTransferPairs,
		prefillFromDraftWithSuggestions,
		processPendingBankNotifications,
		processPendingBankNotificationsDetailed,
		reconnectNotificationListener,
		scanActiveNotifications,
		setInterceptPrefill,
		setInterceptTransferPrefill,
		syncInterceptNativeFromSettings,
		transferPrefillFromDrafts,
		transferPrefillFromSelection,
		type InterceptDraft,
		type InterceptTransferPair,
		type TransferCreatePrefill
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';

	let drafts = $state<InterceptDraft[]>([]);
	let enabled = $state(false);
	let scanning = $state(false);
	let listenerConnected = $state<boolean | null>(null);
	let accounts = $state<Account[]>([]);
	let banks = $state<Bank[]>([]);
	let selectedIds = $state<string[]>([]);

	const transferPairs = $derived(listUniqueTransferPairs(drafts));
	const selectedDrafts = $derived(drafts.filter((d) => selectedIds.includes(d.id)));

	function refresh() {
		void $interceptDraftsTick;
		drafts = listInterceptDrafts($user?.id);
		enabled = getCurrentInterceptSettings().enabled;
		const alive = new Set(drafts.map((d) => d.id));
		const nextSelected = selectedIds.filter((id) => alive.has(id));
		if (nextSelected.length !== selectedIds.length) {
			selectedIds = nextSelected;
		}
	}

	async function refreshListenerState() {
		if (!isNativeApp()) {
			listenerConnected = null;
			return;
		}
		const st = await getNotificationListenerState();
		listenerConnected = st ? Boolean(st.listenerConnected) : false;
	}

	$effect(() => {
		void $interceptDraftsTick;
		void $user?.id;
		refresh();
	});

	onMount(() => {
		void (async () => {
			if (isNativeApp() && $user?.id) {
				await syncInterceptNativeFromSettings($user.id);
				const re = await reconnectNotificationListener();
				listenerConnected = re.listenerConnected;
			}
			try {
				accounts = await listAccounts('active');
				banks = await listBanks();
			} catch {
				accounts = [];
				banks = [];
			}
			await processPendingBankNotifications();
			refresh();
			await refreshListenerState();
		})();
	});

	function sideLabel(draft: InterceptDraft): string {
		const acc = draft.accountId ? accounts.find((a) => a.id === draft.accountId) : undefined;
		if (acc?.name) return acc.name;
		const key = bankOrWalletLabelKey(draft.parsed.bankId);
		if (key) return $_(key);
		return banks.find((b) => b.id === draft.parsed.bankId)?.name || draft.parsed.bankId;
	}

	async function openDraft(draft: InterceptDraft) {
		const prefill = await prefillFromDraftWithSuggestions(draft);
		setInterceptPrefill(prefill);
		void goto(resolveAppPath(interceptCreateRoute(prefill.type ?? draftTxType(draft))));
	}

	function openTransfer(prefill: TransferCreatePrefill | null) {
		if (!prefill) {
			toast.error($_('bankNotifications.drafts.pairInvalid'));
			return;
		}
		setInterceptTransferPrefill(prefill);
		void goto(resolveAppPath(interceptTransferRoute()));
	}

	function openDraftTransfer(draft: InterceptDraft) {
		const complement = findUniqueTransferComplement(draft, drafts);
		openTransfer(transferPrefillFromDrafts(draft, complement));
	}

	function openPairTransfer(pair: InterceptTransferPair) {
		openTransfer(transferPrefillFromDrafts(pair.from, pair.to));
	}

	function openSelectedTransfer() {
		openTransfer(transferPrefillFromSelection(selectedDrafts));
	}

	function toggleSelected(draftId: string) {
		if (selectedIds.includes(draftId)) {
			selectedIds = selectedIds.filter((id) => id !== draftId);
			return;
		}
		selectedIds = [...selectedIds, draftId];
	}

	function removeDraft(draft: InterceptDraft) {
		deleteInterceptDraft(draft.id, $user?.id);
		refresh();
		toast($_('bankNotifications.drafts.deleted'));
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
			refresh();
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
		<button type="button" class="btn w-full" disabled={scanning} onclick={() => void scanActive()}>
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
		{#if selectedIds.length > 0}
			<div class="card mb-3 flex items-center justify-between gap-3">
				<p class="text-sm">
					{$_('bankNotifications.drafts.selected', { values: { n: selectedIds.length } })}
				</p>
				<button type="button" class="btn" onclick={openSelectedTransfer}>
					{$_('bankNotifications.drafts.createTransfer')}
				</button>
			</div>
		{/if}
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
						<button type="button" class="btn" onclick={() => openPairTransfer(pair)}>
							{$_('bankNotifications.drafts.createTransfer')}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		<ul class="space-y-3">
			{#each drafts as draft (draft.id)}
				{@const complement = findUniqueTransferComplement(draft, drafts)}
				<li class="card space-y-2">
					<label class="flex items-start gap-3">
						<input
							type="checkbox"
							class="mt-1.5"
							checked={selectedIds.includes(draft.id)}
							onchange={() => toggleSelected(draft.id)}
						/>
						<div class="min-w-0 flex-1">
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
							{#if complement}
								<p class="text-xs" style:color="var(--text-muted)">
									{$_('bankNotifications.drafts.pairHint')}
								</p>
							{/if}
						</div>
					</label>
					<div class="flex flex-wrap gap-2">
						<button type="button" class="btn" onclick={() => openDraft(draft)}>
							{$_('bankNotifications.drafts.create')}
						</button>
						<button type="button" class="btn" onclick={() => openDraftTransfer(draft)}>
							{$_('bankNotifications.drafts.createTransfer')}
						</button>
						<button type="button" class="btn-ghost" onclick={() => removeDraft(draft)}>
							{$_('bankNotifications.drafts.delete')}
						</button>
					</div>
				</li>
			{/each}
		</ul>
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
