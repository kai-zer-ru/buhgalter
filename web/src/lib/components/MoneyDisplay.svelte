<script lang="ts">
	import { currencyCatalogVersion } from '$lib/finance';
	import { formatMoneyForDisplay } from '$lib/money-display';

	type Props = {
		value?: string;
		cents?: number;
		currency?: string;
		class?: string;
	};

	let { value, cents, currency, class: className = 'tabular-nums' }: Props = $props();

	const display = $derived.by(() => {
		void $currencyCatalogVersion;
		return formatMoneyForDisplay({ value, cents, currency });
	});
</script>

<span class={className}>{display}</span>
