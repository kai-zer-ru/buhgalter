<script lang="ts">
	import { resolve } from '$app/paths';

	type EntityKind = 'account' | 'credit' | 'debtor';

	let {
		kind,
		id,
		label,
		class: className = ''
	}: {
		kind: EntityKind;
		id: string;
		label: string;
		class?: string;
	} = $props();

	const href = $derived(kind === 'account' ? resolve(`/accounts/${id}`) : null);
</script>

{#if href}
	<a {href} class="entity-link hover:underline {className}">{label}</a>
{:else}
	<span class={className}>{label}</span>
{/if}

<style>
	.entity-link {
		color: var(--primary);
	}
</style>
