<script lang="ts">
	import EntityLink from '$lib/components/EntityLink.svelte';
	import type { Transaction } from '$lib/api/client';
	import { _ } from 'svelte-i18n';
	import {
		transferAccountIds,
		transferRoute,
		type AccountLabelMode
	} from '$lib/transaction-display';

	let {
		tx,
		siblings = [],
		mode = 'plain'
	}: {
		tx: Transaction;
		siblings?: Transaction[];
		mode?: AccountLabelMode;
	} = $props();

	type AccountRef = { id: string; name: string; status?: string; masked?: boolean };

	function statusForAccountId(accountId: string): string | undefined {
		if (accountId === tx.account_id) return tx.account_status;
		if (accountId === tx.transfer_account_id) return tx.transfer_account_status;
		return undefined;
	}

	const route = $derived.by(() => {
		if (tx.type !== 'transfer') {
			const name = tx.account_name ?? '';
			if (!name) return { prefix: '', accounts: [] as AccountRef[] };
			const label = mode === 'plain' ? name : tx.type === 'expense' ? `с ${name}` : `на ${name}`;
			return {
				prefix: mode === 'plain' ? '' : label.slice(0, label.indexOf(name)),
				accounts: [{ id: tx.account_id, name, status: tx.account_status }]
			};
		}

		const names = transferRoute(tx, siblings);
		const { fromAccountId, toAccountId } = transferAccountIds(tx, siblings);
		const maskTo = Boolean(tx.transfer_account_masked && tx.transfer_is_out);
		const maskFrom = Boolean(tx.transfer_account_masked && !tx.transfer_is_out);
		const accounts: AccountRef[] = [];
		if (names.from && (fromAccountId || maskFrom)) {
			accounts.push({
				id: fromAccountId,
				name: names.from,
				status: statusForAccountId(fromAccountId),
				masked: maskFrom
			});
		}
		if (names.to && (toAccountId || maskTo)) {
			accounts.push({
				id: toAccountId,
				name: names.to,
				status: statusForAccountId(toAccountId),
				masked: maskTo
			});
		}
		return { prefix: '', accounts };
	});
</script>

{#snippet accountLink(ref: AccountRef)}
	{#if ref.masked || !ref.id}
		<span>{ref.name}</span>
	{:else}
		<EntityLink kind="account" id={ref.id} label={ref.name} />
	{/if}
	{#if ref.status === 'archived'}
		<span class="text-xs" style:color="var(--text-muted)"> ({$_('accounts.status.archived')})</span>
	{:else if ref.status === 'deleted'}
		<span class="text-xs" style:color="var(--text-muted)"> ({$_('accounts.status.deleted')})</span>
	{/if}
{/snippet}

{#if route.accounts.length === 0}
	<span>—</span>
{:else if route.accounts.length === 1}
	{#if route.prefix}
		<span>{route.prefix}</span>
	{/if}
	{@render accountLink(route.accounts[0])}
{:else}
	{@render accountLink(route.accounts[0])}
	<span> → </span>
	{@render accountLink(route.accounts[1])}
{/if}
