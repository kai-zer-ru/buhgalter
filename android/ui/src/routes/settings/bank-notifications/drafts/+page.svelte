<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import RowActionsMenu from '$lib/components/RowActionsMenu.svelte';
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
		// Settings are per user: re-read once the user loads (cold start / deep link).
		void $user;
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

{#snippet typeGlyph(kind: 'expense' | 'income' | 'transfer')}
	<span class="draft-glyph draft-{kind}" aria-hidden="true">
		<svg
			class="h-5 w-5"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			{#if kind === 'income'}
				<path d="M12 19V5M5 12l7-7 7 7" />
			{:else if kind === 'transfer'}
				<path d="M4 8h14l-4-4M20 16H6l4 4" />
			{:else}
				<path d="M12 5v14M19 12l-7 7-7-7" />
			{/if}
		</svg>
	</span>
{/snippet}

{#if !enabled}
	<EmptyStateCard message={$_('bankNotifications.drafts.disabled')} />
{:else}
	<div class="space-y-8">
		{#if isNativeApp() && listenerConnected === false}
			<p
				class="rounded-xl border p-3 text-sm"
				style:color="var(--danger)"
				style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
			>
				{$_('bankNotifications.drafts.listenerOff')}
			</p>
		{/if}

		<div class="card flex flex-col gap-3 sm:flex-row sm:items-center">
			<p class="min-w-0 flex-1 text-sm" style:color="var(--text-muted)">
				{$_('bankNotifications.history.scanActiveHint')}
			</p>
			<button
				type="button"
				class="btn-primary shrink-0"
				disabled={scanning}
				onclick={() => void scanActive()}
			>
				{$_('bankNotifications.history.scanActive')}
			</button>
		</div>

		{#if drafts.length === 0}
			<EmptyStateCard message={$_('bankNotifications.drafts.empty')} />
		{:else}
			{#if transferPairs.length > 0}
				<section class="space-y-3">
					<GroupHeading
						title={$_('bankNotifications.drafts.group.pairs', {
							values: { count: transferPairs.length }
						})}
						hint={$_('bankNotifications.drafts.transferHint')}
					/>
					<ul class="space-y-3">
						{#each transferPairs as pair (`${pair.from.id}:${pair.to.id}`)}
							<li class="card space-y-3">
								<div class="flex items-start gap-3">
									{@render typeGlyph('transfer')}
									<div class="min-w-0 flex-1">
										<p class="truncate font-medium">
											{$_('bankNotifications.drafts.fromTo', {
												values: { from: sideLabel(pair.from), to: sideLabel(pair.to) }
											})}
										</p>
										<p class="text-xs" style:color="var(--text-muted)">
											{$_('bankNotifications.drafts.kindTransferPair')} · {formatWhen(
												pair.from.parsed.occurredAt
											)}
										</p>
									</div>
									<span class="shrink-0 font-semibold tabular-nums"
										>{formatMoneyDisplay(pair.from.parsed.amount)} ₽</span
									>
								</div>
								<div class="flex items-center gap-2">
									<button
										type="button"
										class="btn-primary flex-1"
										onclick={() => openPairTransfer(pair)}
									>
										{$_('bankNotifications.drafts.createTransfer')}
									</button>
									<RowActionsMenu
										actions={[
											{
												icon: 'repeat',
												label: $_('bankNotifications.drafts.showSeparate'),
												onclick: () => showAsSeparate(pair)
											},
											{
												icon: 'delete',
												label: $_('bankNotifications.drafts.delete'),
												variant: 'danger',
												onclick: () => removeDrafts([pair.from.id, pair.to.id])
											}
										]}
									/>
								</div>
							</li>
						{/each}
					</ul>
				</section>
			{/if}

			{#if visibleDrafts.length > 0}
				<section class="space-y-3">
					<GroupHeading
						title={$_('bankNotifications.drafts.group.single', {
							values: { count: visibleDrafts.length }
						})}
						hint={$_('bankNotifications.drafts.group.single_hint')}
					/>
					<ul class="space-y-3">
						{#each visibleDrafts as draft (draft.id)}
							{@const kind = draftTxType(draft) === 'income' ? 'income' : 'expense'}
							<li class="card space-y-3">
								<div class="flex items-start gap-3">
									{@render typeGlyph(kind)}
									<div class="min-w-0 flex-1">
										<p class="truncate font-medium">
											{draft.merchantName ||
												draft.parsed.merchantText ||
												$_('bankNotifications.drafts.noMerchant')}
										</p>
										<p class="text-xs" style:color="var(--text-muted)">
											{sideLabel(draft)}{#if draft.parsed.last4}&nbsp;· *{draft.parsed.last4}{/if} · {formatWhen(
												draft.parsed.occurredAt
											)}
										</p>
									</div>
									<span
										class="shrink-0 font-semibold tabular-nums"
										style:color={kind === 'income' ? 'var(--primary)' : undefined}
										>{kind === 'income' ? '+' : '−'}{formatMoneyDisplay(draft.parsed.amount)} ₽</span
									>
								</div>
								<div class="flex items-center gap-2">
									<button type="button" class="btn-primary flex-1" onclick={() => openDraft(draft)}>
										{$_('bankNotifications.drafts.create')}
									</button>
									<RowActionsMenu
										actions={[
											{
												icon: 'transfer',
												label: $_('bankNotifications.drafts.createTransfer'),
												onclick: () => openTransfer(transferPrefillFromDrafts(draft))
											},
											{
												icon: 'delete',
												label: $_('bankNotifications.drafts.delete'),
												variant: 'danger',
												onclick: () => removeDrafts([draft.id])
											}
										]}
									/>
								</div>
							</li>
						{/each}
					</ul>
				</section>
			{/if}
		{/if}
	</div>
{/if}

{#if isNativeApp()}
	<div class="mt-8">
		<button
			type="button"
			class="btn-ghost w-full"
			onclick={() => void goto(resolve('/settings/bank-notifications/history'))}
		>
			{$_('bankNotifications.history.open')}
		</button>
	</div>
{/if}

<style>
	.draft-glyph {
		display: inline-flex;
		height: 2.5rem;
		width: 2.5rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		color: var(--text-muted);
		background-color: color-mix(in srgb, var(--border) 45%, transparent);
	}

	.draft-expense {
		color: var(--danger);
		background-color: color-mix(in srgb, var(--danger) 12%, transparent);
	}

	.draft-income {
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}
</style>
