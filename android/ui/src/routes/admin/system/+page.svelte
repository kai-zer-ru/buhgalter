<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		getAdminSettings,
		putAdminNotificationSecretKey,
		putAdminSettings
	} from '$lib/api/client';
	import { toast } from '$lib/toast';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { user } from '$lib/stores/auth';

	let externalURL = $state('');
	let secretKeySet = $state(false);
	let notificationSecretKey = $state('');
	let loading = $state(false);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);

	onMount(() => {
		void reloadSettings();
	});

	async function reloadSettings() {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		pageLoading = true;
		try {
			const s = await getAdminSettings();
			externalURL = s.external_url ?? '';
			secretKeySet = s.secret_key_set;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err);
			if (msg) loadError = msg;
		} finally {
			pageLoading = false;
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			await putAdminSettings({
				external_url: externalURL.trim()
			});
			const s = await getAdminSettings();
			secretKeySet = s.secret_key_set;
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveSecretKey() {
		if (!notificationSecretKey.trim()) {
			toast.error($_('admin.system.secret.enter'));
			return;
		}
		loading = true;
		try {
			await putAdminNotificationSecretKey(notificationSecretKey.trim());
			notificationSecretKey = '';
			const s = await getAdminSettings();
			secretKeySet = s.secret_key_set;
			toast($_('admin.system.secret.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}
</script>

{#snippet statusBadge(set: boolean, setKey: string, unsetKey: string)}
	<span class="badge" class:badge-success={set} class:admin-badge-warning={!set}
		>{$_(set ? setKey : unsetKey)}</span
	>
{/snippet}

<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void reloadSettings()}>
	<div class="space-y-4">
		<form class="card space-y-4" onsubmit={submit}>
			<div class="space-y-1">
				<h2 class="font-semibold">{$_('admin.system.external_url')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('admin.system.external_url_hint')}
				</p>
			</div>
			<input
				id="external"
				type="url"
				class="input"
				aria-label={$_('admin.system.external_url')}
				placeholder="https://buhgalter.example.com"
				bind:value={externalURL}
			/>
			<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
				<button type="submit" class="btn-primary" disabled={loading}>{$_('common.save')}</button>
			</div>
		</form>

		<div class="card space-y-4">
			<div class="space-y-1">
				<div class="flex flex-wrap items-center gap-2">
					<h2 class="font-semibold">{$_('admin.system.secret.title')}</h2>
					{@render statusBadge(
						secretKeySet,
						'admin.system.secret.status.set',
						'admin.system.secret.status.unset'
					)}
				</div>
				<p class="text-sm" style:color="var(--text-muted)">{$_('admin.system.secret.hint')}</p>
			</div>
			<div>
				<label class="field-label" for="notification-secret">
					{$_('admin.system.secret.label')}
				</label>
				<input
					id="notification-secret"
					type="password"
					class="input"
					placeholder={$_('admin.system.secret.placeholder')}
					bind:value={notificationSecretKey}
					autocomplete="new-password"
					minlength="32"
					maxlength="32"
				/>
			</div>
			<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
				<button type="button" class="btn-primary" onclick={saveSecretKey} disabled={loading}>
					{$_('admin.system.secret.save')}
				</button>
			</div>
		</div>
	</div>
</PageLoadGate>

<style>
	.admin-badge-warning {
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 30%, transparent);
		background-color: color-mix(in srgb, var(--warning) 14%, transparent);
	}
</style>
