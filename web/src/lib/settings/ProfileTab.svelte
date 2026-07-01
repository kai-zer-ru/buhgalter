<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { getUserSettings, putUserSettings } from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { applyTheme } from '$lib/theme';
	import { setLocale } from '$lib/i18n';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import Select from '$lib/components/Select.svelte';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let displayName = $state('');
	let language = $state('ru');
	let currency = $state('RUB');
	let timezone = $state('Europe/Moscow');
	let theme = $state<'light' | 'dark'>('light');

	onMount(() => {
		void loadProfile().catch((err) => toast.fromError(err));
	});

	async function loadProfile() {
		const s = await getUserSettings();
		displayName = s.display_name;
		language = s.language;
		currency = s.currency;
		timezone = s.timezone;
		theme = s.theme === 'dark' ? 'dark' : 'light';
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
			applyTheme(updated.theme === 'dark' ? 'dark' : 'light');
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
			{ value: 'light', label: $_('settings.theme.light') },
			{ value: 'dark', label: $_('settings.theme.dark') }
		]}
	/>
	<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
</form>
