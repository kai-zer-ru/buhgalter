<script lang="ts">
	import type { IconName } from '$lib/components/IconButton.svelte';

	let {
		icon,
		class: className = 'h-5 w-5'
	}: {
		icon: IconName;
		class?: string;
	} = $props();

	const faAdd = '\uf067';
	const faMinus = '\uf068';
	const faTransfer = '\uf0ec';
</script>

{#if icon === 'edit'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M12 20h9" />
		<path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" />
	</svg>
{:else if icon === 'delete'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M3 6h18" />
		<path d="M8 6V4h8v2" />
		<path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
		<path d="M10 11v6M14 11v6" />
	</svg>
{:else if icon === 'menu'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M4 7h16M4 12h16M4 17h16" />
	</svg>
{:else if icon === 'more-vertical'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<circle cx="12" cy="5" r="1" fill="currentColor" stroke="none" />
		<circle cx="12" cy="12" r="1" fill="currentColor" stroke="none" />
		<circle cx="12" cy="19" r="1" fill="currentColor" stroke="none" />
	</svg>
{:else if icon === 'logout'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
		<path d="M16 17l5-5-5-5M21 12H9" />
	</svg>
{:else if icon === 'add' || icon === 'income'}
	<span class="fa-solid-glyph {className}" aria-hidden="true">{faAdd}</span>
{:else if icon === 'expense'}
	<span class="fa-solid-glyph {className}" aria-hidden="true">{faMinus}</span>
{:else if icon === 'pay'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M3 7h18v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z" />
		<path d="M3 10h18" />
		<path d="M12 13v5" />
		<path d="m9.5 15.5 2.5-2.5 2.5 2.5" />
	</svg>
{:else if icon === 'transfer'}
	<span class="fa-solid-glyph {className}" aria-hidden="true">{faTransfer}</span>
{:else if icon === 'save'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M20 6 9 17l-5-5" />
	</svg>
{:else if icon === 'cancel'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M18 6 6 18M6 6l12 12" />
	</svg>
{:else if icon === 'create'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M12 5v14M5 12h14" />
	</svg>
{:else if icon === 'archive'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M21 8v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8" />
		<path d="M1 3h22v5H1z" />
		<path d="M10 12h4" />
	</svg>
{:else if icon === 'repeat'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M17 1l4 4-4 4" />
		<path d="M3 11V9a4 4 0 0 1 4-4h14" />
		<path d="M7 23l-4-4 4-4" />
		<path d="M21 13v2a4 4 0 0 1-4 4H3" />
	</svg>
{:else if icon === 'bank'}
	<svg
		aria-hidden="true"
		class={className}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<path d="M3 21h18" />
		<path d="M3 10h18" />
		<path d="M12 3 2 10h20L12 3Z" />
		<path d="M7 10v8" />
		<path d="M12 10v8" />
		<path d="M17 10v8" />
	</svg>
{/if}
