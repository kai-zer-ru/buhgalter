<script lang="ts">
	type Props = {
		value: number;
		onchange: (value: number) => void;
		id?: string;
		class?: string;
		min?: number;
		max?: number;
		disabled?: boolean;
	};

	let {
		value,
		onchange,
		id,
		class: className = 'input w-full tabular-nums',
		min,
		max,
		disabled = false
	}: Props = $props();

	function digitsOnly(raw: string) {
		return raw.replace(/\D/g, '');
	}

	function displayValue(v: number) {
		return Number.isFinite(v) ? String(v) : '';
	}

	function commit(next: number) {
		onchange(next);
	}

	function onInput(e: Event) {
		if (disabled) return;
		const el = e.currentTarget as HTMLInputElement;
		const digits = digitsOnly(el.value);
		if (el.value !== digits) el.value = digits;
		commit(digits === '' ? NaN : Number.parseInt(digits, 10));
	}

	function onBlur() {
		if (disabled) return;
		if (!Number.isFinite(value)) {
			commit(min ?? 0);
			return;
		}
		if (min !== undefined && value < min) commit(min);
		if (max !== undefined && value > max) commit(max);
	}
</script>

<input
	{id}
	class={className}
	type="text"
	inputmode="numeric"
	autocomplete="off"
	{disabled}
	value={displayValue(value)}
	oninput={onInput}
	onblur={onBlur}
/>
