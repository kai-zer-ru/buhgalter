<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		getAdminSettings,
		putAdminNotificationSecretKey,
		putAdminReceiptApiToken,
		putAdminSettings
	} from '$lib/api/client';
	import { loadFeatureFlags } from '$lib/features';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let externalURL = $state('');
	let secretKeySet = $state(false);
	let receiptApiTokenSet = $state(false);
	let notificationSecretKey = $state('');
	let receiptApiToken = $state('');
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
			receiptApiTokenSet = s.receipt_api_token_set === true;
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
			receiptApiTokenSet = s.receipt_api_token_set === true;
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
			receiptApiTokenSet = s.receipt_api_token_set === true;
			toast($_('admin.system.secret.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveReceiptApiToken() {
		if (!receiptApiToken.trim()) {
			toast.error($_('admin.system.receiptToken.enter'));
			return;
		}
		loading = true;
		try {
			await putAdminReceiptApiToken(receiptApiToken.trim());
			receiptApiToken = '';
			const s = await getAdminSettings();
			receiptApiTokenSet = s.receipt_api_token_set === true;
			await loadFeatureFlags();
			toast($_('admin.system.receiptToken.saved'));
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

		<div class="grid gap-4 lg:grid-cols-2">
			<div class="card flex h-full flex-col gap-4">
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
				<div class="mt-auto flex justify-end border-t pt-4" style:border-color="var(--border)">
					<button type="button" class="btn-primary" onclick={saveSecretKey} disabled={loading}>
						{$_('admin.system.secret.save')}
					</button>
				</div>
			</div>

			<div class="card flex h-full flex-col gap-4">
				<div class="space-y-1">
					<div class="flex flex-wrap items-center gap-2">
						<h2 class="font-semibold">{$_('admin.system.receiptToken.title')}</h2>
						{@render statusBadge(
							receiptApiTokenSet,
							'admin.system.receiptToken.status.set',
							'admin.system.receiptToken.status.unset'
						)}
					</div>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('admin.system.receiptToken.hintBefore')}<a
							href="https://proverkacheka.com/"
							target="_blank"
							rel="noopener noreferrer"
							class="underline underline-offset-2">proverkacheka.com</a
						>{$_('admin.system.receiptToken.hintAfter')}
					</p>
				</div>
				<div
					class="space-y-2 rounded-xl border p-3 text-xs leading-relaxed"
					style:border-color="var(--border)"
					style:color="var(--text-muted)"
				>
					<p>
						{$_('admin.system.receiptToken.enableModuleBeforeLink')}<a
							href={resolve('/admin/features')}
							class="underline underline-offset-2">{$_('admin.tab.features')}</a
						>{$_('admin.system.receiptToken.enableModuleAfterLink')}
					</p>
					<details>
						<summary class="cursor-pointer">{$_('admin.system.receiptToken.more')}</summary>
						<p class="mt-1">{$_('admin.system.receiptToken.limits')}</p>
					</details>
				</div>
				<div>
					<label class="field-label" for="receipt-api-token">
						{$_('admin.system.receiptToken.label')}
					</label>
					<input
						id="receipt-api-token"
						type="password"
						class="input"
						placeholder={$_('admin.system.receiptToken.placeholder')}
						bind:value={receiptApiToken}
						autocomplete="new-password"
					/>
				</div>
				<div
					class="mt-auto flex flex-wrap items-center justify-end gap-2 border-t pt-4"
					style:border-color="var(--border)"
				>
					<a
						href="https://proverkacheka.com/cabinet/profile/edit"
						target="_blank"
						rel="noopener noreferrer"
						class="btn-ghost inline-flex items-center gap-2 sm:mr-auto"
					>
						<svg
							aria-hidden="true"
							class="h-4 w-4 shrink-0"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
							<polyline points="15 3 21 3 21 9" />
							<line x1="10" y1="14" x2="21" y2="3" />
						</svg>
						{$_('admin.system.receiptToken.getToken')}
					</a>
					<button
						type="button"
						class="btn-primary"
						onclick={saveReceiptApiToken}
						disabled={loading}
					>
						{$_('admin.system.receiptToken.save')}
					</button>
				</div>
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
