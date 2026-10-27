<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { CatalogScope } from '$lib/api/client';
	import { householdMe, householdNavLabelKey, inHousehold } from '$lib/household/membership';

	let {
		value,
		onchange,
		mode = 'stats'
	}: {
		value: CatalogScope | 'personal' | 'household';
		onchange: (next: CatalogScope | 'personal' | 'household') => void;
		mode?: 'stats' | 'catalog';
	} = $props();

	const householdTabLabel = $derived(
		$householdMe.household
			? $_(householdNavLabelKey($householdMe))
			: $_('household.scope.household')
	);
	const tabs = $derived.by(() => {
		if (mode === 'catalog') {
			return [
				{ id: 'personal' as const, label: $_('household.scope.personal') },
				{ id: 'household' as const, label: householdTabLabel }
			];
		}
		return [
			{ id: 'all' as const, label: $_('household.scope.all') },
			{ id: 'personal' as const, label: $_('household.scope.personal') },
			{ id: 'household' as const, label: householdTabLabel }
		];
	});
</script>

{#if inHousehold($householdMe)}
	<div class="flex flex-wrap items-center gap-2">
		{#each tabs as tab (tab.id)}
			<button
				type="button"
				class={tab.id === value ? 'tab tab-active' : 'tab'}
				onclick={() => onchange(tab.id)}
			>
				{tab.label}
			</button>
		{/each}
	</div>
{/if}
