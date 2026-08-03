<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { changeAccountCreditLimit, type Account } from '$lib/api/client';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { isCreditCardFullyPaid } from '$lib/credit-card';
	import { toAPIAmount, toCents } from '$lib/money';
	import { requireOnline } from '$lib/offline/require-online';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	type Props = {
		backHref?: string;
		account: Account;
		onclose: () => void;
		onsaved: () => void;
	};

	let { backHref = '/accounts', account, onclose, onsaved }: Props = $props();

	let newLimit = $state(account.credit_limit_display ?? '');
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

	async function save(e: Event) {
		e.preventDefault();
		if (decreaseBlocked) return;
		if (!requireOnline('offline.onlineOnly.creditLimit')) return;
		saving = true;
		try {
			await changeAccountCreditLimit(account.id, toAPIAmount(newLimit));
			toast($_('common.saved'));
			onsaved();
		} catch (err) {
			toast.fromError(err, 'common.error');
		} finally {
			saving = false;
		}
	}
</script>

<FormPageShell title={$_('accounts.creditCard.changeLimit')} {backHref} onback={onclose}>
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
		<button type="button" class="btn-ghost" onclick={onclose}>{$_('common.cancel')}</button>
		<button
			type="submit"
			form="cc-change-limit-form"
			class="btn-primary"
			disabled={saving || decreaseBlocked || !newLimit.trim()}
		>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</FormPageShell>
