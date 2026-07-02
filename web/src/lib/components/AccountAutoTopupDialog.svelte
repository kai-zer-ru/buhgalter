<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { listAccounts, updateAccount, type Account } from '$lib/api/client';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import MoneyInput from '$lib/components/MoneyInput.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import {
		autoTopupSourceOptions,
		defaultAutoTopupSourceId,
		validateAutoTopupForm
	} from '$lib/accounts/auto-topup';
	import { formatMoneyForInput, toAPIAmount } from '$lib/money';
	import { toast } from '$lib/toast';

	type Props = {
		open: boolean;
		account: Account | null;
		onclose: () => void;
		onsaved: () => void;
	};

	let { open = $bindable(), account, onclose, onsaved }: Props = $props();

	let enabled = $state(false);
	let threshold = $state('');
	let target = $state('');
	let sourceId = $state('');
	let accounts = $state<Account[]>([]);
	let saving = $state(false);
	let formError = $state('');

	const sourceOptions = $derived(account ? autoTopupSourceOptions(accounts, account.id) : []);

	$effect(() => {
		if (!open || !account) return;
		enabled = account.auto_topup_enabled ?? false;
		threshold = account.auto_topup_threshold_display
			? formatMoneyForInput(account.auto_topup_threshold_display)
			: '';
		target = account.auto_topup_target_display
			? formatMoneyForInput(account.auto_topup_target_display)
			: '';
		sourceId = account.auto_topup_source_account_id ?? '';
		formError = '';
		void listAccounts().then((rows) => {
			accounts = rows;
			if (!account.auto_topup_source_account_id) {
				sourceId = defaultAutoTopupSourceId(rows, account.id);
			}
		});
	});

	async function save() {
		if (!account) return;
		const validation = validateAutoTopupForm(enabled, threshold, target, sourceId);
		if (validation === 'required') {
			formError = $_('accounts.autoTopup.error.required');
			return;
		}
		if (validation === 'range') {
			formError = $_('accounts.autoTopup.error.range');
			return;
		}
		if (validation === 'invalid') {
			formError = $_('accounts.autoTopup.error.invalid');
			return;
		}
		formError = '';
		saving = true;
		try {
			await updateAccount(account.id, {
				name: account.name,
				bank_id: account.bank_id ?? undefined,
				auto_topup_enabled: enabled,
				auto_topup_threshold: enabled ? toAPIAmount(threshold) : undefined,
				auto_topup_target: enabled ? toAPIAmount(target) : undefined,
				auto_topup_source_account_id: enabled ? sourceId : undefined
			});
			toast($_('accounts.autoTopup.saved'));
			onsaved();
			onclose();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

<ModalShell bind:open title={$_('accounts.autoTopup.title')} {onclose}>
	<div class="space-y-4">
		<label class="flex items-center justify-between gap-3">
			<span>{$_('accounts.autoTopup.enabled')}</span>
			<ToggleSwitch
				checked={enabled}
				label={$_('accounts.autoTopup.enabled')}
				onchange={() => (enabled = !enabled)}
			/>
		</label>
		{#if enabled}
			<div>
				<label class="mb-1 block text-sm" for="auto-topup-source">
					{$_('accounts.autoTopup.source')}
				</label>
				<Select
					id="auto-topup-source"
					bind:value={sourceId}
					options={sourceOptions}
					disabled={sourceOptions.length === 0}
				/>
			</div>
			<div>
				<label class="mb-1 block text-sm" for="auto-topup-threshold">
					{$_('accounts.autoTopup.threshold')}
				</label>
				<MoneyInput id="auto-topup-threshold" bind:value={threshold} />
			</div>
			<div>
				<label class="mb-1 block text-sm" for="auto-topup-target">
					{$_('accounts.autoTopup.target')}
				</label>
				<MoneyInput id="auto-topup-target" bind:value={target} />
			</div>
		{/if}
		{#if formError}
			<p class="text-sm text-red-600">{formError}</p>
		{/if}
	</div>

	{#snippet footer()}
		<button type="button" class="btn-ghost" disabled={saving} onclick={onclose}>
			{$_('common.cancel')}
		</button>
		<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
