<script lang="ts">
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import type { Credit } from '$lib/api/client';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { bankIconUrl, formatBalance } from '$lib/finance';

	let {
		credits,
		tz,
		currency,
		nameFor,
		bankIconFor = () => null
	}: {
		credits: Credit[];
		tz: string;
		currency: string;
		nameFor: (credit: Credit) => string;
		bankIconFor?: (credit: Credit) => string | null;
	} = $props();
</script>

<div class="hidden md:block">
	<table class="w-full text-left text-sm">
		<thead>
			<tr style:color="var(--text-muted)">
				<th class="p-3">{$_('credits.col.name')}</th>
				<th class="p-3">{$_('credits.col.remaining')}</th>
				<th class="p-3">{$_('credits.col.payment')}</th>
				<th class="p-3">{$_('credits.col.next')}</th>
			</tr>
		</thead>
		<tbody>
			{#each credits as c (c.id)}
				<tr class="border-t" style:border-color="var(--border)">
					<td class="p-3">
						<div class="flex items-center gap-2">
							{#if bankIconFor(c)}
								<img
									src={bankIconUrl(bankIconFor(c)!)}
									alt=""
									class="h-6 w-6 rounded-md"
									width="24"
									height="24"
								/>
							{/if}
							<a href={resolve(`/credits/${c.id}`)} class="font-medium hover:underline">
								{nameFor(c)}
							</a>
						</div>
						<div class="mt-1 flex flex-wrap items-center gap-1.5">
							{#if c.credit_kind === 'mortgage'}
								<span class="badge">{$_('credits.badge.mortgage')}</span>
							{:else if c.is_installment}
								<span class="badge">{$_('credits.badge.installment')}</span>
							{:else}
								<span class="badge">{$_('credits.badge.credit')}</span>
							{/if}
							{#if c.added_retroactively}
								<span class="badge">{$_('credits.badge.retroactive')}</span>
							{/if}
						</div>
					</td>
					<td class="p-3">{formatBalance(c.remaining_amount_display, currency)}</td>
					<td class="p-3">{formatBalance(c.monthly_payment_display, currency)}</td>
					<td class="p-3">
						{#if c.next_payment_date}
							{formatAPIDateTimeForDisplay(c.next_payment_date, tz)}
						{:else}
							—
						{/if}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<div class="space-y-3 md:hidden">
	{#each credits as c (c.id)}
		<article class="rounded-xl border p-4" style:border-color="var(--border)">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						{#if bankIconFor(c)}
							<img
								src={bankIconUrl(bankIconFor(c)!)}
								alt=""
								class="h-6 w-6 rounded-md"
								width="24"
								height="24"
							/>
						{/if}
						<a href={resolve(`/credits/${c.id}`)} class="font-medium hover:underline">
							{nameFor(c)}
						</a>
					</div>
					<div class="mt-1 flex flex-wrap items-center gap-1.5">
						{#if c.credit_kind === 'mortgage'}
							<span class="badge">{$_('credits.badge.mortgage')}</span>
						{:else if c.is_installment}
							<span class="badge">{$_('credits.badge.installment')}</span>
						{:else}
							<span class="badge">{$_('credits.badge.credit')}</span>
						{/if}
						{#if c.added_retroactively}
							<span class="badge">{$_('credits.badge.retroactive')}</span>
						{/if}
					</div>
				</div>
				<p class="shrink-0 text-base font-semibold tabular-nums">
					{formatBalance(c.remaining_amount_display, currency)}
				</p>
			</div>
			<dl class="mt-3 grid gap-2 text-sm">
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('credits.col.payment')}</dt>
					<dd>{formatBalance(c.monthly_payment_display, currency)}</dd>
				</div>
				<div class="flex justify-between gap-2">
					<dt style:color="var(--text-muted)">{$_('credits.col.next')}</dt>
					<dd>
						{#if c.next_payment_date}
							{formatAPIDateTimeForDisplay(c.next_payment_date, tz)}
						{:else}
							—
						{/if}
					</dd>
				</div>
			</dl>
		</article>
	{/each}
</div>
