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

<div class="space-y-4">
	<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void reloadSettings()}>
		<form class="card max-w-lg space-y-4" onsubmit={submit}>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="external"
					>{$_('admin.system.external_url')}</label
				>
				<input
					id="external"
					type="url"
					class="input"
					placeholder="https://buhgalter.example.com"
					bind:value={externalURL}
				/>
			</div>
			<button type="submit" class="btn-primary" disabled={loading}>{$_('common.save')}</button>
		</form>

		<div class="card max-w-lg space-y-2">
			<p class="text-sm font-medium">{$_('admin.system.secret.title')}</p>
			<p class="text-xs" style:color="var(--text-muted)">{$_('admin.system.secret.hint')}</p>
			<p class="text-xs" style:color="var(--text-muted)">
				{$_('admin.system.secret.status')}
				{secretKeySet
					? $_('admin.system.secret.status.set')
					: $_('admin.system.secret.status.unset')}
			</p>
			<label class="mb-1.5 block text-sm font-medium" for="notification-secret">
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
			<button type="button" class="btn-primary" onclick={saveSecretKey} disabled={loading}>
				{$_('admin.system.secret.save')}
			</button>
		</div>

		<div class="card max-w-lg space-y-2">
			<p class="text-sm font-medium">{$_('admin.system.receiptToken.title')}</p>
			<p class="text-xs" style:color="var(--text-muted)">
				{$_('admin.system.receiptToken.hintBefore')}<a
					href="https://proverkacheka.com/"
					target="_blank"
					rel="noopener noreferrer"
					class="underline underline-offset-2"
					style:color="var(--text-primary)">proverkacheka.com</a
				>{$_('admin.system.receiptToken.hintAfter')}
			</p>
			<details>
				<summary class="cursor-pointer text-xs" style:color="var(--text-muted)">
					{$_('admin.system.receiptToken.more')}
				</summary>
				<p class="mt-1 text-xs" style:color="var(--text-muted)">
					{$_('admin.system.receiptToken.limits')}
				</p>
			</details>
			<p class="text-xs" style:color="var(--text-muted)">
				{$_('admin.system.receiptToken.status')}
				{receiptApiTokenSet
					? $_('admin.system.receiptToken.status.set')
					: $_('admin.system.receiptToken.status.unset')}
			</p>
			<p class="text-xs" style:color="var(--text-muted)">
				{$_('admin.system.receiptToken.enableModuleBeforeLink')}<a
					href={resolve('/admin/features')}
					class="underline underline-offset-2"
					style:color="var(--text-primary)">{$_('admin.tab.features')}</a
				>{$_('admin.system.receiptToken.enableModuleAfterLink')}
			</p>
			<label class="mb-1.5 block text-sm font-medium" for="receipt-api-token">
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
			<div class="flex flex-wrap gap-2 pt-1">
				<button type="button" class="btn-primary" onclick={saveReceiptApiToken} disabled={loading}>
					{$_('admin.system.receiptToken.save')}
				</button>
				<a
					href="https://proverkacheka.com/cabinet/profile/edit"
					target="_blank"
					rel="noopener noreferrer"
					class="btn-ghost inline-flex items-center gap-2"
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
			</div>
		</div>
	</PageLoadGate>
</div>
