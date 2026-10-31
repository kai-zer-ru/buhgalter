<script lang="ts">
	import { _ } from 'svelte-i18n';
	import MoneyDisplay from '$lib/components/MoneyDisplay.svelte';
	import { user } from '$lib/stores/auth';

	type Props = {
		iOwe: number;
		owedToMe: number;
	};

	let { iOwe, owedToMe }: Props = $props();

	const currency = $derived($user?.currency ?? 'RUB');
</script>

<div class="card overflow-hidden p-0 sm:p-0">
	<div class="stat-grid grid-cols-2">
		<div>
			<p class="stat-label">{$_('debts.summary.iOwe')}</p>
			<p class="stat-value sm:text-2xl" style:color="var(--danger)">
				<MoneyDisplay cents={iOwe} {currency} class="" />
			</p>
		</div>
		<div>
			<p class="stat-label">{$_('debts.summary.owedToMe')}</p>
			<p class="stat-value sm:text-2xl" style:color="var(--primary)">
				<MoneyDisplay cents={owedToMe} {currency} class="" />
			</p>
		</div>
	</div>
</div>
