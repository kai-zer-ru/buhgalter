<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import {
		PIN_LENGTH,
		getAppLockConfig,
		isBiometricAvailable,
		retryBlockedMs,
		verifyBiometric,
		verifyPin
	} from '$lib/platform/app-lock';

	type Props = {
		onunlocked: () => void;
	};

	let { onunlocked }: Props = $props();

	let digits = $state('');
	let error = $state<string | null>(null);
	let busy = $state(false);
	let biometricReady = $state(false);
	let showBiometric = $state(false);

	const filled = $derived(digits.length);
	const keypad = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '', '0', 'back'];

	onMount(() => {
		void initBiometric();
	});

	async function initBiometric() {
		const config = getAppLockConfig();
		if (!config.biometricEnabled) return;
		biometricReady = await isBiometricAvailable();
		showBiometric = biometricReady;
		if (showBiometric) {
			await tryBiometric();
		}
	}

	async function tryBiometric() {
		if (busy || retryBlockedMs() > 0) return;
		busy = true;
		error = null;
		const ok = await verifyBiometric(
			$_('appLock.biometricReason'),
			$_('common.cancel'),
			$_('appLock.biometricTitle')
		);
		busy = false;
		if (ok) onunlocked();
	}

	function appendDigit(digit: string) {
		if (busy || retryBlockedMs() > 0) return;
		if (digits.length >= PIN_LENGTH) return;
		error = null;
		digits += digit;
		if (digits.length === PIN_LENGTH) {
			void submitPin();
		}
	}

	function backspace() {
		if (busy) return;
		error = null;
		digits = digits.slice(0, -1);
	}

	async function submitPin() {
		busy = true;
		const result = await verifyPin(digits);
		busy = false;
		if (result.ok) {
			digits = '';
			onunlocked();
			return;
		}
		digits = '';
		if (result.retryAfterMs && result.retryAfterMs > 0) {
			error = $_('appLock.retryAfter', {
				values: { seconds: Math.ceil(result.retryAfterMs / 1000) }
			});
			return;
		}
		error = $_('appLock.wrongPin');
	}
</script>

<div class="flex min-h-screen flex-col items-center justify-center px-6 py-10">
	<div class="w-full max-w-sm text-center">
		<div class="mb-6 flex justify-center">
			<AppIcon size={56} />
		</div>
		<h1 class="text-xl font-semibold">{$_('appLock.title')}</h1>
		<p class="mt-2 text-sm" style:color="var(--text-muted)">{$_('appLock.subtitle')}</p>

		<div class="mt-8 flex justify-center gap-3" aria-hidden="true">
			{#each [...Array(PIN_LENGTH).keys()] as index (index)}
				<span
					class="inline-block h-3 w-3 rounded-full border"
					style:border-color="var(--border)"
					style:background-color={index < filled ? 'var(--primary)' : 'transparent'}
				></span>
			{/each}
		</div>

		{#if error}
			<p class="mt-4 text-sm" style:color="var(--danger)" role="alert">{error}</p>
		{/if}

		{#if showBiometric}
			<button
				type="button"
				class="btn-ghost mt-4 text-sm"
				disabled={busy || retryBlockedMs() > 0}
				onclick={() => void tryBiometric()}
			>
				{$_('appLock.useBiometric')}
			</button>
		{/if}

		<div class="mt-8 grid grid-cols-3 gap-3">
			{#each keypad as key, index (index)}
				{#if key === ''}
					<div></div>
				{:else if key === 'back'}
					<button
						type="button"
						class="btn-ghost h-14 text-lg"
						disabled={busy || retryBlockedMs() > 0}
						aria-label={$_('appLock.backspace')}
						onclick={backspace}
					>
						⌫
					</button>
				{:else}
					<button
						type="button"
						class="btn-ghost h-14 text-xl font-medium tabular-nums"
						disabled={busy || retryBlockedMs() > 0}
						onclick={() => appendDigit(key)}
					>
						{key}
					</button>
				{/if}
			{/each}
		</div>
	</div>
</div>
