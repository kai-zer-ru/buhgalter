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
	import { formatApiError } from '$lib/api/errors';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let registrationEnabled = $state(false);
	let externalURL = $state('');
	let secretKeySet = $state(false);
	let notificationSecretKey = $state('');
	let systemError = $state('');
	let systemSuccess = $state('');
	let secretError = $state('');
	let secretSuccess = $state('');
	let loading = $state(false);

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		const s = await getAdminSettings();
		registrationEnabled = s.registration_enabled;
		externalURL = s.external_url ?? '';
		secretKeySet = s.secret_key_set;
	});

	async function submit(e: Event) {
		e.preventDefault();
		systemError = '';
		systemSuccess = '';
		loading = true;
		try {
			await putAdminSettings({
				registration_enabled: registrationEnabled,
				external_url: externalURL.trim()
			});
			const s = await getAdminSettings();
			secretKeySet = s.secret_key_set;
			systemSuccess = $_('common.saved');
			toast($_('common.saved'));
		} catch (err) {
			systemError = formatApiError(err);
		} finally {
			loading = false;
		}
	}

	async function saveSecretKey() {
		secretError = '';
		secretSuccess = '';
		if (!notificationSecretKey.trim()) {
			secretError = $_('admin.system.secret.enter');
			return;
		}
		loading = true;
		try {
			await putAdminNotificationSecretKey(notificationSecretKey.trim());
			notificationSecretKey = '';
			const s = await getAdminSettings();
			secretKeySet = s.secret_key_set;
			secretSuccess = $_('admin.system.secret.saved');
			toast($_('admin.system.secret.saved'));
		} catch (err) {
			secretError = formatApiError(err);
		} finally {
			loading = false;
		}
	}
</script>

<form class="card max-w-lg space-y-4" onsubmit={submit}>
	<div class="flex items-center justify-between gap-4">
		<div>
			<p class="text-sm font-medium">{$_('admin.system.registration.title')}</p>
			<p class="text-xs" style:color="var(--text-muted)">{$_('admin.system.registration.hint')}</p>
		</div>
		<button
			type="button"
			role="switch"
			aria-label={$_('admin.system.registration.title')}
			aria-checked={registrationEnabled}
			class="relative h-6 w-11 shrink-0 rounded-full transition-colors"
			style:background-color={registrationEnabled ? 'var(--primary)' : 'var(--border)'}
			onclick={() => (registrationEnabled = !registrationEnabled)}
		>
			<span
				class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform"
				class:translate-x-5={registrationEnabled}
			></span>
		</button>
	</div>
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
	<FormFeedback error={systemError} success={systemSuccess} />
</form>

<div class="card mt-4 max-w-lg space-y-2">
	<p class="text-sm font-medium">{$_('admin.system.secret.title')}</p>
	<p class="text-xs" style:color="var(--text-muted)">{$_('admin.system.secret.hint')}</p>
	<p class="text-xs" style:color="var(--text-muted)">
		{$_('admin.system.secret.status')}
		{secretKeySet ? $_('admin.system.secret.status.set') : $_('admin.system.secret.status.unset')}
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
	<FormFeedback error={secretError} success={secretSuccess} />
</div>
