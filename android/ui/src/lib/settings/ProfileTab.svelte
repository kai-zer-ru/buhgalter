<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import { deleteUserData, getUserSettings, putUserSettings } from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { applyThemePreference, isThemePreference, type ThemePreference } from '$lib/theme';
	import { setLocale } from '$lib/i18n';
	import { syncRemoteI18nOnMismatch } from '$lib/i18n/remote-sync';
	import { APP_VERSION } from '$lib/platform/app-version';
	import { fetchAppVersionInfo } from '$lib/version-check';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { requireOnline } from '$lib/offline/require-online';
	import { clearOutbox } from '$lib/offline/store';
	import { notifyServerDataChanged, warmRefCache } from '$lib/offline/sync';
	import { clearInterceptDrafts } from '$lib/android/notification-intercept/drafts';
	import {
		loadInterceptSettings,
		saveInterceptSettings
	} from '$lib/android/notification-intercept/settings';

	let loading = $state(false);
	let wiping = $state(false);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);
	let displayName = $state('');
	let language = $state('ru');
	let currency = $state('RUB');
	let timezone = $state('Europe/Moscow');
	let theme = $state<ThemePreference>('system');

	onMount(() => {
		void loadProfile();
	});

	async function loadProfile() {
		pageLoading = true;
		try {
			const s = await getUserSettings();
			displayName = s.display_name;
			language = s.language;
			currency = s.currency;
			timezone = s.timezone;
			theme = isThemePreference(s.theme) ? s.theme : 'system';
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err);
			if (msg) loadError = msg;
		} finally {
			pageLoading = false;
		}
	}

	async function saveProfile(e: Event) {
		e.preventDefault();
		if (!requireOnline('offline.onlineOnly.settings')) return;
		loading = true;
		try {
			const updated = await putUserSettings({
				display_name: displayName,
				language,
				currency,
				timezone,
				theme
			});
			localStorage.setItem('theme', updated.theme);
			if (isThemePreference(updated.theme)) {
				applyThemePreference(updated.theme);
			}
			setLocale(updated.language);
			user.update((u) => (u ? { ...u, ...updated } : u));
			timezone = updated.timezone;
			const versionInfo = await fetchAppVersionInfo(APP_VERSION);
			await syncRemoteI18nOnMismatch(APP_VERSION, versionInfo.serverVersion, updated.language);
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function wipeData() {
		if (!requireOnline('offline.onlineOnly.settings')) return;
		const first = await confirm({
			title: $_('settings.data.confirm.title'),
			message: $_('settings.data.confirm.message'),
			confirmLabel: $_('settings.data.confirm.continue'),
			danger: true
		});
		if (!first) return;
		const second = await confirm({
			title: $_('settings.data.confirm.finalTitle'),
			message: $_('settings.data.confirm.finalMessage'),
			confirmLabel: $_('settings.data.title'),
			danger: true
		});
		if (!second) return;
		wiping = true;
		try {
			await deleteUserData();
			clearOutbox();
			const userId = get(user)?.id;
			if (userId) {
				clearInterceptDrafts(userId);
				const prev = loadInterceptSettings(userId);
				saveInterceptSettings(userId, { ...prev, bankBindings: [], cardBindings: [] });
			}
			notifyServerDataChanged();
			const { resetWidgetPublishForTests, publishWidgetSnapshot } =
				await import('$lib/widgets/publish');
			resetWidgetPublishForTests();
			void publishWidgetSnapshot();
			void warmRefCache({ force: true });
			toast($_('settings.data.success'));
			await goto(resolve('/'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			wiping = false;
		}
	}
</script>

<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void loadProfile()}>
	<form class="card max-w-lg space-y-4" onsubmit={saveProfile}>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="login">{$_('settings.login')}</label>
			<input
				id="login"
				class="input cursor-not-allowed opacity-80"
				type="text"
				value={$user?.login ?? ''}
				readonly
				tabindex="-1"
			/>
			<p class="mt-1 text-xs" style:color="var(--text-muted)">{$_('settings.login.readonly')}</p>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="display"
				>{$_('register.display_name')}</label
			>
			<input id="display" class="input" bind:value={displayName} />
		</div>
		<Select
			id="lang"
			label={$_('settings.language')}
			bind:value={language}
			options={[
				{ value: 'ru', label: 'Русский' },
				{ value: 'en', label: 'English' }
			]}
		/>
		<Select
			id="currency"
			label={$_('settings.currency')}
			bind:value={currency}
			options={[
				{ value: 'RUB', label: 'RUB' },
				{ value: 'USD', label: 'USD' },
				{ value: 'EUR', label: 'EUR' }
			]}
		/>
		<TimezonePicker
			id="tz"
			label={$_('settings.timezone')}
			hint={$_('settings.timezone.hint')}
			bind:value={timezone}
		/>
		<Select
			id="theme"
			label={$_('settings.theme')}
			bind:value={theme}
			options={[
				{ value: 'system', label: $_('settings.theme.system') },
				{ value: 'light', label: $_('settings.theme.light') },
				{ value: 'dark', label: $_('settings.theme.dark') }
			]}
		/>
		<button type="submit" class="btn-primary" disabled={loading || wiping}
			>{$_('settings.save')}</button
		>
	</form>
	<div class="card mt-4 max-w-lg space-y-3">
		<h2 class="font-medium">{$_('settings.data.title')}</h2>
		<p class="text-sm" style:color="var(--text-muted)">{$_('settings.data.hint')}</p>
		<button
			type="button"
			class="btn-danger"
			disabled={loading || wiping}
			onclick={() => void wipeData()}>{$_('settings.data.title')}</button
		>
	</div>
</PageLoadGate>
