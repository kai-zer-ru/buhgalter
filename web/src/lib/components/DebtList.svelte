<script lang="ts">
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import type { Debt } from '$lib/api/client';
	import EntityLink from '$lib/components/EntityLink.svelte';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { formatBalance } from '$lib/finance';

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
						{formatBalance(d.amount_display, currency)}
					</td>
					<td class="p-3 whitespace-nowrap">
						{formatAPIDateTimeForDisplay(d.debt_date, tz)}
					</td>
					<td class="p-3 whitespace-nowrap">
						{formatAPIDateTimeForDisplay(d.due_date, tz)}
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
							<div class="inline-flex flex-nowrap items-center justify-end gap-1">
								{#if onsettle && !d.is_settled}
									<button type="button" class="btn-ghost text-sm" onclick={() => onsettle(d)}>
										{$_('debts.action.settle')}
									</button>
								{/if}
								{#if ondelete}
									<button
										type="button"
										class="btn-ghost text-sm"
										style:color="var(--danger)"
										onclick={() => ondelete(d)}
									>
										{$_('common.delete')}
									</button>
								{/if}
							</div>
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
				<p class="shrink-0 text-base font-semibold tabular-nums">
					{formatBalance(d.amount_display, currency)}
				</p>
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
					<dd>{formatAPIDateTimeForDisplay(d.debt_date, tz)}</dd>
				</div>
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('debts.col.dueShort')}</dt>
					<dd>
						{formatAPIDateTimeForDisplay(d.due_date, tz)}
						{#if d.is_overdue}
							<span class="ml-1 text-xs" style:color="var(--danger)">
								{$_('debts.badge.overdue')}
							</span>
						{/if}
					</dd>
				</div>
			</dl>
			{#if showActions}
				<div class="mt-3 flex justify-end gap-2">
					{#if onsettle && !d.is_settled}
						<button type="button" class="btn-ghost text-sm" onclick={() => onsettle(d)}>
							{$_('debts.action.settle')}
						</button>
					{/if}
					{#if ondelete}
						<button
							type="button"
							class="btn-ghost text-sm"
							style:color="var(--danger)"
							onclick={() => ondelete(d)}
						>
							{$_('common.delete')}
						</button>
					{/if}
				</div>
			{/if}
		</article>
	{/each}
</div>
