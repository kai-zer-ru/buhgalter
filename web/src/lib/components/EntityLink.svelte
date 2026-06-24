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

	const href = $derived(
		kind === 'account'
			? resolve(`/accounts/${id}`)
			: kind === 'credit'
				? resolve(`/credits/${id}`)
				: resolve(`/debtors/${id}`)
	);
</script>

<a {href} class="entity-link hover:underline {className}">{label}</a>

<style>
	.entity-link {
		color: var(--primary);
	}
</style>
