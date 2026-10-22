<script lang="ts">
	import { page } from '$app/stores';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import type { AndroidNavItem } from '$lib/android/nav-items';

	let {
		items,
		ariaLabelKey = 'nav.menu'
	}: {
		items: AndroidNavItem[];
		ariaLabelKey?: string;
	} = $props();

	const path = $derived($page.url.pathname);
</script>

<nav class="section-nav-hub card" aria-label={$_(ariaLabelKey)}>
	{#each items as item (item.href)}
		<a
			href={resolve(item.href as '/')}
			class="section-nav-hub-link"
			class:active={item.isActive(path)}
			aria-current={item.isActive(path) ? 'page' : undefined}
		>
			<span class="flex flex-1 items-center justify-between gap-2">
				<span>{$_(item.labelKey)}</span>
				<svg
					aria-hidden="true"
					class="h-4 w-4 shrink-0 opacity-70"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
				>
					<path d="m9 6 6 6-6 6" />
				</svg>
			</span>
		</a>
	{/each}
</nav>
