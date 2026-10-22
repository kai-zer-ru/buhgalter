<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';

	let { children }: { children: Snippet } = $props();

	let panelEl: HTMLDetailsElement | undefined = $state();
	let desktop = $state(true);

	onMount(() => {
		const mq = window.matchMedia('(min-width: 768px)');
		desktop = mq.matches;
		if (panelEl) panelEl.open = mq.matches;

		const onChange = (event: MediaQueryListEvent) => {
			desktop = event.matches;
			if (panelEl) panelEl.open = event.matches ? true : panelEl.open;
		};
		mq.addEventListener('change', onChange);
		return () => mq.removeEventListener('change', onChange);
	});

	function onToggle() {
		if (desktop && panelEl) panelEl.open = true;
	}
</script>

<details class="filter-panel card" open bind:this={panelEl} ontoggle={onToggle}>
	<summary class="filter-panel-summary md:hidden">
		<span>{$_('transactions.filters.toggle')}</span>
		<svg
			class="filter-panel-chevron"
			aria-hidden="true"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<path d="m6 9 6 6 6-6" />
		</svg>
	</summary>
	{@render children()}
</details>
