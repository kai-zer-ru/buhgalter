<script lang="ts">
	import { formatMoneyInput, formatMoneyLive } from '$lib/money';

	type Props = {
		value?: string;
		id?: string;
		required?: boolean;
		placeholder?: string;
		class?: string;
	};

	let {
		value = $bindable(''),
		id,
		required = false,
		placeholder = '0.00',
		class: className = 'input w-full tabular-nums'
	}: Props = $props();
</script>

<input
	{id}
	class={className}
	type="text"
	inputmode="decimal"
	{required}
	{placeholder}
	{value}
	oninput={(e) => (value = formatMoneyLive(e.currentTarget.value))}
	onblur={() => (value = formatMoneyInput(value))}
/>
