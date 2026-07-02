<script lang="ts">
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import type { Debt } from '$lib/api/client';
	import EntityLink from '$lib/components/EntityLink.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import { formatAPIDateForDisplay, formatAPIOperationDateTimeForDisplay } from '$lib/dates';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';

	let {
		debts,
		tz,
		currency,
		showDebtor = true,
		onsettle,
		ondelete
	}: {
		debts: Debt[];
		tz: string;
		currency: string;
		showDebtor?: boolean;
		onsettle?: (debt: Debt) => void;
		ondelete?: (debt: Debt) => void;
	} = $props();

	const showActions = $derived(Boolean(onsettle || ondelete));

	function rowActions(d: Debt): RowAction[] {
		const actions: RowAction[] = [];
		if (onsettle && !d.is_settled) {
			actions.push({
				icon: 'save',
				label: $_('debts.action.settle'),
				onclick: () => onsettle(d)
			});
		}
		if (ondelete) {
			actions.push({
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => ondelete(d)
			});
		}
		return actions;
	}
</script>

<div class="hidden md:block">
	<table class="w-full min-w-[48rem] text-left text-sm">
		<thead>
			<tr style:color="var(--text-muted)">
				{#if showDebtor}
					<th class="p-3">{$_('debts.col.debtor')}</th>
				{/if}
				<th class="p-3 whitespace-nowrap">{$_('debts.col.direction')}</th>
				<th class="p-3 whitespace-nowrap">{$_('transactions.col.account')}</th>
				<th class="p-3 whitespace-nowrap text-right">{$_('transactions.col.amount')}</th>
				<th class="p-3 whitespace-nowrap">{$_('debts.col.debtDate')}</th>
				<th class="p-3 whitespace-nowrap" title={$_('debts.col.due')}>
					{$_('debts.col.dueShort')}
				</th>
				{#if showActions}
					<th class="p-3 w-0"></th>
				{/if}
			</tr>
		</thead>
		<tbody>
			{#each debts as d (d.id)}
				<tr class="border-t" style:border-color="var(--border)">
					{#if showDebtor}
						<td class="p-3 font-medium">
							<a
								href={resolve(`/debtors/${d.debtor_id}`)}
								class="hover:underline"
								style:color="var(--primary)"
							>
								{d.debtor_name}
							</a>
						</td>
					{/if}
					<td class="p-3 whitespace-nowrap" style:color="var(--text-muted)">
						{d.direction === 'lent' ? $_('debts.direction.lent') : $_('debts.direction.borrowed')}
						{#if !d.affects_balance}
							<span class="ml-1 text-xs">({$_('debts.badge.noBalance')})</span>
						{/if}
					</td>
					<td class="p-3 whitespace-nowrap">
						{#if d.account_id}
							<EntityLink kind="account" id={d.account_id} label={d.account_name || d.account_id} />
						{:else}
							<span style:color="var(--text-muted)">—</span>
						{/if}
					</td>
					<td class="p-3 whitespace-nowrap text-right tabular-nums font-medium">
						<MoneyDisplay value={d.amount_display} {currency} class="" />
					</td>
					<td class="p-3 whitespace-nowrap">
						{formatAPIOperationDateTimeForDisplay(d.debt_date, tz)}
					</td>
					<td class="p-3 whitespace-nowrap">
						{formatAPIDateForDisplay(d.due_date, tz)}
						{#if d.is_overdue}
							<span
								class="ml-2 inline-block rounded px-1.5 py-0.5 text-xs whitespace-nowrap"
								style:background-color="color-mix(in srgb, var(--danger) 15%, transparent)"
								style:color="var(--danger)"
							>
								{$_('debts.badge.overdue')}
							</span>
						{/if}
					</td>
					{#if showActions}
						<td class="p-3 whitespace-nowrap text-right">
							<RowActionsMenu actions={rowActions(d)} />
						</td>
					{/if}
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<div class="space-y-3 md:hidden">
	{#each debts as d (d.id)}
		<article class="rounded-xl border p-4" style:border-color="var(--border)">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					{#if showDebtor}
						<a
							href={resolve(`/debtors/${d.debtor_id}`)}
							class="font-medium hover:underline"
							style:color="var(--primary)"
						>
							{d.debtor_name}
						</a>
					{/if}
					<p class="mt-1 text-sm" style:color="var(--text-muted)">
						{d.direction === 'lent' ? $_('debts.direction.lent') : $_('debts.direction.borrowed')}
						{#if !d.affects_balance}
							· {$_('debts.badge.noBalance')}
						{/if}
					</p>
				</div>
				<div class="flex shrink-0 items-start gap-2">
					<p class="text-base font-semibold tabular-nums">
						<MoneyDisplay value={d.amount_display} {currency} class="" />
					</p>
					{#if showActions}
						<RowActionsMenu actions={rowActions(d)} />
					{/if}
				</div>
			</div>
			<dl class="mt-3 grid gap-2 text-sm">
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('transactions.col.account')}</dt>
					<dd class="text-right">
						{#if d.account_id}
							<EntityLink kind="account" id={d.account_id} label={d.account_name || d.account_id} />
						{:else}
							—
						{/if}
					</dd>
				</div>
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('debts.col.debtDate')}</dt>
					<dd>{formatAPIOperationDateTimeForDisplay(d.debt_date, tz)}</dd>
				</div>
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('debts.col.dueShort')}</dt>
					<dd>
						{formatAPIDateForDisplay(d.due_date, tz)}
						{#if d.is_overdue}
							<span class="ml-1 text-xs" style:color="var(--danger)">
								{$_('debts.badge.overdue')}
							</span>
						{/if}
					</dd>
				</div>
			</dl>
		</article>
	{/each}
</div>
