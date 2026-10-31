<script lang="ts">
	import { _ } from 'svelte-i18n';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import type { StatsNetWorthSnapshot } from '$lib/api/client';

	let {
		current,
		items = [],
		currency,
		periodLabel
	}: {
		current: StatsNetWorthSnapshot;
		items?: StatsNetWorthSnapshot[];
		currency: string;
		periodLabel: (period: string) => string;
	} = $props();
</script>

<div class="card space-y-4">
	<div>
		<h2 class="text-base font-semibold">{$_('stats.netWorth.title')}</h2>
		<p class="text-2xl font-semibold tabular-nums">
			<MoneyDisplay cents={current.net_worth} {currency} class="" />
		</p>
	</div>
	<div class="grid gap-3 sm:grid-cols-2">
		<div>
			<p class="text-xs" style:color="var(--text-muted)">{$_('stats.netWorth.assets')}</p>
			<p class="font-medium tabular-nums">
				<MoneyDisplay cents={current.assets} {currency} class="" />
			</p>
			<ul class="mt-2 space-y-1 text-sm" style:color="var(--text-muted)">
				<li class="flex justify-between gap-2">
					<span>{$_('stats.netWorth.cashBank')}</span>
					<span class="tabular-nums"
						><MoneyDisplay cents={current.cash_bank} {currency} class="" /></span
					>
				</li>
				<li class="flex justify-between gap-2">
					<span>{$_('stats.netWorth.owedToMe')}</span>
					<span class="tabular-nums"
						><MoneyDisplay cents={current.owed_to_me} {currency} class="" /></span
					>
				</li>
			</ul>
		</div>
		<div>
			<p class="text-xs" style:color="var(--text-muted)">{$_('stats.netWorth.liabilities')}</p>
			<p class="font-medium tabular-nums">
				<MoneyDisplay cents={current.liabilities} {currency} class="" />
			</p>
			<ul class="mt-2 space-y-1 text-sm" style:color="var(--text-muted)">
				<li class="flex justify-between gap-2">
					<span>{$_('stats.netWorth.iOwe')}</span>
					<span class="tabular-nums"
						><MoneyDisplay cents={current.i_owe} {currency} class="" /></span
					>
				</li>
				<li class="flex justify-between gap-2">
					<span>{$_('stats.netWorth.credits')}</span>
					<span class="tabular-nums"
						><MoneyDisplay cents={current.credits_remaining} {currency} class="" /></span
					>
				</li>
				<li class="flex justify-between gap-2">
					<span>{$_('stats.netWorth.creditCards')}</span>
					<span class="tabular-nums"
						><MoneyDisplay cents={current.credit_cards_debt} {currency} class="" /></span
					>
				</li>
			</ul>
		</div>
	</div>
	{#if items.length > 0}
		<details class="text-sm">
			<summary class="cursor-pointer font-medium">{$_('stats.netWorth.history')}</summary>
			<ul class="mt-2 divide-y divide-[var(--border-color)]">
				{#each items as row (row.period)}
					<li class="flex items-center justify-between gap-2 py-2">
						<span>{periodLabel(row.period ?? '')}</span>
						<span class="tabular-nums font-medium"
							><MoneyDisplay cents={row.net_worth} {currency} class="" /></span
						>
					</li>
				{/each}
			</ul>
		</details>
	{/if}
</div>
