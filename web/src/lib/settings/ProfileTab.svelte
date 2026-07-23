<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { getUserSettings, putUserSettings } from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { applyThemePreference, isThemePreference, type ThemePreference } from '$lib/theme';
	import { setLocale } from '$lib/i18n';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

	let loading = $state(false);
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
			toast($_('common.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
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
		<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
	</form>
</PageLoadGate>
