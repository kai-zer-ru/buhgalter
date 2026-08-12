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
	import {
		deleteInterceptDraft,
		draftTxType,
		getCurrentInterceptSettings,
		getNotificationListenerState,
		interceptCreateRoute,
		interceptDraftsTick,
		listInterceptDrafts,
		prefillFromDraftWithSuggestions,
		processPendingBankNotifications,
		processPendingBankNotificationsDetailed,
		reconnectNotificationListener,
		scanActiveNotifications,
		setInterceptPrefill,
		syncInterceptNativeFromSettings,
		type InterceptDraft
	} from '$lib/android/notification-intercept';
	import { isNativeApp } from '$lib/platform/native';

	let drafts = $state<InterceptDraft[]>([]);
	let enabled = $state(false);
	let scanning = $state(false);
	let listenerConnected = $state<boolean | null>(null);

	function refresh() {
		void $interceptDraftsTick;
		drafts = listInterceptDrafts($user?.id);
		enabled = getCurrentInterceptSettings().enabled;
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
			await processPendingBankNotifications();
			refresh();
			await refreshListenerState();
		})();
	});

	async function openDraft(draft: InterceptDraft) {
		const prefill = await prefillFromDraftWithSuggestions(draft);
		setInterceptPrefill(prefill);
		void goto(resolveAppPath(interceptCreateRoute(prefill.type ?? draftTxType(draft))));
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
		<ul class="space-y-3">
			{#each drafts as draft (draft.id)}
				<li class="card space-y-2">
					<div class="flex items-start justify-between gap-3">
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
								{#if draft.parsed.last4}
									*{draft.parsed.last4} ·
								{/if}
								{formatWhen(draft.parsed.occurredAt)}
							</p>
						</div>
					</div>
					<div class="flex flex-wrap gap-2">
						<button type="button" class="btn" onclick={() => openDraft(draft)}>
							{$_('bankNotifications.drafts.create')}
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
