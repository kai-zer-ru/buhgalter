<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		createToken,
		deleteToken,
		getUserSettings,
		listTokens,
		type APIToken,
		type APITokenCreated
	} from '$lib/api/client';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import RowActionsMenu from '$lib/components/RowActionsMenu.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { confirm } from '$lib/confirm';
	import {
		apiDateTimeToRFC3339,
		defaultTokenExpiryLocal,
		formatAPIOperationDateTimeForDisplay,
		fromDateLocalEnd
	} from '$lib/dates';
	import { futureDateOnlyPicker } from '$lib/datetime-picker-standards';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);
	let timezone = $state('Europe/Moscow');
	let tokens = $state<APIToken[]>([]);
	let newTokenName = $state('');
	let newTokenExpiresAt = $state('');
	let newTokenNeverExpires = $state(false);
	let createdToken = $state<APITokenCreated | null>(null);

	onMount(() => {
		void reloadPage();
	});

	async function reloadPage() {
		pageLoading = true;
		try {
			const s = await getUserSettings();
			timezone = s.timezone;
			newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
			await loadTokens();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: tokens.length > 0 });
			if (msg) loadError = msg;
		} finally {
			pageLoading = false;
		}
	}

	$effect(() => {
		if (!newTokenExpiresAt && timezone) {
			newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
		}
	});

	async function loadTokens() {
		tokens = await listTokens();
	}

	function formatOptional(value: string | null) {
		return value && value.trim() !== '' ? value : '—';
	}

	function formatTokenExpiry(value: string | null) {
		if (!value || value.trim() === '') {
			return $_('settings.tokens.col.never');
		}
		return formatAPIOperationDateTimeForDisplay(value, timezone);
	}

	async function handleCreateToken(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			const opts = newTokenNeverExpires
				? { neverExpires: true }
				: {
						expiresAt: apiDateTimeToRFC3339(fromDateLocalEnd(newTokenExpiresAt, timezone))
					};
			createdToken = await createToken(newTokenName.trim(), opts);
			newTokenName = '';
			newTokenNeverExpires = false;
			newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
			toast($_('common.saved'));
			await loadTokens();
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function revokeToken(id: string) {
		const ok = await confirm({
			message: $_('settings.tokens.confirm.revoke'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		await deleteToken(id);
		toast($_('common.deleted'));
		await loadTokens();
	}

	function copyToken(token: string) {
		navigator.clipboard.writeText(token);
	}

	function closeTokenModal() {
		createdToken = null;
	}
</script>

<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void reloadPage()}>
	<div class="space-y-8">
		<form class="card space-y-4" onsubmit={handleCreateToken}>
			<div class="space-y-1">
				<h2 class="font-semibold">{$_('settings.tokens.add')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('settings.tokens.add_hint')}</p>
			</div>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="field-label" for="token-name">{$_('settings.tokens.name')}</label>
					<input
						id="token-name"
						class="input w-full"
						bind:value={newTokenName}
						placeholder="Home Assistant"
						required
					/>
				</div>
				<DateTimePicker
					id="token-expires"
					label={$_('settings.tokens.expires')}
					bind:value={newTokenExpiresAt}
					disabled={newTokenNeverExpires}
					required={!newTokenNeverExpires}
					{timezone}
					{...futureDateOnlyPicker}
				/>
			</div>
			<div class="flex items-center justify-between gap-4">
				<span class="text-sm">{$_('settings.tokens.never_expires')}</span>
				<ToggleSwitch
					checked={newTokenNeverExpires}
					label={$_('settings.tokens.never_expires')}
					onchange={() => (newTokenNeverExpires = !newTokenNeverExpires)}
				/>
			</div>
			{#if newTokenNeverExpires}
				<p
					class="rounded-xl border p-3 text-sm font-medium"
					style:color="var(--danger)"
					style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
				>
					{$_('settings.tokens.perpetual_warning')}
				</p>
			{/if}
			<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
				<button type="submit" class="btn-primary w-full sm:w-auto" disabled={loading}>
					{$_('common.create')}
				</button>
			</div>
		</form>

		{#if createdToken}
			{@const newToken = createdToken}
			<ModalShell open={true} title={$_('settings.tokens.created.title')} onclose={closeTokenModal}>
				<p class="mb-4 text-sm" style:color="var(--text-muted)">
					{$_('settings.tokens.created.hint')}
				</p>
				<code
					class="block overflow-x-auto rounded-lg px-3 py-2 text-sm"
					style:background-color="var(--bg)">{newToken.token}</code
				>
				{#snippet footer()}
					<button type="button" class="btn-primary" onclick={() => copyToken(newToken.token)}>
						{$_('settings.tokens.copy')}
					</button>
					<button type="button" class="btn-ghost" onclick={closeTokenModal}
						>{$_('common.close')}</button
					>
				{/snippet}
			</ModalShell>
		{/if}

		{#if tokens.length === 0}
			<EmptyStateCard message={$_('settings.tokens.empty')} />
		{:else}
			<section class="space-y-3">
				<GroupHeading
					title={$_('settings.tokens.group.all', { values: { count: tokens.length } })}
					hint={$_('settings.tokens.group.all_hint')}
				/>
				<div class="card list-divided p-0 sm:p-0">
					{#each tokens as t (t.id)}
						{@const perpetual = !t.expires_at || t.expires_at.trim() === ''}
						<div
							class="flex items-center gap-3 py-2 pr-1.5 pl-3 sm:pr-2 sm:pl-4"
							data-testid="token-row"
						>
							<span class="token-glyph" aria-hidden="true">
								<svg
									class="h-5 w-5"
									viewBox="0 0 24 24"
									fill="none"
									stroke="currentColor"
									stroke-width="2"
									stroke-linecap="round"
									stroke-linejoin="round"
								>
									<circle cx="7.5" cy="15.5" r="3.5" />
									<path d="m10 13 9-9M16 7l2 2M14 9l2 2" />
								</svg>
							</span>
							<div class="min-w-0 flex-1">
								<div class="flex flex-wrap items-center gap-2">
									<span class="truncate font-medium">{t.name}</span>
									<code class="text-xs" style:color="var(--text-muted)">{t.token_prefix}…</code>
									{#if perpetual}
										<span class="badge token-badge-danger">{$_('settings.tokens.col.never')}</span>
									{/if}
								</div>
								<div class="mt-0.5 text-xs" style:color="var(--text-muted)">
									{#if !perpetual}
										{$_('settings.tokens.col.expires')}: {formatTokenExpiry(t.expires_at)} ·
									{/if}
									{$_('settings.tokens.col.last_used')}: {formatOptional(t.last_used_at)}
								</div>
							</div>
							<RowActionsMenu
								actions={[
									{
										icon: 'delete',
										label: $_('common.delete'),
										variant: 'danger',
										onclick: () => void revokeToken(t.id)
									}
								]}
							/>
						</div>
					{/each}
				</div>
			</section>
		{/if}
	</div>
</PageLoadGate>

<style>
	.token-glyph {
		display: inline-flex;
		height: 2.25rem;
		width: 2.25rem;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}

	.token-badge-danger {
		color: var(--danger);
		border-color: color-mix(in srgb, var(--danger) 30%, transparent);
		background-color: color-mix(in srgb, var(--danger) 12%, transparent);
	}
</style>
