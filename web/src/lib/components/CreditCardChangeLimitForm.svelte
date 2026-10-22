<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { changeAccountCreditLimit, type Account } from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { isCreditCardFullyPaid } from '$lib/credit-card';
	import { toAPIAmount, toCents } from '$lib/money';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		open: boolean;
		account: Account;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), account, onclose, onsaved }: Props = $props();

	let newLimit = $state('');
	let saving = $state(false);

	const currency = $derived($user?.currency ?? 'RUB');
	const decreaseBlocked = $derived.by(() => {
		if (!newLimit.trim() || account.credit_limit == null) return false;
		let parsed: number;
		try {
			parsed = toCents(newLimit);
		} catch {
			return false;
		}
		if (parsed <= 0) return false;
		return parsed < account.credit_limit && !isCreditCardFullyPaid(account);
	});

	$effect(() => {
		if (!open) return;
		newLimit = account.credit_limit_display ?? '';
	});

	async function save(e: Event) {
		e.preventDefault();
		if (decreaseBlocked) return;
		saving = true;
		try {
			await changeAccountCreditLimit(account.id, toAPIAmount(newLimit));
			open = false;
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err, 'common.error');
		} finally {
			saving = false;
		}
	}

	function close() {
		open = false;
		onclose();
	}
</script>

<ModalShell bind:open title={$_('accounts.creditCard.changeLimit')} onclose={close}>
	<form id="cc-change-limit-form" class="space-y-4" onsubmit={save}>
		<div>
			<p class="mb-1 text-sm" style:color="var(--text-muted)">
				{$_('accounts.creditCard.currentLimit')}
			</p>
			<p class="text-lg font-semibold tabular-nums">
				{#if account.credit_limit_display}
					<MoneyDisplay value={account.credit_limit_display} {currency} class="" />
				{:else}
					—
				{/if}
			</p>
		</div>
		<div>
			<label class="mb-1 block text-sm font-medium" for="cc-new-limit">
				{$_('accounts.creditCard.newLimit')}
			</label>
			<MoneyInput id="cc-new-limit" bind:value={newLimit} required />
		</div>
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('accounts.creditCard.changeLimitHint')}
		</p>
		{#if decreaseBlocked}
			<p class="text-sm" style:color="var(--danger)">
				{$_('errors.ERR_CREDIT_CARD_LIMIT_DECREASE_NOT_FULLY_PAID')}
			</p>
		{/if}
	</form>

	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={close}>{$_('common.cancel')}</button>
		<button
			type="submit"
			form="cc-change-limit-form"
			class="btn-primary"
			disabled={saving || decreaseBlocked || !newLimit.trim()}
		>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
