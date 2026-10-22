<script lang="ts">
	import { formatMoneyForDisplay } from '$lib/money-display';

	type Props = {
		value?: string;
		cents?: number;
		currency?: string;
		class?: string;
	};

	let { value, cents, currency, class: className = 'tabular-nums' }: Props = $props();

	const display = $derived(formatMoneyForDisplay({ value, cents, currency }));
</script>

<span class={className}>{display}</span>
