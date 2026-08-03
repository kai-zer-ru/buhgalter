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
	const keypad = $derived([
		'1',
		'2',
		'3',
		'4',
		'5',
		'6',
		'7',
		'8',
		'9',
		showBiometric ? 'bio' : '',
		'0',
		'back'
	]);

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

		<div class="mt-8 grid grid-cols-3 gap-3">
			{#each keypad as key, index (index)}
				{#if key === ''}
					<div></div>
				{:else if key === 'bio'}
					<button
						type="button"
						class="btn-ghost flex h-14 items-center justify-center"
						disabled={busy || retryBlockedMs() > 0}
						aria-label={$_('appLock.useBiometric')}
						onclick={() => void tryBiometric()}
					>
						<svg
							aria-hidden="true"
							class="h-6 w-6"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
						>
							<path d="M12 10a2 2 0 0 0-2 2c0 1.02-.1 2.51-.26 4" />
							<path d="M14 13.12c0 2.38 0 6.38-1 8.88" />
							<path d="M17.29 21.02c.12-.6.43-2.3.5-3.02" />
							<path d="M19 13c0 1-.08 3.17-.24 4.5" />
							<path d="M2 12a10 10 0 0 1 18-6" />
							<path d="M2 16.5c.64.98 1.5 1.8 2.5 2.4" />
							<path d="M3.38 17.5c.36.94.97 1.82 1.79 2.5" />
							<path d="M7 11a5 5 0 0 1 10 0" />
							<path d="M8 16c0 1.5.02 3.01.06 4" />
							<path d="M9.5 21c.14-.96.34-2.58.5-4" />
						</svg>
					</button>
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
