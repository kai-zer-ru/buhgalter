<script lang="ts">
	import { onMount } from 'svelte';
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
	import { toast } from '$lib/toast';
	import { appInviteUrl, formatInviteCode, isCompleteInviteCode } from './invite-code';
	import { householdKindKey, householdMe } from './membership';

	let {
		initialCode = '',
		showOpenInApp = false
	}: { initialCode?: string; showOpenInApp?: boolean } = $props();

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let me = $state<HouseholdMe>({ household: null });
	let codeInput = $state('');
	let preview = $state<HouseholdInvitePreview | null>(null);
	let previewError = $state('');
	let previewing = $state(false);
	let joining = $state(false);
	let lastPreviewed = '';
	let mobile = $state(false);

	const formatted = $derived(formatInviteCode(codeInput));
	const complete = $derived(isCompleteInviteCode(codeInput));

	async function load() {
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

	onMount(() => {
		mobile = /Mobi|Android|iPhone|iPad/i.test(navigator.userAgent);
		if (initialCode) codeInput = formatInviteCode(initialCode);
		void load().then(() => {
			if (isCompleteInviteCode(codeInput) && !me.household) {
				void runPreview(codeInput);
			}
		});
	});

	$effect(() => {
		if (loading || me.household) return;
		if (complete) void runPreview(codeInput);
		else {
			preview = null;
			previewError = '';
			lastPreviewed = '';
		}
	});

	async function join() {
		if (!complete || joining) return;
		joining = true;
		try {
			const joined = await acceptHouseholdInvite(codeInput);
			householdMe.set(joined);
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
					data-testid="household-accept-code"
					value={formatted}
					maxlength="11"
					autocomplete="off"
					spellcheck="false"
					oninput={(e) => {
						codeInput = formatInviteCode((e.currentTarget as HTMLInputElement).value);
					}}
				/>
			</label>
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
				<button type="submit" class="btn-primary" disabled={joining} data-testid="household-join">
					{joining ? $_('common.loading') : $_('household.accept.join')}
				</button>
			{/if}
			{#if showOpenInApp && mobile && complete}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Android deep link, not an app route -->
				<a class="btn-ghost inline-flex" href={appInviteUrl(codeInput)}>
					{$_('household.openInApp')}
				</a>
			{/if}
		</form>
	{/if}
</PageLoadGate>
