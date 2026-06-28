<script lang="ts">
	import { _ } from 'svelte-i18n';
	import IconButton from '$lib/components/IconButton.svelte';

	let {
		onincome,
		onexpense,
		ontransfer
	}: {
		onincome: () => void;
		onexpense: () => void;
		ontransfer?: () => void;
	} = $props();
</script>

<IconButton icon="income" label={$_('transactions.type.income')} variant="primary" onclick={onincome} />
<IconButton icon="expense" label={$_('transactions.type.expense')} onclick={onexpense} />
{#if ontransfer}
	<IconButton icon="transfer" label={$_('transactions.transfer')} onclick={ontransfer} />
{/if}
