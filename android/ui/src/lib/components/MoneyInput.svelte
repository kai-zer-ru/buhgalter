<script lang="ts">
	import { tick } from 'svelte';
	import { portal } from '$lib/actions/portal';
	import MoneyKeypad from '$lib/components/MoneyKeypad.svelte';
	import { clearMoneyKeypadInset, setMoneyKeypadInset } from '$lib/money-keypad-inset';
	import { pushModalEscape } from '$lib/modal-escape';
	import {
		applyMoneyKeypadKey,
		formatMoneyInput,
		formatMoneyLive,
		mapMoneyInputCursor,
		type MoneyKeypadKey
	} from '$lib/money';

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
	let keypadHostEl = $state<HTMLDivElement | null>(null);
	let prevAutoFocus = $state(false);
	let keypadOpen = $state(false);
	let cursor = $state(0);
	/** True while interacting with the keypad so blur does not dismiss it. */
	let keypadPointerDown = $state(false);

	async function setFormatted(raw: string, nextCursor: number) {
		const formatted = formatMoneyLive(raw);
		value = formatted;
		cursor = mapMoneyInputCursor(raw, nextCursor, formatted);
		await tick();
		inputEl?.setSelectionRange(cursor, cursor);
	}

	async function onInput(e: Event) {
		const el = e.currentTarget as HTMLInputElement;
		const raw = el.value;
		const sel = el.selectionStart ?? raw.length;
		await setFormatted(raw, sel);
	}

	function syncCursorFromInput() {
		if (!inputEl) return;
		cursor = inputEl.selectionStart ?? value.length;
	}

	function openKeypad() {
		keypadOpen = true;
		syncCursorFromInput();
		void tick().then(() => {
			inputEl?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		});
	}

	function closeKeypad(normalize: boolean) {
		keypadOpen = false;
		clearMoneyKeypadInset();
		if (normalize) {
			value = formatMoneyInput(value);
		}
	}

	function dismissKeypad() {
		keypadPointerDown = false;
		closeKeypad(true);
		inputEl?.blur();
	}

	function onFocus() {
		openKeypad();
	}

	function onBlur() {
		if (keypadPointerDown) {
			void tick().then(() => {
				inputEl?.focus();
				inputEl?.setSelectionRange(cursor, cursor);
			});
			return;
		}
		closeKeypad(true);
	}

	function onKeypadPointerDown(e: PointerEvent) {
		// Keep focus on the input (no system keyboard / no blur dismiss).
		e.preventDefault();
		keypadPointerDown = true;
	}

	function onKeypadPointerUp() {
		queueMicrotask(() => {
			keypadPointerDown = false;
		});
	}

	async function onKeypadKey(key: MoneyKeypadKey) {
		const currentCursor = inputEl?.selectionStart ?? cursor;
		const next = applyMoneyKeypadKey(value, currentCursor, key);
		value = next.value;
		cursor = next.cursor;
		await tick();
		inputEl?.focus();
		inputEl?.setSelectionRange(cursor, cursor);
	}

	function onKeypadDone() {
		dismissKeypad();
	}

	// Hardware back / Escape — dismiss keypad first (same stack as modals).
	$effect(() => {
		if (!keypadOpen) return;
		return pushModalEscape(dismissKeypad);
	});

	// Keep page scroll area above the fixed keypad.
	$effect(() => {
		const host = keypadHostEl;
		if (!keypadOpen || !host) {
			clearMoneyKeypadInset();
			return;
		}
		const sync = () => setMoneyKeypadInset(host.getBoundingClientRect().height);
		sync();
		const ro = new ResizeObserver(sync);
		ro.observe(host);
		return () => {
			ro.disconnect();
			clearMoneyKeypadInset();
		};
	});

	$effect(() => {
		if (!autoFocus || prevAutoFocus) {
			prevAutoFocus = autoFocus;
			return;
		}
		prevAutoFocus = autoFocus;
		void tick().then(() => {
			inputEl?.focus();
			inputEl?.select();
			cursor = inputEl?.selectionStart ?? value.length;
			openKeypad();
		});
	});
</script>

<input
	bind:this={inputEl}
	{id}
	class={className}
	type="text"
	inputmode="none"
	autocomplete="off"
	{required}
	{placeholder}
	{value}
	oninput={onInput}
	onfocus={onFocus}
	onblur={onBlur}
	onmouseup={syncCursorFromInput}
	onkeyup={syncCursorFromInput}
	onclick={syncCursorFromInput}
/>

{#if keypadOpen}
	<div
		bind:this={keypadHostEl}
		class="money-keypad-host"
		use:portal={typeof document !== 'undefined' ? document.body : null}
		onpointerdown={onKeypadPointerDown}
		onpointerup={onKeypadPointerUp}
		onpointercancel={onKeypadPointerUp}
	>
		<MoneyKeypad onkey={onKeypadKey} ondone={onKeypadDone} />
	</div>
{/if}

<style>
	.money-keypad-host {
		position: fixed;
		inset-inline: 0;
		bottom: 0;
		z-index: 55;
	}
</style>
