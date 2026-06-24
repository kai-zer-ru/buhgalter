<script lang="ts">
	import { bankIconUrl } from '$lib/finance';

	let {
		type,
		bankIcon,
		size = 48
	}: {
		type: 'cash' | 'bank';
		bankIcon?: string | null;
		size?: number;
	} = $props();

	const bankSrc = $derived(bankIcon ? bankIconUrl(bankIcon) : null);
</script>

{#if type === 'cash'}
	<svg
		class="shrink-0 rounded-xl"
		width={size}
		height={size}
		viewBox="0 0 48 48"
		role="img"
		aria-label="Наличные"
	>
		<rect width="48" height="48" rx="10" fill="#10B981" />
		<rect
			x="11"
			y="15"
			width="26"
			height="18"
			rx="4"
			fill="none"
			stroke="#fff"
			stroke-width="2.5"
		/>
		<circle cx="24" cy="24" r="3.5" fill="#fff" />
		<path d="M15 19h5M28 19h5" stroke="#fff" stroke-width="2" stroke-linecap="round" />
	</svg>
{:else if bankSrc}
	<img src={bankSrc} alt="" class="shrink-0 rounded-xl object-contain" width={size} height={size} />
{:else}
	<svg class="shrink-0 rounded-xl" width={size} height={size} viewBox="0 0 48 48" role="img">
		<rect width="48" height="48" rx="10" fill="#6B7280" />
		<path
			fill="#fff"
			d="M24 14c-5 0-9 3.5-9 8h4c0-2.8 2.2-5 5-5s5 2.2 5 5c0 3-2 4.5-5 6.5-1 .6-2 1.2-2.5 2H18v3h12v-3h-2.2c.5-1.2 1.8-2 3.2-2.8 3-1.7 5.5-3.5 5.5-7.7 0-4.5-4-8-9-8z"
		/>
	</svg>
{/if}
