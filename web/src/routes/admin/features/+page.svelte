<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { getAdminFeatures, putAdminFeatures, type AdminFeatureFlag } from '$lib/api/client';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { loadFeatureFlags } from '$lib/features';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let featureItems = $state<AdminFeatureFlag[]>([]);
	let savingKey = $state<string | null>(null);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);

	onMount(() => {
		void reload();
	});

	async function reload() {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		pageLoading = true;
		try {
			featureItems = await getAdminFeatures();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err);
			if (msg) loadError = msg;
		} finally {
			pageLoading = false;
		}
	}

	async function toggle(key: string) {
		if (savingKey) return;
		const item = featureItems.find((f) => f.key === key);
		if (!item) return;
		const next = !item.enabled;
		const prev = featureItems;
		featureItems = featureItems.map((f) => (f.key === key ? { ...f, enabled: next } : f));
		savingKey = key;
		try {
			const snap = await putAdminFeatures({ [key]: next });
			featureItems = featureItems.map((f) => ({
				...f,
				enabled: snap[f.key] ?? f.enabled
			}));
			await loadFeatureFlags();
		} catch (err) {
			featureItems = prev;
			toast.fromError(err);
		} finally {
			savingKey = null;
		}
	}
</script>

<div>
	<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void reload()}>
		<section class="space-y-3">
			<GroupHeading
				title={$_('admin.features.group.title')}
				hint={$_('admin.features.warn.module')}
			/>
			<ul class="card list-divided p-0 sm:p-0">
				{#each featureItems as item (item.key)}
					<li class="flex items-center gap-4 px-4 py-3 sm:px-5">
						<div class="min-w-0 flex-1">
							<p class="text-sm font-medium">{$_(item.title_key)}</p>
							<p class="mt-0.5 text-xs leading-snug" style:color="var(--text-muted)">
								{$_(item.description_key)}
							</p>
							{#if item.key === 'registration' && !item.enabled}
								<p class="mt-1 text-xs leading-snug" style:color="var(--warning)">
									{$_('admin.features.warn.registration')}
								</p>
							{/if}
						</div>
						<ToggleSwitch
							checked={item.enabled}
							disabled={savingKey !== null}
							label={$_(item.title_key)}
							onchange={() => void toggle(item.key)}
						/>
					</li>
				{/each}
			</ul>
		</section>
	</PageLoadGate>
</div>
