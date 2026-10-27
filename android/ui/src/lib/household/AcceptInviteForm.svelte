<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		acceptHouseholdInvite,
		getHouseholdMe,
		previewHouseholdInvite,
		type HouseholdInvitePreview,
		type HouseholdMe
	} from '$lib/api/client';
	import { formatApiError } from '$lib/api/errors';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { requireOnline } from '$lib/offline/require-online';
	import { getServerProfile } from '$lib/platform/server-profile';
	import { getServerUrl } from '$lib/platform/server-url';
	import {
		decodeQrFromImageFile,
		startQrScanner,
		type QrScannerHandle
	} from '$lib/receipt/parse-qr';
	import { toast } from '$lib/toast';
	import { formatInviteCode, isCompleteInviteCode } from './invite-code';
	import { resetCachesAfterMembershipChange } from './membership-cache';
	import { householdKindKey, householdMe } from './membership';
	import { parseInviteQr } from './parse-invite-qr';

	let { initialCode = '' }: { initialCode?: string } = $props();

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let me = $state<HouseholdMe>({ household: null });
	let codeInput = $state('');
	let preview = $state<HouseholdInvitePreview | null>(null);
	let previewError = $state('');
	let previewing = $state(false);
	let joining = $state(false);
	let lastPreviewed = '';
	let scanning = $state(false);
	let scannerHandle: QrScannerHandle | null = null;
	let foreignHost = $state('');
	const qrReaderId = 'household-invite-qr';

	const formatted = $derived(formatInviteCode(codeInput));
	const complete = $derived(isCompleteInviteCode(codeInput));

	async function load() {
		if (!requireOnline()) {
			loadError = $_('offline.onlineOnly');
			loading = false;
			return;
		}
		loading = true;
		loadError = null;
		try {
			me = await getHouseholdMe();
			householdMe.set(me);
		} catch (err) {
			loadError = formatApiError(err);
		} finally {
			loading = false;
		}
	}

	async function runPreview(code: string) {
		if (!isCompleteInviteCode(code) || me.household) return;
		if (code === lastPreviewed || previewing) return;
		previewing = true;
		previewError = '';
		preview = null;
		try {
			preview = await previewHouseholdInvite(code);
			lastPreviewed = code;
		} catch (err) {
			lastPreviewed = code;
			if (err instanceof ApiError && err.code === 'ERR_HOUSEHOLD_ALREADY_MEMBER') {
				me = await getHouseholdMe();
				householdMe.set(me);
				return;
			}
			previewError =
				err instanceof ApiError && err.code === 'ERR_INVITE_INVALID'
					? $_('household.accept.invalid')
					: formatApiError(err);
		} finally {
			previewing = false;
		}
	}

	function applyParsedCode(raw: string) {
		foreignHost = '';
		const parsed = parseInviteQr(raw, getServerUrl(), getServerProfile());
		if (!parsed.ok) {
			if (parsed.reason === 'foreign_host') {
				foreignHost = parsed.host;
				preview = null;
				previewError = '';
				return;
			}
			previewError = $_('household.accept.invalid');
			return;
		}
		codeInput = formatInviteCode(parsed.code);
	}

	async function stopScanner() {
		if (scannerHandle) {
			await scannerHandle.stop().catch(() => undefined);
			scannerHandle = null;
		}
		scanning = false;
	}

	async function toggleScanner() {
		if (!requireOnline()) return;
		if (scanning) {
			await stopScanner();
			return;
		}
		scanning = true;
		previewError = '';
		foreignHost = '';
		await tick();
		try {
			scannerHandle = await startQrScanner(qrReaderId, (text) => {
				void stopScanner().then(() => applyParsedCode(text));
			});
		} catch {
			scanning = false;
			previewError = $_('household.scanError');
		}
	}

	async function onGallery(ev: Event) {
		const input = ev.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		if (!requireOnline()) return;
		const text = await decodeQrFromImageFile(file);
		if (!text) {
			previewError = $_('household.scanError');
			return;
		}
		applyParsedCode(text);
	}

	onMount(() => {
		if (initialCode) codeInput = formatInviteCode(initialCode);
		void load().then(() => {
			if (isCompleteInviteCode(codeInput) && !me.household) {
				void runPreview(codeInput);
			}
		});
	});

	onDestroy(() => {
		void stopScanner();
	});

	$effect(() => {
		if (loading || me.household || foreignHost) return;
		if (complete) void runPreview(codeInput);
		else {
			preview = null;
			previewError = '';
			lastPreviewed = '';
		}
	});

	async function join() {
		if (!requireOnline() || !complete || joining || foreignHost) return;
		joining = true;
		try {
			const joined = await acceptHouseholdInvite(codeInput);
			householdMe.set(joined);
			resetCachesAfterMembershipChange();
			toast($_(householdKindKey('household.joined', joined.household?.kind)));
			await goto(resolve('/settings/household'));
		} catch (err) {
			toast.error(
				err instanceof ApiError && err.code === 'ERR_INVITE_INVALID'
					? $_('household.accept.invalid')
					: formatApiError(err)
			);
		} finally {
			joining = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('household.accept.title')} — {$_('app.title')}</title>
</svelte:head>

<PageLoadGate {loading} error={loadError} onretry={() => void load()}>
	{#if me.household}
		<EmptyStateCard message={$_('household.already', { values: { name: me.household.name } })}>
			<button
				type="button"
				class="btn-primary"
				onclick={() => void goto(resolve('/settings/household'))}
			>
				{$_(householdKindKey('household.goToFamily', me.household?.kind))}
			</button>
		</EmptyStateCard>
	{:else}
		<form
			class="card max-w-lg space-y-4"
			onsubmit={(e) => {
				e.preventDefault();
				void join();
			}}
		>
			<label class="block space-y-1">
				<span class="text-sm font-medium">{$_('household.accept.code')}</span>
				<input
					class="input w-full font-mono tracking-wide"
					value={formatted}
					maxlength="11"
					autocomplete="off"
					spellcheck="false"
					oninput={(e) => {
						foreignHost = '';
						codeInput = formatInviteCode((e.currentTarget as HTMLInputElement).value);
					}}
				/>
			</label>
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn-ghost" onclick={() => void toggleScanner()}>
					{scanning ? $_('household.stopScan') : $_('household.scanQr')}
				</button>
				<label class="btn-ghost cursor-pointer">
					{$_('household.scanFile')}
					<input class="hidden" type="file" accept="image/*" onchange={(e) => void onGallery(e)} />
				</label>
			</div>
			<div
				id={qrReaderId}
				class="overflow-hidden rounded-lg"
				class:hidden={!scanning}
				style:min-height={scanning ? '16rem' : '0'}
			></div>
			{#if foreignHost}
				<p class="text-sm" style:color="var(--danger)">
					{$_('household.foreignHost', { values: { host: foreignHost } })}
				</p>
			{/if}
			{#if previewing}
				<p class="text-sm" style:color="var(--text-muted)">{$_('common.loading')}</p>
			{/if}
			{#if previewError}
				<p class="text-sm" style:color="var(--danger)">{previewError}</p>
			{/if}
			{#if preview}
				<p>
					{$_('household.accept.preview', {
						values: { owner: preview.owner_display_name, name: preview.household_name }
					})}
				</p>
				<button type="submit" class="btn-primary" disabled={joining}>
					{joining ? $_('common.loading') : $_('household.accept.join')}
				</button>
			{/if}
		</form>
	{/if}
</PageLoadGate>
