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
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { confirm } from '$lib/confirm';
	import {
		apiDateTimeToRFC3339,
		defaultTokenExpiryLocal,
		formatAPIOperationDateTimeForDisplay,
		fromDateLocalEnd
	} from '$lib/dates';
	import { futureDateOnlyPicker } from '$lib/datetime-picker-standards';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let timezone = $state('Europe/Moscow');
	let tokens = $state<APIToken[]>([]);
	let newTokenName = $state('');
	let newTokenExpiresAt = $state('');
	let newTokenNeverExpires = $state(false);
	let createdToken = $state<APITokenCreated | null>(null);

	onMount(() => {
		void (async () => {
			try {
				const s = await getUserSettings();
				timezone = s.timezone;
				newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
				await loadTokens();
			} catch (err) {
				toast.fromError(err);
			}
		})();
	});

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

<div class="space-y-6">
	<form class="card space-y-4" onsubmit={handleCreateToken}>
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="token-name"
					>{$_('settings.tokens.name')}</label
				>
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
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<label class="flex cursor-pointer items-center gap-3">
				<ToggleSwitch
					checked={newTokenNeverExpires}
					label={$_('settings.tokens.never_expires')}
					onchange={() => (newTokenNeverExpires = !newTokenNeverExpires)}
				/>
				<span class="text-sm">{$_('settings.tokens.never_expires')}</span>
			</label>
			<button type="submit" class="btn-primary w-full sm:w-auto" disabled={loading}>
				{$_('common.create')}
			</button>
		</div>
		{#if newTokenNeverExpires}
			<p class="text-sm font-medium" style:color="var(--danger)">
				{$_('settings.tokens.perpetual_warning')}
			</p>
		{/if}
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

	<div class="card md:overflow-x-auto">
		<div class="hidden md:block">
			<table class="w-full text-left text-sm">
				<thead>
					<tr style:color="var(--text-muted)">
						<th class="pb-3 pr-4">{$_('settings.tokens.col.name')}</th>
						<th class="pb-3 pr-4">Prefix</th>
						<th class="pb-3 pr-4">{$_('settings.tokens.col.expires')}</th>
						<th class="pb-3 pr-4">{$_('settings.tokens.col.last_used')}</th>
						<th class="pb-3"></th>
					</tr>
				</thead>
				<tbody>
					{#each tokens as t (t.id)}
						<tr class="border-t" style:border-color="var(--border)">
							<td class="py-3 pr-4">{t.name}</td>
							<td class="py-3 pr-4 font-mono">{t.token_prefix}</td>
							<td class="py-3 pr-4">{formatTokenExpiry(t.expires_at)}</td>
							<td class="py-3 pr-4">{formatOptional(t.last_used_at)}</td>
							<td class="py-3 text-right">
								<button type="button" class="btn-ghost" onclick={() => revokeToken(t.id)}>
									{$_('common.delete')}
								</button>
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="5" class="py-4" style:color="var(--text-muted)"
								>{$_('settings.tokens.empty')}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="space-y-3 md:hidden">
			{#each tokens as t (t.id)}
				<article class="rounded-xl border p-4" style:border-color="var(--border)">
					<p class="font-medium">{t.name}</p>
					<dl class="mt-2 grid gap-2 text-sm">
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">Prefix</dt>
							<dd class="font-mono">{t.token_prefix}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('settings.tokens.col.expires')}</dt>
							<dd>{formatTokenExpiry(t.expires_at)}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('settings.tokens.col.last_used')}</dt>
							<dd>{formatOptional(t.last_used_at)}</dd>
						</div>
					</dl>
					<button type="button" class="btn-ghost mt-3 w-full" onclick={() => revokeToken(t.id)}>
						{$_('common.delete')}
					</button>
				</article>
			{:else}
				<p class="py-4 text-sm" style:color="var(--text-muted)">{$_('settings.tokens.empty')}</p>
			{/each}
		</div>
	</div>
</div>
