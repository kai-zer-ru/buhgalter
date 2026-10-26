<script lang="ts">
	import { nativeTimePicker } from '$lib/actions/native-time-picker';
	import { _ } from 'svelte-i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';

	let {
		open,
		time = $bindable(''),
		saving = false,
		onclose,
		onsave
	}: {
		open: boolean;
		time?: string;
		saving?: boolean;
		onclose: () => void;
		onsave: () => void;
	} = $props();
</script>

<ModalShell {open} title={$_('transactions.changeTime')} maxWidth="max-w-sm" {onclose}>
	<label class="block text-sm">
		<span class="font-medium">{$_('transactions.changeTime')}</span>
		<input class="input mt-1 w-full" type="time" use:nativeTimePicker step="60" bind:value={time} />
	</label>
	{#snippet footer()}
		<button type="button" class="btn-ghost" disabled={saving} onclick={onclose}>
			{$_('common.cancel')}
		</button>
		<button type="button" class="btn-primary" disabled={saving || !time} onclick={onsave}>
			{saving ? $_('common.loading') : $_('common.save')}
		</button>
	{/snippet}
</ModalShell>
