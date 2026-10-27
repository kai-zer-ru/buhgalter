<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { householdKindKey, householdMe } from '$lib/household/membership';

	let { compact = false }: { compact?: boolean } = $props();

	const label = $derived($_(householdKindKey('household.badge', $householdMe.household?.kind)));
</script>

<span
	class="inline-flex items-center gap-1 rounded-full font-medium {compact
		? 'px-1.5 py-0 text-[10px] leading-4'
		: 'px-2 py-0.5 text-xs'}"
	style:background-color="color-mix(in srgb, var(--text-muted) 16%, transparent)"
	style:color="var(--text-muted)"
	title={label}
	data-testid="family-badge"
>
	<svg
		aria-hidden="true"
		class={compact ? 'h-3 w-3' : 'h-3.5 w-3.5'}
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<circle cx="9" cy="8" r="3" />
		<circle cx="16" cy="9" r="2.5" />
		<path d="M3.5 19c.6-3 2.8-5 5.5-5s4.9 2 5.5 5" />
		<path d="M14 14.2c1.9.3 3.6 1.7 4.5 4.3" />
	</svg>
	<span>{label}</span>
</span>
