<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		deleteUserData,
		getUserSettings,
		listCurrencies,
		putUserSettings
	} from '$lib/api/client';
	import { currencySelectOptions } from '$lib/currency';
	import { user } from '$lib/stores/auth';
	import { applyThemePreference, isThemePreference, type ThemePreference } from '$lib/theme';
	import { isUiStyle, readUiStyle, setUiStyle, type UiStyle } from '$lib/ui-style';
	import { setLocale } from '$lib/i18n';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import Select from '$lib/components/Select.svelte';
	import { confirm } from '$lib/confirm';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let wiping = $state(false);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);
	let displayName = $state('');
	let language = $state('ru');
	let currency = $state('RUB');
	let currencyOptions = $state<{ value: string; label: string }[]>([]);
	let timezone = $state('Europe/Moscow');
	let theme = $state<ThemePreference>('system');
	let uiStyle = $state<UiStyle>('classic');

	onMount(() => {
		uiStyle = readUiStyle();
		void loadProfile();
	});

	function onUiStyleChange(value: string) {
		if (!isUiStyle(value)) return;
		uiStyle = value;
	}

	async function loadProfile() {
		pageLoading = true;
		try {
			const [s, currencies] = await Promise.all([getUserSettings(), listCurrencies()]);
			displayName = s.display_name;
			language = s.language;
			currency = s.currency;
			currencyOptions = currencySelectOptions(currencies);
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
			setUiStyle(uiStyle);
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

	async function wipeData() {
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
			toast($_('settings.data.success'));
			await goto(resolve('/'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			wiping = false;
		}
	}
</script>

{#snippet segmented(
	id: string,
	label: string,
	value: string,
	options: { value: string; label: string }[],
	onpick: (v: string) => void
)}
	<div class="space-y-1.5">
		<span class="block text-sm font-medium" id={`${id}-label`}>{label}</span>
		<div class="segmented" role="radiogroup" {id} aria-labelledby={`${id}-label`}>
			{#each options as opt (opt.value)}
				<button
					type="button"
					role="radio"
					aria-checked={value === opt.value}
					onclick={() => onpick(opt.value)}
				>
					{opt.label}
				</button>
			{/each}
		</div>
	</div>
{/snippet}

<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void loadProfile()}>
	<div class="max-w-2xl space-y-8">
		<form class="card space-y-6" onsubmit={saveProfile}>
			<section class="space-y-4">
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('settings.profile.section.account')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('settings.profile.section.account_hint')}
					</p>
				</div>
				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<label class="field-label" for="login">{$_('settings.login')}</label>
						<input
							id="login"
							class="input cursor-not-allowed opacity-80"
							type="text"
							value={$user?.login ?? ''}
							readonly
							tabindex="-1"
						/>
						<p class="mt-1 text-xs" style:color="var(--text-muted)">
							{$_('settings.login.readonly')}
						</p>
					</div>
					<div>
						<label class="field-label" for="display">{$_('register.display_name')}</label>
						<input id="display" class="input" bind:value={displayName} />
					</div>
				</div>
			</section>

			<section class="space-y-4 border-t pt-6" style:border-color="var(--border)">
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('settings.profile.section.region')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('settings.profile.section.region_hint')}
					</p>
				</div>
				<div class="grid gap-4 sm:grid-cols-2">
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
						options={currencyOptions}
					/>
				</div>
				<TimezonePicker
					id="tz"
					label={$_('settings.timezone')}
					hint={$_('settings.timezone.hint')}
					bind:value={timezone}
				/>
			</section>

			<section class="space-y-4 border-t pt-6" style:border-color="var(--border)">
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('settings.profile.section.appearance')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('settings.profile.section.appearance_hint')}
					</p>
				</div>
				{@render segmented(
					'theme',
					$_('settings.theme'),
					theme,
					[
						{ value: 'system', label: $_('settings.theme.system') },
						{ value: 'light', label: $_('settings.theme.light') },
						{ value: 'dark', label: $_('settings.theme.dark') }
					],
					(v) => {
						if (isThemePreference(v)) theme = v;
					}
				)}
				{@render segmented(
					'ui-style',
					$_('settings.ui_style'),
					uiStyle,
					[
						{ value: 'classic', label: $_('settings.ui_style.classic') },
						{ value: 'material', label: $_('settings.ui_style.material') }
					],
					onUiStyleChange
				)}
			</section>

			<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
				<button type="submit" class="btn-primary" disabled={loading || wiping}
					>{$_('settings.save')}</button
				>
			</div>
		</form>

		<section class="space-y-3">
			<GroupHeading
				title={$_('settings.profile.section.danger')}
				hint={$_('settings.profile.section.danger_hint')}
			/>
			<div
				class="card space-y-3"
				style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
			>
				<h3 class="font-medium">{$_('settings.data.title')}</h3>
				<p class="text-sm" style:color="var(--text-muted)">{$_('settings.data.hint')}</p>
				<div class="flex justify-end">
					<button
						type="button"
						class="btn-danger"
						disabled={loading || wiping}
						onclick={() => void wipeData()}>{$_('settings.data.title')}</button
					>
				</div>
			</div>
		</section>
	</div>
</PageLoadGate>
