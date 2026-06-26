<script lang="ts">
	import { tick } from 'svelte';
	import { formatMoneyInput, formatMoneyLive, mapMoneyInputCursor } from '$lib/money';

	type Props = {
		value?: string;
		id?: string;
		required?: boolean;
		placeholder?: string;
		class?: string;
		autoFocus?: boolean;
	};

	let {
		value = $bindable(''),
		id,
		required = false,
		placeholder = '0.00',
		class: className = 'input w-full tabular-nums',
		autoFocus = false
	}: Props = $props();

	let inputEl = $state<HTMLInputElement | null>(null);
	let prevAutoFocus = $state(false);

	async function onInput(e: Event) {
		const el = e.currentTarget as HTMLInputElement;
		const raw = el.value;
		const cursor = el.selectionStart ?? raw.length;
		const formatted = formatMoneyLive(raw);
		value = formatted;
		const nextCursor = mapMoneyInputCursor(raw, cursor, formatted);
		await tick();
		inputEl?.setSelectionRange(nextCursor, nextCursor);
	}

	$effect(() => {
		if (!autoFocus || prevAutoFocus) {
			prevAutoFocus = autoFocus;
			return;
		}
		prevAutoFocus = autoFocus;
		void tick().then(() => {
			inputEl?.focus();
			inputEl?.select();
		});
	});
</script>

<input
	bind:this={inputEl}
	{id}
	class={className}
	type="text"
	inputmode="decimal"
	{required}
	{placeholder}
	{value}
	oninput={onInput}
	onblur={() => (value = formatMoneyInput(value))}
/>
