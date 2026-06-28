<script lang="ts">
	import EntityLink from '$lib/components/EntityLink.svelte';
	import type { Transaction } from '$lib/api/client';
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

	type AccountRef = { id: string; name: string };

	const route = $derived.by(() => {
		if (tx.type !== 'transfer') {
			const name = tx.account_name ?? '';
			if (!name) return { prefix: '', accounts: [] as AccountRef[] };
			const label = mode === 'plain' ? name : tx.type === 'expense' ? `с ${name}` : `на ${name}`;
			return {
				prefix: mode === 'plain' ? '' : label.slice(0, label.indexOf(name)),
				accounts: [{ id: tx.account_id, name }]
			};
		}

		const names = transferRoute(tx, siblings);
		const { fromAccountId, toAccountId } = transferAccountIds(tx, siblings);
		const accounts: AccountRef[] = [];
		if (names.from && fromAccountId) accounts.push({ id: fromAccountId, name: names.from });
		if (names.to && toAccountId) accounts.push({ id: toAccountId, name: names.to });
		return { prefix: '', accounts };
	});
</script>

{#if route.accounts.length === 0}
	<span>—</span>
{:else if route.accounts.length === 1}
	{#if route.prefix}
		<span>{route.prefix}</span>
	{/if}
	<EntityLink kind="account" id={route.accounts[0].id} label={route.accounts[0].name} />
{:else}
	<EntityLink kind="account" id={route.accounts[0].id} label={route.accounts[0].name} />
	<span> → </span>
	<EntityLink kind="account" id={route.accounts[1].id} label={route.accounts[1].name} />
{/if}
