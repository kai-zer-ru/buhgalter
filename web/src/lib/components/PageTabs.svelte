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

<div class="page-tabs-scroll">
	<div class="page-tabs-row" role="tablist">
		{#each tabs as tab (tab.id)}
			<button
				type="button"
				class="tab shrink-0 {active === tab.id ? 'tab-active' : ''}"
				role="tab"
				aria-selected={active === tab.id}
				onclick={() => onchange(tab.id)}
			>
				{tab.label}
			</button>
		{/each}
	</div>
</div>
