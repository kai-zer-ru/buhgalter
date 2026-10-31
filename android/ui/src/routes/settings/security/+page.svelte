<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import {
		BACKGROUND_LOCK_MS,
		BACKGROUND_LOCK_OPTIONS_MS,
		changePin,
		disableAppLock,
		enableAppLock,
		getAppLockConfig,
		isBiometricAvailable,
		refreshAppLockConfig,
		setBackgroundLockMs,
		setBiometricEnabled,
		setShowWidgetsWhenLocked,
		type BackgroundLockMs,
		validateNewPin
	} from '$lib/platform/app-lock';
	import { toast } from '$lib/toast';

	let config = $state(getAppLockConfig());
	let biometricAvailable = $state(false);
	let loading = $state(true);

	let setupOpen = $state(false);
	let setupPin = $state('');
	let setupConfirm = $state('');

	let changeOpen = $state(false);
	let currentPin = $state('');
	let newPin = $state('');
	let newPinConfirm = $state('');

	let disablePin = $state('');
	let disableOpen = $state(false);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		config = await refreshAppLockConfig();
		biometricAvailable = await isBiometricAvailable();
		loading = false;
	}

	function timeoutLabel(ms: BackgroundLockMs): string {
		if (ms < 60_000) return $_('appLock.timeout.seconds', { values: { n: ms / 1000 } });
		if (ms < 60 * 60_000) return $_('appLock.timeout.minutes', { values: { n: ms / 60_000 } });
		return $_('appLock.timeout.hour');
	}

	async function toggleEnabled() {
		if (config.enabled) {
			disableOpen = true;
			return;
		}
		setupOpen = true;
		setupPin = '';
		setupConfirm = '';
	}

	function validateSetupPin(pin: string, confirm: string): boolean {
		if (pin !== confirm) {
			toast($_('appLock.pinMismatch'));
			return false;
		}
		const validation = validateNewPin(pin);
		if (validation === 'weak') {
			toast($_('appLock.pinWeak'));
			return false;
		}
		if (validation === 'format') {
			toast($_('appLock.pinMismatch'));
			return false;
		}
		return true;
	}

	async function saveSetup(e: Event) {
		e.preventDefault();
		if (!validateSetupPin(setupPin, setupConfirm)) return;
		await enableAppLock(setupPin);
		config = getAppLockConfig();
		setupOpen = false;
		toast($_('common.saved'));
	}

	async function saveChange(e: Event) {
		e.preventDefault();
		if (!validateSetupPin(newPin, newPinConfirm)) return;
		const ok = await changePin(currentPin, newPin);
		if (!ok) {
			toast($_('appLock.wrongPin'));
			return;
		}
		changeOpen = false;
		currentPin = '';
		newPin = '';
		newPinConfirm = '';
		toast($_('common.saved'));
	}

	async function confirmDisable(e: Event) {
		e.preventDefault();
		const ok = await disableAppLock(disablePin);
		if (!ok) {
			toast($_('appLock.wrongPin'));
			return;
		}
		config = getAppLockConfig();
		disableOpen = false;
		disablePin = '';
		toast($_('common.saved'));
	}

	async function toggleBiometric() {
		if (!biometricAvailable) return;
		await setBiometricEnabled(!config.biometricEnabled);
		config = getAppLockConfig();
		toast($_('common.saved'));
	}

	async function onTimeoutChange(e: Event) {
		const value = Number((e.currentTarget as HTMLSelectElement).value);
		const ms = BACKGROUND_LOCK_OPTIONS_MS.find((opt) => opt === value) ?? BACKGROUND_LOCK_MS;
		await setBackgroundLockMs(ms);
		config = getAppLockConfig();
		toast($_('common.saved'));
	}

	async function toggleShowWidgetsWhenLocked() {
		await setShowWidgetsWhenLocked(!config.showWidgetsWhenLocked);
		config = getAppLockConfig();
		toast($_('common.saved'));
	}
</script>

