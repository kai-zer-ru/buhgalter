<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { formatBalance } from '$lib/finance';
	import { fromCents } from '$lib/money';
	import { user } from '$lib/stores/auth';

	type Props = {
		iOwe: number;
		owedToMe: number;
	};

	let { iOwe, owedToMe }: Props = $props();

	const currency = $derived($user?.currency ?? 'RUB');
</script>

<div class="card flex flex-wrap gap-6">
	<div>
		<p class="text-sm" style:color="var(--text-muted)">{$_('debts.summary.iOwe')}</p>
		<p class="text-xl font-semibold tabular-nums" style:color="var(--danger)">
			{formatBalance(fromCents(iOwe), currency)}
		</p>
	</div>
	<div>
		<p class="text-sm" style:color="var(--text-muted)">{$_('debts.summary.owedToMe')}</p>
		<p class="text-xl font-semibold tabular-nums" style:color="var(--primary)">
			{formatBalance(fromCents(owedToMe), currency)}
		</p>
	</div>
</div>
