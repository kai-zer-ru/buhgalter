<script lang="ts">
	import { resolve } from '$app/paths';

	/** Parent routes used as back-navigation targets. */
	export type BackLinkHref = '/' | '/accounts' | '/settings' | '/admin' | '/debts' | '/credits';

	let {
		href,
		label,
		search = ''
	}: { href: BackLinkHref; label: string; search?: string } = $props();

	const target = $derived(search ? `${resolve(href)}?${search}` : resolve(href));
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- path built with resolve() in derived -->
<a href={target} class="back-link">
	<svg
		class="h-5 w-5 shrink-0"
		viewBox="0 0 20 20"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		aria-hidden="true"
	>
		<path stroke-linecap="round" stroke-linejoin="round" d="M12 4l-6 6 6 6" />
	</svg>
	<span>{label}</span>
</a>
<!-- eslint-enable svelte/no-navigation-without-resolve -->
