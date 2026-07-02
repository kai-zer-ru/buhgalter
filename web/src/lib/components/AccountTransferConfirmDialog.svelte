<script lang="ts">
	import { onDestroy } from 'svelte';
	import { locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import Select from '$lib/components/Select.svelte';
	import { user } from '$lib/stores/auth';
	import {
		accountTransferConfirmStore,
		resolveAccountTransferConfirm,
		setAccountTransferTarget,
		type AccountTransferConfirmState
	} from '$lib/accounts/account-transfer-confirm';
	import { pushModalEscape } from '$lib/modal-escape';

	let state = $state<AccountTransferConfirmState>({
		open: false,
		options: {
			message: '',
			needsTransfer: false,
			balanceMessageBefore: '',
			balanceMessageAfter: '',
			balanceDisplay: '',
			transferLabel: '',
			noTargetsMessage: '',
			transferOptions: []
		},
		transferToAccountId: ''
	});

	const unsubscribe = accountTransferConfirmStore.subscribe((value) => {
		state = value;
	});

	onDestroy(unsubscribe);

	const needsTransfer = $derived(state.options.needsTransfer);
	const canConfirm = $derived(
		!needsTransfer ||
			(state.options.transferOptions.length > 0 && state.transferToAccountId.trim() !== '')
	);

	$effect(() => {
		if (!state.open) return;
		return pushModalEscape(() => resolveAccountTransferConfirm(false));
	});

	function onBackdropClick() {
		resolveAccountTransferConfirm(false);
	}

	function onConfirm() {
		if (!canConfirm) return;
		resolveAccountTransferConfirm(true, state.transferToAccountId);
	}

	const title = $derived.by(() => {
		void $locale;
		return tr('common.confirm.title');
	});
	const confirmLabel = $derived.by(() => {
		void $locale;
		return state.options.confirmLabel ?? tr('common.confirm.confirm');
	});
	const cancelLabel = $derived.by(() => {
		void $locale;
		return state.options.cancelLabel ?? tr('common.cancel');
	});
	const currency = $derived($user?.currency ?? 'RUB');
</script>

{#if state.open}
	<div
		class="modal-backdrop fixed inset-0 z-[60] flex items-end justify-center sm:items-center sm:p-4"
		style:background-color="color-mix(in srgb, #000 55%, transparent)"
		role="presentation"
		onclick={onBackdropClick}
	>
		<div
			class="modal-panel card w-full max-w-md shadow-xl sm:rounded-2xl"
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="account-transfer-dialog-title"
			aria-describedby="account-transfer-dialog-message"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<div class="p-4 sm:p-6">
				<h2 id="account-transfer-dialog-title" class="text-lg font-semibold">{title}</h2>
				<p
					id="account-transfer-dialog-message"
					class="mt-2 text-sm"
					style:color="var(--text-muted)"
				>
					{state.options.message}
				</p>
				{#if needsTransfer}
					<p class="mt-3 text-sm">
						{state.options.balanceMessageBefore}<MoneyDisplay
							value={state.options.balanceDisplay}
							{currency}
							class="tabular-nums font-medium"
						/>{state.options.balanceMessageAfter}
					</p>
					<div class="mt-3">
						<Select
							id="account-transfer-to"
							label={state.options.transferLabel}
							options={state.options.transferOptions}
							value={state.transferToAccountId}
							usePortal
							onchange={(next) => setAccountTransferTarget(next)}
						/>
					</div>
				{:else if state.options.noTargetsMessage}
					<p class="mt-3 text-sm" style:color="var(--danger)">{state.options.noTargetsMessage}</p>
				{/if}
				<div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:flex-wrap sm:justify-end">
					<button
						type="button"
						class="btn-ghost w-full sm:w-auto"
						onclick={() => resolveAccountTransferConfirm(false)}
					>
						{cancelLabel}
					</button>
					<button
						type="button"
						class="{state.options.danger ? 'btn-danger' : 'btn-primary'} w-full sm:w-auto"
						disabled={!canConfirm}
						onclick={onConfirm}
					>
						{confirmLabel}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
