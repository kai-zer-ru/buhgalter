<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		createHousehold,
		createHouseholdInvite,
		dissolveHousehold,
		getHouseholdMe,
		kickHouseholdMember,
		leaveHousehold,
		listHouseholdInvites,
		renameHousehold,
		revokeHouseholdInvite,
		transferHousehold,
		type HouseholdInvite,
		type HouseholdKind,
		type HouseholdMe
	} from '$lib/api/client';
	import { formatApiError } from '$lib/api/errors';
	import { confirm } from '$lib/confirm';
	import { formatDisplayDateTime } from '$lib/dates';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconGlyph from '$lib/components/IconGlyph.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu from '$lib/components/RowActionsMenu.svelte';
	import { refCacheUpdate } from '$lib/ref-cache';
	import { refCacheUpdateMatches } from '$lib/ref-cache-watch';
	import { toast } from '$lib/toast';
	import { formatInviteCode } from './invite-code';
	import { householdKindKey, householdMe } from './membership';
	import { inviteQrDataUrl } from './qr';

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let me = $state<HouseholdMe>({ household: null });
	let invites = $state<HouseholdInvite[]>([]);
	let editingName = $state(false);
	let nameDraft = $state('');
	let savingName = $state(false);
	let creating = $state(false);
	let inviteOpen = $state(false);
	let createdInvite = $state<HouseholdInvite | null>(null);
	let qrSrc = $state('');
	let busy = $state(false);

	const isOwner = $derived(me.role === 'owner');
	const kindKey = (base: string) => householdKindKey(base, me.household?.kind);
	const members = $derived(me.members ?? []);
	const otherMembers = $derived(members.filter((m) => m.role !== 'owner'));

	const HOUSEHOLD_WATCH_PATHS = ['/api/v1/households/me', '/api/v1/households/me/invites'];
	let ready = false;

	async function load(opts: { silent?: boolean } = {}) {
		if (!opts.silent) loading = true;
		loadError = null;
		try {
			me = await getHouseholdMe();
			householdMe.set(me);
			if (me.household && me.role === 'owner') {
				invites = await listHouseholdInvites();
			} else {
				invites = [];
			}
			// Invite accepted or revoked elsewhere — its QR is no longer valid.
			if (inviteOpen && createdInvite && !invites.some((i) => i.id === createdInvite?.id)) {
				inviteOpen = false;
			}
			if (me.household && !editingName) nameDraft = me.household.name;
		} catch (err) {
			if (opts.silent) toast.fromError(err);
			else loadError = formatApiError(err);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load().then(() => (ready = true));
	});

	// Another member accepted an invite / left (WebSocket invalidate) — refresh in place.
	$effect(() => {
		const update = $refCacheUpdate;
		if (!update || !ready) return;
		if (refCacheUpdateMatches(update, HOUSEHOLD_WATCH_PATHS)) {
			void load({ silent: true });
		}
	});

	async function onCreate(kind: HouseholdKind) {
		if (creating) return;
		creating = true;
		try {
			me = await createHousehold({ kind });
			toast(kind === 'team' ? $_('household.created.team') : $_('household.created'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		} finally {
			creating = false;
		}
	}

	async function saveName() {
		const name = nameDraft.trim();
		if (!name || savingName) return;
		savingName = true;
		try {
			me = await renameHousehold(name);
			toast($_('household.renamed'));
			editingName = false;
		} catch (err) {
			toast.error(formatApiError(err));
		} finally {
			savingName = false;
		}
	}

	async function showInvite(invite: HouseholdInvite) {
		createdInvite = invite;
		qrSrc = invite.url ? await inviteQrDataUrl(invite.url) : '';
		inviteOpen = true;
	}

	async function onCreateInvite() {
		if (busy) return;
		busy = true;
		try {
			const invite = await createHouseholdInvite();
			await showInvite(invite);
			toast($_('household.invite.created'));
			invites = await listHouseholdInvites();
		} catch (err) {
			toast.error(formatApiError(err));
		} finally {
			busy = false;
		}
	}

	async function copyText(value: string, okKey: string) {
		try {
			await navigator.clipboard.writeText(value);
			toast($_(okKey));
		} catch {
			toast.error($_('common.error'));
		}
	}

	async function onRevoke(id: string) {
		if (busy) return;
		busy = true;
		try {
			await revokeHouseholdInvite(id);
			toast($_('household.invite.revoked'));
			invites = await listHouseholdInvites();
		} catch (err) {
			toast.error(formatApiError(err));
		} finally {
			busy = false;
		}
	}

	async function onKick(userId: string, displayName: string) {
		const ok = await confirm({
			message: $_(kindKey('household.kick.confirm'), { values: { name: displayName } }),
			confirmLabel: $_('household.kick'),
			danger: true
		});
		if (!ok) return;
		try {
			await kickHouseholdMember(userId);
			toast($_('household.kicked'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onTransfer(userId: string, displayName: string) {
		const ok = await confirm({
			message: $_('household.transfer.confirm', { values: { name: displayName } }),
			confirmLabel: $_('household.transfer')
		});
		if (!ok) return;
		try {
			me = await transferHousehold(userId);
			toast($_('household.transferred'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onLeave() {
		const ok = await confirm({
			message: $_(kindKey('household.leave.confirm')),
			confirmLabel: $_(kindKey('household.leave')),
			danger: true
		});
		if (!ok) return;
		try {
			await leaveHousehold();
			toast($_(kindKey('household.left')));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onDissolve() {
		const first = await confirm({
			message: $_(kindKey('household.dissolve.confirm')),
			confirmLabel: $_(kindKey('household.dissolve')),
			danger: true
		});
		if (!first) return;
		let confirmMembers = false;
		if (otherMembers.length > 0) {
			const second = await confirm({
				message: $_(kindKey('household.dissolve.confirmMembers'), {
					values: { count: String(otherMembers.length) }
				}),
				confirmLabel: $_(kindKey('household.dissolve')),
				danger: true
			});
			if (!second) return;
			confirmMembers = true;
		}
		try {
			await dissolveHousehold(confirmMembers);
			toast($_(kindKey('household.dissolved')));
			await load();
		} catch (err) {
			if (err instanceof ApiError && err.code === 'ERR_HOUSEHOLD_HAS_MEMBERS') {
				toast.error(formatApiError(err));
				return;
			}
			toast.error(formatApiError(err));
		}
	}

	function inviteUntil(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		return formatDisplayDateTime(d);
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()}>
	{#if !me.household}
		<EmptyStateCard message={$_('household.empty')}>
			<div class="flex flex-wrap justify-center gap-2">
				<button
					type="button"
					class="btn-primary"
					data-testid="household-create-family"
					disabled={creating}
					onclick={() => void onCreate('family')}
				>
					{creating ? $_('common.loading') : $_('household.create.family')}
				</button>
				<button
					type="button"
					class="btn-primary"
					data-testid="household-create-team"
					disabled={creating}
					onclick={() => void onCreate('team')}
				>
					{creating ? $_('common.loading') : $_('household.create.team')}
				</button>
				<button
					type="button"
					class="btn-ghost"
					data-testid="household-accept-link"
					onclick={() => void goto(resolve('/settings/household/accept'))}
				>
					{$_('household.accept')}
				</button>
			</div>
		</EmptyStateCard>
	{:else}
		<div class="space-y-8">
			<div class="card">
				{#if editingName && isOwner}
					<form
						class="flex flex-wrap items-end gap-2"
						onsubmit={(e) => {
							e.preventDefault();
							void saveName();
						}}
					>
						<label class="min-w-48 flex-1 space-y-1">
							<span class="field-label">{$_('household.name')}</span>
							<input class="input w-full" bind:value={nameDraft} maxlength="64" required />
						</label>
						<button type="button" class="btn-ghost" onclick={() => (editingName = false)}>
							{$_('common.cancel')}
						</button>
						<button type="submit" class="btn-primary" disabled={savingName}>
							{$_('common.save')}
						</button>
					</form>
				{:else}
					<div class="flex items-center gap-4">
						<span class="household-glyph" aria-hidden="true">
							<IconGlyph icon="share" class="h-6 w-6" />
						</span>
						<div class="min-w-0 flex-1">
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="truncate text-xl font-semibold">{me.household.name}</h2>
								<span class="badge badge-success">{$_(kindKey('household.badge'))}</span>
							</div>
							<p class="text-sm" style:color="var(--text-muted)">
								{$_('household.summary', { values: { count: members.length } })}
							</p>
						</div>
						{#if isOwner}
							<button type="button" class="btn-ghost shrink-0" onclick={() => (editingName = true)}>
								{$_('common.edit')}
							</button>
						{/if}
					</div>
				{/if}
			</div>

			<section class="space-y-3">
				<GroupHeading
					title={$_('household.members.title', { values: { count: members.length } })}
					hint={isOwner ? $_('household.members.hint_owner') : ''}
				/>
				<ul class="card list-divided p-0 sm:p-0">
					{#each members as member (member.user_id)}
						<li
							class="flex items-center gap-3 py-2 pr-1.5 pl-3 sm:pr-2 sm:pl-4"
							data-testid="household-member-row"
						>
							<span class="member-avatar" aria-hidden="true"
								>{(member.display_name || member.login).slice(0, 1).toUpperCase()}</span
							>
							<div class="min-w-0 flex-1">
								<div class="flex flex-wrap items-center gap-2">
									<span class="truncate font-medium">{member.display_name}</span>
									{#if member.role === 'owner'}
										<span class="badge badge-success">{$_('household.owner')}</span>
									{/if}
								</div>
								<p class="truncate text-xs" style:color="var(--text-muted)">{member.login}</p>
							</div>
							{#if isOwner && member.role === 'member'}
								<RowActionsMenu
									actions={[
										{
											icon: 'edit',
											label: $_('household.transfer'),
											onclick: () => void onTransfer(member.user_id, member.display_name)
										},
										{
											icon: 'delete',
											label: $_('household.kick'),
											variant: 'danger',
											onclick: () => void onKick(member.user_id, member.display_name)
										}
									]}
								/>
							{:else}
								<span class="min-h-11" aria-hidden="true"></span>
							{/if}
						</li>
					{/each}
				</ul>
			</section>

			{#if isOwner}
				<section class="space-y-3">
					<div class="flex flex-wrap items-end justify-between gap-3">
						<GroupHeading
							title={$_('household.invites.title', { values: { count: invites.length } })}
							hint={$_('household.invites.hint')}
						/>
						<button
							type="button"
							class="btn-primary"
							data-testid="household-invite-create"
							disabled={busy}
							onclick={() => void onCreateInvite()}
						>
							{$_('household.invite.create')}
						</button>
					</div>
					<ul class="card list-divided p-0 sm:p-0">
						{#if invites.length === 0}
							<li class="px-4 py-6 text-center text-sm" style:color="var(--text-muted)">
								{$_('household.invite.empty')}
							</li>
						{/if}
						{#each invites as invite (invite.id)}
							<li
								class="flex items-center gap-3 py-2 pr-1.5 pl-3 sm:pr-2 sm:pl-4"
								data-testid="household-invite-row"
							>
								<button
									type="button"
									class="invite-qr"
									aria-label={$_('household.invite.showQr')}
									onclick={() => void showInvite(invite)}
								>
									<IconGlyph icon="qr" class="h-5 w-5" />
								</button>
								<div class="min-w-0 flex-1">
									<p
										class="font-mono text-base font-semibold tracking-wider"
										data-testid="household-invite-code"
									>
										{formatInviteCode(invite.code)}
									</p>
									<p class="text-xs" style:color="var(--text-muted)">
										{$_('household.invite.until', {
											values: { date: inviteUntil(invite.expires_at) }
										})}
									</p>
								</div>
								<RowActionsMenu
									actions={[
										{
											icon: 'qr',
											label: $_('household.invite.showQr'),
											onclick: () => void showInvite(invite)
										},
										{
											icon: 'delete',
											label: $_('household.invite.revoke'),
											variant: 'danger',
											onclick: () => void onRevoke(invite.id)
										}
									]}
								/>
							</li>
						{/each}
					</ul>
				</section>
			{/if}

			<section class="space-y-3">
				<GroupHeading title={$_('settings.profile.section.danger')} />
				<div
					class="card flex flex-wrap items-center justify-between gap-3"
					style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
				>
					<p class="min-w-0 flex-1 text-sm" style:color="var(--text-muted)">
						{#if isOwner}
							{otherMembers.length > 0
								? $_('household.ownerCannotLeave')
								: $_(kindKey('household.dissolve.hint'))}
						{:else}
							{$_(kindKey('household.leave.hint'))}
						{/if}
					</p>
					{#if isOwner}
						<button type="button" class="btn-danger" onclick={() => void onDissolve()}>
							{$_(kindKey('household.dissolve'))}
						</button>
					{:else}
						<button type="button" class="btn-danger" onclick={() => void onLeave()}>
							{$_(kindKey('household.leave'))}
						</button>
					{/if}
				</div>
			</section>
		</div>
	{/if}
</PageLoadGate>

<ModalShell
	bind:open={inviteOpen}
	title={$_('household.invite.title')}
	onclose={() => (inviteOpen = false)}
>
	{#if createdInvite}
		{@const invite = createdInvite}
		<div class="space-y-4 text-center">
			<p class="font-mono text-2xl tracking-wide" data-testid="household-invite-dialog-code">
				{formatInviteCode(invite.code)}
			</p>
			{#if qrSrc}
				<img class="mx-auto" src={qrSrc} alt={$_('household.invite.qr')} width="240" height="240" />
			{/if}
			<p class="text-sm" style:color="var(--text-muted)">
				{$_('household.invite.until', { values: { date: inviteUntil(invite.expires_at) } })}
			</p>
			<div class="flex flex-wrap justify-center gap-2">
				<button
					type="button"
					class="btn-primary"
					onclick={() => void copyText(formatInviteCode(invite.code), 'household.copied')}
				>
					{$_('household.invite.copyCode')}
				</button>
				{#if invite.url}
					<button
						type="button"
						class="btn-ghost"
						onclick={() => void copyText(invite.url ?? '', 'household.copied')}
					>
						{$_('household.invite.copyLink')}
					</button>
				{/if}
			</div>
		</div>
	{/if}
	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={() => (inviteOpen = false)}>
			{$_('common.close')}
		</button>
	{/snippet}
</ModalShell>

<style>
	.household-glyph,
	.member-avatar,
	.invite-qr {
		display: inline-flex;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 0.75rem;
		color: var(--primary);
		background-color: color-mix(in srgb, var(--primary) 12%, transparent);
	}

	.household-glyph {
		height: 3rem;
		width: 3rem;
	}

	.member-avatar,
	.invite-qr {
		height: 2.5rem;
		width: 2.5rem;
		font-weight: 600;
	}

	.member-avatar {
		border-radius: 9999px;
	}
</style>
