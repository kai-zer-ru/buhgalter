<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { getCredit, updateCredit, type Credit } from '$lib/api/client';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { leaveForm } from '$lib/android/form-nav';
	import FormPageShell from '$lib/components/FormPageShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { toast } from '$lib/toast';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { reportPageLoadFailure } from '$lib/page-load';

	const creditId = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), `/credits/${creditId}`)
	);

	let credit = $state<Credit | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let saving = $state(false);
	let newCreditName = $state('');

	$effect(() => {
		if (!creditId) return;
		void load();
	});

	async function load() {
		loading = true;
		try {
			const c = await getCredit(creditId);
			credit = c;
			newCreditName = c.name?.trim() || '';
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!credit });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}

	async function save() {
		if (!credit) return;
		const trimmedName = newCreditName.trim();
		if (!trimmedName) {
			toast.error($_('credits.error.nameRequired'));
			return;
		}
		saving = true;
		try {
			await updateCredit(credit.id, { name: trimmedName });
			toast($_('common.saved'));
			finish();
		} catch (err) {
			toast.fromError(err);
		} finally {
			saving = false;
		}
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()} inline>
	{#if credit}
		<FormPageShell title={$_('credits.action.changeName')} backHref={returnTo} onback={finish}>
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)">{$_('credits.field.name')}</span>
				<input class="input w-full" bind:value={newCreditName} maxlength="128" />
			</label>
			{#snippet footer()}
				<button type="button" class="btn-ghost" onclick={finish}>{$_('common.cancel')}</button>
				<button type="button" class="btn-primary" disabled={saving} onclick={() => void save()}>
					{saving ? $_('common.loading') : $_('common.save')}
				</button>
			{/snippet}
		</FormPageShell>
	{/if}
</PageLoadGate>
