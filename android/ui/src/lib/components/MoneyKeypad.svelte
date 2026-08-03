<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { MoneyKeypadKey } from '$lib/money';

	type Props = {
		ondone: () => void;
		onkey: (key: MoneyKeypadKey) => void;
	};

	let { ondone, onkey }: Props = $props();

	const keys: Array<{ id: string; key?: MoneyKeypadKey; label: string; aria?: string; done?: boolean }> =
		[
			{ id: '1', key: '1', label: '1' },
			{ id: '2', key: '2', label: '2' },
			{ id: '3', key: '3', label: '3' },
			{ id: 'plus', key: '+', label: '+', aria: 'moneyInput.plus' },
			{ id: '4', key: '4', label: '4' },
			{ id: '5', key: '5', label: '5' },
			{ id: '6', key: '6', label: '6' },
			{ id: 'minus', key: '-', label: '−', aria: 'moneyInput.minus' },
			{ id: '7', key: '7', label: '7' },
			{ id: '8', key: '8', label: '8' },
			{ id: '9', key: '9', label: '9' },
			{ id: 'dot', key: '.', label: '.', aria: 'moneyInput.decimal' },
			{ id: 'back', key: 'backspace', label: '⌫', aria: 'moneyInput.backspace' },
			{ id: '0', key: '0', label: '0' },
			{ id: 'done', label: '✓', aria: 'moneyInput.done', done: true }
		];
</script>

<div class="money-keypad" role="group" aria-label={$_('moneyInput.keypad')}>
	{#each keys as item (item.id)}
		{#if item.done}
			<button
				type="button"
				class="money-keypad-key money-keypad-done"
				aria-label={item.aria ? $_(item.aria) : undefined}
				onclick={ondone}
			>
				{item.label}
			</button>
		{:else if item.key}
			<button
				type="button"
				class="money-keypad-key"
				class:money-keypad-op={item.key === '+' || item.key === '-'}
				aria-label={item.aria ? $_(item.aria) : undefined}
				onclick={() => onkey(item.key!)}
			>
				{item.label}
			</button>
		{/if}
	{/each}
</div>

<style>
	.money-keypad {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 0.5rem;
		padding: 0.75rem 1rem;
		padding-bottom: max(0.75rem, env(safe-area-inset-bottom));
		border-top: 1px solid var(--border);
		background-color: var(--bg-elevated);
	}

	.money-keypad-key {
		display: flex;
		min-height: 3rem;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		border: 1px solid var(--border);
		background-color: var(--bg);
		color: var(--text);
		font-size: 1.25rem;
		font-weight: 500;
		font-variant-numeric: tabular-nums;
		cursor: pointer;
		user-select: none;
		-webkit-tap-highlight-color: transparent;
	}

	.money-keypad-key:active {
		background-color: color-mix(in srgb, var(--border) 55%, var(--bg));
	}

	.money-keypad-op {
		color: var(--primary);
		font-weight: 600;
	}

	.money-keypad-done {
		grid-column: span 2;
		background-color: var(--primary);
		border-color: color-mix(in srgb, var(--primary) 80%, var(--border));
		color: #fff;
		font-weight: 600;
	}

	.money-keypad-done:active {
		filter: brightness(1.08);
		background-color: var(--primary);
	}
</style>
