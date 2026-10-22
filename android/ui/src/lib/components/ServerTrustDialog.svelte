<script lang="ts">
	import { _ } from 'svelte-i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import FieldHint from '$lib/components/FieldHint.svelte';

	type Props = {
		open: boolean;
		origin: string;
		trusted: boolean;
		onconfirm: () => void;
		oncancel: () => void;
	};

	let {
		open = $bindable(),
		origin,
		trusted = $bindable(false),
		onconfirm,
		oncancel
	}: Props = $props();
</script>

<ModalShell bind:open title={$_('serverSetup.ssl.title')} onclose={oncancel}>
	<div class="space-y-4">
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('serverSetup.ssl.message', { values: { origin } })}
		</p>
		<div class="flex items-center justify-between gap-4">
			<div>
				<p class="text-sm font-medium">{$_('serverSetup.ssl.trustToggle')}</p>
				<FieldHint text={$_('serverSetup.ssl.trustHint')} />
			</div>
			<ToggleSwitch
				checked={trusted}
				label={$_('serverSetup.ssl.trustToggle')}
				onchange={() => (trusted = !trusted)}
			/>
		</div>
	</div>
	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={oncancel}>{$_('common.cancel')}</button>
		<button type="button" class="btn-primary" onclick={onconfirm}>
			{$_('serverSetup.ssl.continue')}
		</button>
	{/snippet}
</ModalShell>
