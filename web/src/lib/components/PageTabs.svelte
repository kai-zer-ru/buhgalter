<script lang="ts">
	type Tab = {
		id: string;
		label: string;
	};

	let {
		tabs,
		active,
		onchange
	}: {
		tabs: Tab[];
		active: string;
		onchange: (id: string) => void;
	} = $props();
</script>

<div class="flex flex-wrap gap-2">
	{#each tabs as tab (tab.id)}
		<button
			type="button"
			class="tab {active === tab.id ? 'tab-active' : ''}"
			aria-current={active === tab.id ? 'page' : undefined}
			onclick={() => onchange(tab.id)}
		>
			{tab.label}
		</button>
	{/each}
</div>
