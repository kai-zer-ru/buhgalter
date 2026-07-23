<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
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
</script>

{#if loading}
	<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
{:else}
	<div class="card space-y-6">
		<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.settingsHint')}</p>

		<div class="flex items-center justify-between gap-4">
			<div>
				<p class="font-medium">{$_('appLock.enable')}</p>
				<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.enableHint')}</p>
			</div>
			<ToggleSwitch
				label={$_('appLock.enable')}
				checked={config.enabled}
				onchange={() => void toggleEnabled()}
			/>
		</div>

		{#if config.enabled}
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn-ghost" onclick={() => (changeOpen = true)}>
					{$_('appLock.changePin')}
				</button>
			</div>

			<div
				class="flex items-center justify-between gap-4 border-t pt-4"
				style:border-color="var(--border)"
			>
				<div>
					<p class="font-medium">{$_('appLock.biometric')}</p>
					<p class="text-sm" style:color="var(--text-muted)">
						{biometricAvailable ? $_('appLock.biometricHint') : $_('appLock.biometricUnavailable')}
					</p>
				</div>
				<ToggleSwitch
					label={$_('appLock.biometric')}
					checked={config.biometricEnabled}
					disabled={!biometricAvailable}
					onchange={() => void toggleBiometric()}
				/>
			</div>

			<label class="block space-y-1.5 border-t pt-4" style:border-color="var(--border)">
				<span class="font-medium">{$_('appLock.backgroundTimeout')}</span>
				<span class="block text-sm" style:color="var(--text-muted)">
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
		{/if}

		<p class="text-xs" style:color="var(--text-muted)">{$_('appLock.limitations')}</p>
	</div>
{/if}

{#if setupOpen}
	<form class="card mt-4 space-y-4" onsubmit={saveSetup}>
		<h2 class="text-lg font-semibold">{$_('appLock.setupTitle')}</h2>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={setupPin}
			placeholder="••••"
			autocomplete="off"
		/>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={setupConfirm}
			placeholder="••••"
			autocomplete="off"
		/>
		<div class="flex gap-2">
			<button type="submit" class="btn-primary">{$_('common.save')}</button>
			<button type="button" class="btn-ghost" onclick={() => (setupOpen = false)}
				>{$_('common.cancel')}</button
			>
		</div>
	</form>
{/if}

{#if changeOpen}
	<form class="card mt-4 space-y-4" onsubmit={saveChange}>
		<h2 class="text-lg font-semibold">{$_('appLock.changePin')}</h2>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={currentPin}
			autocomplete="off"
		/>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={newPin}
			autocomplete="off"
		/>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={newPinConfirm}
			autocomplete="off"
		/>
		<div class="flex gap-2">
			<button type="submit" class="btn-primary">{$_('common.save')}</button>
			<button type="button" class="btn-ghost" onclick={() => (changeOpen = false)}
				>{$_('common.cancel')}</button
			>
		</div>
	</form>
{/if}

{#if disableOpen}
	<form class="card mt-4 space-y-4" onsubmit={confirmDisable}>
		<h2 class="text-lg font-semibold">{$_('appLock.disableTitle')}</h2>
		<p class="text-sm" style:color="var(--text-muted)">{$_('appLock.disableHint')}</p>
		<input
			class="input text-center tracking-[0.5em]"
			type="password"
			inputmode="numeric"
			maxlength="4"
			bind:value={disablePin}
			autocomplete="off"
		/>
		<div class="flex gap-2">
			<button type="submit" class="btn-danger">{$_('appLock.disableConfirm')}</button>
			<button type="button" class="btn-ghost" onclick={() => (disableOpen = false)}
				>{$_('common.cancel')}</button
			>
		</div>
	</form>
{/if}
