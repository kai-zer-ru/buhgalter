<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { accountGroupLabelKey, type AccountGroupKind } from '$lib/accounts/group-by-type';

	let {
		kind,
		count,
		children
	}: {
		kind: AccountGroupKind;
		count: number;
		children: Snippet;
	} = $props();

	let panelEl: HTMLDetailsElement | undefined = $state();

	onMount(() => {
		const mq = window.matchMedia('(min-width: 640px)');
		if (panelEl) panelEl.open = mq.matches;

		const onChange = (event: MediaQueryListEvent) => {
			if (panelEl && event.matches) panelEl.open = true;
		};
		mq.addEventListener('change', onChange);
		return () => mq.removeEventListener('change', onChange);
	});
</script>

<details class="account-group-panel" bind:this={panelEl}>
	<summary class="account-group-summary">
		<span class="text-lg font-medium">
			{$_(accountGroupLabelKey(kind))}
			<span class="font-normal tabular-nums" style:color="var(--text-muted)">({count})</span>
		</span>
		<svg
			class="account-group-chevron"
			aria-hidden="true"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<path d="m6 9 6 6 6-6" />
		</svg>
	</summary>
	<div class="pt-3 sm:pt-4">
		{@render children()}
	</div>
</details>