{#snippet pinInput(id: string, label: string, value: string, oninput: (v: string) => void)}
	<div class="space-y-1.5">
		<label class="block text-sm font-medium" for={id}>{label}</label>
		<input
			{id}
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			{value}
			oninput={(e) => oninput(e.currentTarget.value)}
			placeholder="••••"
			autocomplete="off"
		/>
	</div>
{/snippet}

{#snippet settingRow(title: string, hint: string, control: import('svelte').Snippet)}
	<div class="flex items-center justify-between gap-4 px-4 py-3.5">
		<div class="min-w-0">
			<p class="font-medium">{title}</p>
			<p class="mt-0.5 text-sm leading-snug" style:color="var(--text-muted)">{hint}</p>
		</div>
		{@render control()}
	</div>
{/snippet}

{#if loading}
	<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
{:else}
	<div class="space-y-8">
		<section class="space-y-3">
			<GroupHeading title={$_('appLock.group.lock')} hint={$_('appLock.settingsHint')} />
			<div class="card list-divided p-0 sm:p-0">
				{#snippet enableToggle()}
					<ToggleSwitch
						label={$_('appLock.enable')}
						checked={config.enabled}
						onchange={() => void toggleEnabled()}
					/>
				{/snippet}
				{@render settingRow($_('appLock.enable'), $_('appLock.enableHint'), enableToggle)}

				{#if config.enabled}
					{#snippet changePinButton()}
						<button type="button" class="btn-ghost shrink-0" onclick={() => (changeOpen = true)}>
							{$_('appLock.changePin')}
						</button>
					{/snippet}
					{@render settingRow($_('appLock.pinRow'), $_('appLock.pinRowHint'), changePinButton)}

					{#snippet biometricToggle()}
						<ToggleSwitch
							label={$_('appLock.biometric')}
							checked={config.biometricEnabled}
							disabled={!biometricAvailable}
							onchange={() => void toggleBiometric()}
						/>
					{/snippet}
					{@render settingRow(
						$_('appLock.biometric'),
						biometricAvailable ? $_('appLock.biometricHint') : $_('appLock.biometricUnavailable'),
						biometricToggle
					)}

					<label class="block space-y-2 px-4 py-3.5">
						<span class="block font-medium">{$_('appLock.backgroundTimeout')}</span>
						<span class="block text-sm leading-snug" style:color="var(--text-muted)">
							{$_('appLock.backgroundTimeoutHint')}
						</span>
						<select
							class="input w-full"
							value={String(config.backgroundLockMs)}
							onchange={(e) => void onTimeoutChange(e)}
						>
							{#each BACKGROUND_LOCK_OPTIONS_MS as ms (ms)}
								<option value={String(ms)}>{timeoutLabel(ms)}</option>
							{/each}
						</select>
					</label>

					{#snippet widgetsToggle()}
						<ToggleSwitch
							label={$_('appLock.showWidgetsWhenLocked')}
							checked={config.showWidgetsWhenLocked}
							onchange={() => void toggleShowWidgetsWhenLocked()}
						/>
					{/snippet}
					{@render settingRow(
						$_('appLock.showWidgetsWhenLocked'),
						$_('appLock.showWidgetsWhenLockedHint'),
						widgetsToggle
					)}
				{/if}
			</div>
			<p class="px-1 text-xs" style:color="var(--text-muted)">{$_('appLock.limitations')}</p>
		</section>

		{#if setupOpen}
			<form class="card space-y-4" onsubmit={saveSetup}>
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('appLock.setupTitle')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.pinRules')}</p>
				</div>
				{@render pinInput('pin-setup', $_('appLock.pinNew'), setupPin, (v) => (setupPin = v))}
				{@render pinInput(
					'pin-setup-confirm',
					$_('appLock.pinRepeat'),
					setupConfirm,
					(v) => (setupConfirm = v)
				)}
				<div class="flex justify-end gap-2 border-t pt-4" style:border-color="var(--border)">
					<button type="button" class="btn-ghost" onclick={() => (setupOpen = false)}
						>{$_('common.cancel')}</button
					>
					<button type="submit" class="btn-primary">{$_('common.save')}</button>
				</div>
			</form>
		{/if}

		{#if changeOpen}
			<form class="card space-y-4" onsubmit={saveChange}>
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('appLock.changePin')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.pinRules')}</p>
				</div>
				{@render pinInput(
					'pin-current',
					$_('appLock.pinCurrent'),
					currentPin,
					(v) => (currentPin = v)
				)}
				{@render pinInput('pin-new', $_('appLock.pinNew'), newPin, (v) => (newPin = v))}
				{@render pinInput(
					'pin-new-confirm',
					$_('appLock.pinRepeat'),
					newPinConfirm,
					(v) => (newPinConfirm = v)
				)}
				<div class="flex justify-end gap-2 border-t pt-4" style:border-color="var(--border)">
					<button type="button" class="btn-ghost" onclick={() => (changeOpen = false)}
						>{$_('common.cancel')}</button
					>
					<button type="submit" class="btn-primary">{$_('common.save')}</button>
				</div>
			</form>
		{/if}

		{#if disableOpen}
			<form
				class="card space-y-4"
				style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
				onsubmit={confirmDisable}
			>
				<div class="space-y-1">
					<h2 class="font-semibold">{$_('appLock.disableTitle')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.disableHint')}</p>
				</div>
				{@render pinInput(
					'pin-disable',
					$_('appLock.pinCurrent'),
					disablePin,
					(v) => (disablePin = v)
				)}
				<div class="flex justify-end gap-2 border-t pt-4" style:border-color="var(--border)">
					<button type="button" class="btn-ghost" onclick={() => (disableOpen = false)}
						>{$_('common.cancel')}</button
					>
					<button type="submit" class="btn-danger">{$_('appLock.disableConfirm')}</button>
				</div>
			</form>
		{/if}
	</div>
{/if}
