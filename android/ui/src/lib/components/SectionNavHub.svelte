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

<nav class="card list-divided p-0 sm:p-0" aria-label={$_(ariaLabelKey)}>
	{#each items as item (item.href)}
		<a
			href={resolve(item.href as '/')}
			class="section-nav-hub-link"
			class:active={item.isActive(path)}
			aria-current={item.isActive(path) ? 'page' : undefined}
		>
			{#if item.iconPath}
				<span class="section-nav-hub-glyph" aria-hidden="true">
					<svg
						class="h-5 w-5"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d={item.iconPath} />
					</svg>
				</span>
			{/if}
			<span class="min-w-0 flex-1">
				<span class="block font-medium">{$_(item.labelKey)}</span>
				{#if item.hintKey}
					<span class="block text-xs leading-snug" style:color="var(--text-muted)"
						>{$_(item.hintKey)}</span
					>
				{/if}
			</span>
			<svg
				aria-hidden="true"
				class="h-4 w-4 shrink-0 opacity-50"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
			>
				<path d="m9 6 6 6-6 6" />
			</svg>
		</a>
	{/each}
</nav>
