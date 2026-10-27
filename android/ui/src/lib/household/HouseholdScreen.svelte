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
	import { gotoReplace } from '$lib/android/form-nav';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import IconGlyph from '$lib/components/IconGlyph.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import RowActionsMenu from '$lib/components/RowActionsMenu.svelte';
	import { requireOnline } from '$lib/offline/require-online';
	import { toast } from '$lib/toast';
	import { formatInviteCode } from './invite-code';
	import { dataRefreshTick } from '$lib/offline/sync';
	import { resetCachesAfterMembershipChange } from './membership-cache';
	import { householdKindKey, householdMe, refreshHouseholdMe } from './membership';
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

	// No socket on Android: manual sync / pull-to-refresh / online write bump the tick.
	$effect(() => {
		void $dataRefreshTick;
		if (ready) void load({ silent: true });
	});

	function afterMembershipChange() {
		resetCachesAfterMembershipChange();
		// Home scope tabs and nav labels read householdMe — do not wait for the next sync.
		void refreshHouseholdMe().catch(() => {});
	}

	async function onCreate(kind: HouseholdKind) {
		if (!requireOnline() || creating) return;
		creating = true;
		try {
			me = await createHousehold({ kind });
			afterMembershipChange();
			toast(kind === 'team' ? $_('household.created.team') : $_('household.created'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		} finally {
			creating = false;
		}
	}

	async function saveName() {
		if (!requireOnline()) return;
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
		if (!requireOnline() || busy) return;
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

	async function copyText(value: string) {
		try {
			await navigator.clipboard.writeText(value);
			toast($_('household.copied'));
		} catch {
			toast.error($_('common.error'));
		}
	}

	async function shareInvite() {
		if (!createdInvite) return;
		const payload = {
			title: $_('household.invite.title'),
			text: formatInviteCode(createdInvite.code),
			url: createdInvite.url ?? formatInviteCode(createdInvite.code)
		};
		try {
			if (navigator.share) {
				await navigator.share(payload);
			} else {
				await copyText(createdInvite.url ?? formatInviteCode(createdInvite.code));
			}
		} catch {
			/* user cancelled share */
		}
	}

	async function onRevoke(id: string) {
		if (!requireOnline() || busy) return;
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
		if (!requireOnline()) return;
		const ok = await confirm({
			message: $_(kindKey('household.kick.confirm'), { values: { name: displayName } }),
			confirmLabel: $_('household.kick'),
			danger: true
		});
		if (!ok) return;
		try {
			await kickHouseholdMember(userId);
			afterMembershipChange();
			toast($_('household.kicked'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onTransfer(userId: string, displayName: string) {
		if (!requireOnline()) return;
		const ok = await confirm({
			message: $_('household.transfer.confirm', { values: { name: displayName } }),
			confirmLabel: $_('household.transfer')
		});
		if (!ok) return;
		try {
			me = await transferHousehold(userId);
			afterMembershipChange();
			toast($_('household.transferred'));
			await load();
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onLeave() {
		if (!requireOnline()) return;
		const ok = await confirm({
			message: $_(kindKey('household.leave.confirm')),
			confirmLabel: $_(kindKey('household.leave')),
			danger: true
		});
		if (!ok) return;
		try {
			await leaveHousehold();
			householdMe.set({ household: null });
			afterMembershipChange();
			toast($_(kindKey('household.left')));
			await gotoReplace('/');
		} catch (err) {
			toast.error(formatApiError(err));
		}
	}

	async function onDissolve() {
		if (!requireOnline()) return;
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
			householdMe.set({ household: null });
			afterMembershipChange();
			toast($_(kindKey('household.dissolved')));
			await gotoReplace('/');
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
					onclick={() => void goto(resolve('/settings/household/accept'))}
				>
					{$_('household.accept')}
				</button>
			</div>
		</EmptyStateCard>
	{:else}
		<div class="space-y-6">
			<div class="card space-y-3">
				{#if editingName && isOwner}
					<form
						class="flex flex-wrap items-end gap-2"
						onsubmit={(e) => {
							e.preventDefault();
							void saveName();
						}}
					>
						<label class="min-w-48 flex-1 space-y-1">
							<span class="text-sm" style:color="var(--text-muted)">{$_('household.name')}</span>
							<input class="input w-full" bind:value={nameDraft} maxlength="64" required />
						</label>
						<button type="submit" class="btn-primary" disabled={savingName}>
							{$_('common.save')}
						</button>
						<button type="button" class="btn-ghost" onclick={() => (editingName = false)}>
							{$_('common.cancel')}
						</button>
					</form>
				{:else}
					<div class="flex items-start justify-between gap-2">
						<h2 class="text-xl font-semibold">{me.household.name}</h2>
						{#if isOwner}
							<button type="button" class="btn-ghost" onclick={() => (editingName = true)}>
								{$_('common.edit')}
							</button>
						{/if}
					</div>
				{/if}

				<ul class="divide-y" style:border-color="var(--border)">
					{#each members as member (member.user_id)}
						<li class="flex items-center justify-between gap-3 py-3">
							<div>
								<p class="font-medium">{member.display_name}</p>
								<p class="text-sm" style:color="var(--text-muted)">{member.login}</p>
							</div>
							<div class="flex items-center gap-2">
								{#if member.role === 'owner'}
									<span class="rounded-full px-2 py-0.5 text-xs" style:background="var(--muted)"
										>{$_('household.owner')}</span
									>
								{/if}
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
								{/if}
							</div>
						</li>
					{/each}
				</ul>
			</div>

			{#if isOwner}
				<div class="card space-y-3">
					<div class="flex items-center justify-between gap-2">
						<h3 class="flex items-center gap-2 font-semibold">
							{$_('household.invite.title')}
							{#if invites.length > 0}
								<span
									class="rounded-full px-2 py-0.5 text-xs font-medium"
									style:background="var(--muted)">{invites.length}</span
								>
							{/if}
						</h3>
						<button
							type="button"
							class="btn-primary"
							disabled={busy}
							onclick={() => void onCreateInvite()}
						>
							{$_('household.invite.create')}
						</button>
					</div>
					{#if invites.length === 0}
						<p class="text-sm" style:color="var(--text-muted)">{$_('household.invite.empty')}</p>
					{:else}
						<ul
							class="divide-y overflow-hidden rounded-xl border"
							style:border-color="var(--border)"
						>
							{#each invites as invite (invite.id)}
								<li class="flex items-center gap-3 px-3 py-3" style:border-color="var(--border)">
									<button
										type="button"
										class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
										style:background="var(--muted)"
										aria-label={$_('household.invite.showQr')}
										onclick={() => void showInvite(invite)}
									>
										<IconGlyph icon="qr" class="h-5 w-5" />
									</button>
									<div class="min-w-0 flex-1">
										<p class="font-mono text-base font-semibold tracking-wider">
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
					{/if}
				</div>
			{/if}

			<div class="card space-y-2">
				{#if isOwner}
					{#if otherMembers.length > 0}
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('household.ownerCannotLeave')}
						</p>
					{/if}
					<button type="button" class="btn-danger" onclick={() => void onDissolve()}>
						{$_(kindKey('household.dissolve'))}
					</button>
				{:else}
					<button type="button" class="btn-danger" onclick={() => void onLeave()}>
						{$_(kindKey('household.leave'))}
					</button>
				{/if}
			</div>
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
			<p class="font-mono text-2xl tracking-wide">{formatInviteCode(invite.code)}</p>
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
					onclick={() => void copyText(formatInviteCode(invite.code))}
				>
					{$_('household.invite.copyCode')}
				</button>
				{#if invite.url}
					<button type="button" class="btn-ghost" onclick={() => void copyText(invite.url ?? '')}>
						{$_('household.invite.copyLink')}
					</button>
				{/if}
				<button type="button" class="btn-ghost" onclick={() => void shareInvite()}>
					{$_('household.invite.share')}
				</button>
			</div>
		</div>
	{/if}
	{#snippet footer()}
		<button type="button" class="btn-ghost" onclick={() => (inviteOpen = false)}>
			{$_('common.close')}
		</button>
	{/snippet}
</ModalShell>
