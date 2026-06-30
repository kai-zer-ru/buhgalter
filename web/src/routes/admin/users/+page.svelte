<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		createAdminUser,
		deleteAdminUser,
		listAdminUsers,
		resetAdminUserPassword,
		updateAdminUserStatus,
		type AdminUser,
		type UserStatus
	} from '$lib/api/client';
	import { formatAuthUserApiError } from '$lib/auth/api-errors';
	import { user } from '$lib/stores/auth';
	import { confirm } from '$lib/confirm';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import RowActionsMenu, { type RowAction } from '$lib/components/RowActionsMenu.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { toast } from '$lib/toast';
	import { refreshPendingUsersBanner } from '$lib/stores/admin-pending-users';
	import { validatePasswordPolicy } from '$lib/password-policy';

	let users = $state<AdminUser[]>([]);
	let login = $state('');
	let displayName = $state('');
	let password = $state('');
	let passwordConfirm = $state('');
	let isAdmin = $state(false);
	let formError = $state('');
	let listError = $state('');
	let loading = $state(false);

	let resetOpen = $state(false);
	let resetUser = $state<AdminUser | null>(null);
	let resetPassword = $state('');
	let resetPasswordConfirm = $state('');
	let resetError = $state('');
	let resetLoading = $state(false);
	/** Prevents the ?reset= query effect from reopening the modal after cancel. */
	let dismissedResetQuery = $state<string | null>(null);

	let moderateOpen = $state(false);
	let moderateUser = $state<AdminUser | null>(null);
	let moderateError = $state('');
	let moderateLoading = $state(false);
	let dismissedModerateQuery = $state<string | null>(null);

	const passwordsMatch = $derived(passwordConfirm.length === 0 || password === passwordConfirm);
	const formValid = $derived(
		login.trim().length >= 3 &&
			validatePasswordPolicy(password, login) &&
			password === passwordConfirm
	);
	const resetPasswordsMatch = $derived(
		resetPasswordConfirm.length === 0 || resetPassword === resetPasswordConfirm
	);
	const resetFormValid = $derived(
		resetUser !== null &&
			validatePasswordPolicy(resetPassword, resetUser.login) &&
			resetPassword === resetPasswordConfirm
	);

	function statusLabel(status: UserStatus): string {
		switch (status) {
			case 'active':
				return $_('admin.users.status.active');
			case 'pending':
				return $_('admin.users.status.pending');
			case 'banned':
				return $_('admin.users.status.banned');
			default:
				return status;
		}
	}

	function roleLabel(isAdminRole: boolean): string {
		return isAdminRole ? $_('admin.users.roleAdmin') : $_('admin.users.roleUser');
	}

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		users = await listAdminUsers();
		openResetFromQuery();
		openModerateFromQuery();
	});

	$effect(() => {
		const userId = $page.url.searchParams.get('reset');
		if (!userId) {
			dismissedResetQuery = null;
			return;
		}
		if (users.length > 0) {
			openResetFromQuery();
		}
	});

	$effect(() => {
		const userId = $page.url.searchParams.get('moderate');
		if (!userId) {
			dismissedModerateQuery = null;
			return;
		}
		if (users.length > 0) {
			openModerateFromQuery();
		}
	});

	function openResetFromQuery() {
		const userId = $page.url.searchParams.get('reset');
		if (!userId || userId === dismissedResetQuery) return;
		const target = users.find((u) => u.id === userId);
		if (target) {
			openResetPassword(target);
		}
	}

	function openResetPassword(u: AdminUser) {
		resetUser = u;
		resetPassword = '';
		resetPasswordConfirm = '';
		resetError = '';
		resetOpen = true;
	}

	function closeResetPassword() {
		const userId = $page.url.searchParams.get('reset');
		if (userId) {
			dismissedResetQuery = userId;
		}
		resetOpen = false;
		resetUser = null;
		void clearResetQueryParam();
	}

	async function clearResetQueryParam() {
		if (!$page.url.searchParams.has('reset')) return;
		const url = new URL($page.url);
		url.searchParams.delete('reset');
		const search = url.searchParams.toString();
		const adminUsersUrl = search ? `${resolve('/admin/users')}?${search}` : resolve('/admin/users');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		await goto(adminUsersUrl, { replaceState: true, keepFocus: true, noScroll: true });
	}

	function openModerateFromQuery() {
		const userId = $page.url.searchParams.get('moderate');
		if (!userId || userId === dismissedModerateQuery) return;
		const target = users.find((u) => u.id === userId);
		if (target && target.status === 'pending') {
			openModeration(target);
		}
	}

	function openModeration(u: AdminUser) {
		moderateUser = u;
		moderateError = '';
		moderateOpen = true;
	}

	function closeModeration() {
		const userId = $page.url.searchParams.get('moderate');
		if (userId) {
			dismissedModerateQuery = userId;
		}
		moderateOpen = false;
		moderateUser = null;
		void clearModerateQueryParam();
	}

	async function clearModerateQueryParam() {
		if (!$page.url.searchParams.has('moderate')) return;
		const url = new URL($page.url);
		url.searchParams.delete('moderate');
		const search = url.searchParams.toString();
		const adminUsersUrl = search ? `${resolve('/admin/users')}?${search}` : resolve('/admin/users');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		await goto(adminUsersUrl, { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function submitModeration(status: 'active' | 'banned') {
		if (!moderateUser) return;
		if (status === 'banned') {
			const ok = await confirm({
				message: $_('admin.users.confirm.ban', { values: { name: moderateUser.login } }),
				confirmLabel: $_('admin.users.action.ban'),
				danger: true
			});
			if (!ok) return;
		}
		moderateError = '';
		moderateLoading = true;
		try {
			await updateAdminUserStatus(moderateUser.id, status);
			users = await listAdminUsers();
			refreshPendingUsersBanner();
			toast($_('common.saved'));
			closeModeration();
		} catch (err) {
			moderateError = formatAuthUserApiError(err);
		} finally {
			moderateLoading = false;
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		formError = '';
		if (!formValid) {
			formError = $_('admin.users.passwordMismatch');
			return;
		}
		loading = true;
		try {
			await createAdminUser({
				login: login.trim(),
				password,
				password_confirm: passwordConfirm,
				display_name: displayName.trim() || login.trim(),
				is_admin: isAdmin
			});
			login = '';
			displayName = '';
			password = '';
			passwordConfirm = '';
			isAdmin = false;
			users = await listAdminUsers();
			toast($_('common.saved'));
		} catch (err) {
			formError = formatAuthUserApiError(err);
		} finally {
			loading = false;
		}
	}

	async function submitResetPassword() {
		if (!resetUser) return;
		resetError = '';
		if (!resetFormValid) {
			resetError = $_('admin.users.reset.invalid');
			return;
		}
		resetLoading = true;
		try {
			await resetAdminUserPassword(resetUser.id, {
				new_password: resetPassword,
				new_password_confirm: resetPasswordConfirm
			});
			toast($_('common.saved'));
			closeResetPassword();
		} catch (err) {
			resetError = formatAuthUserApiError(err);
		} finally {
			resetLoading = false;
		}
	}

	async function changeStatus(u: AdminUser, status: 'active' | 'banned') {
		if (status === 'banned') {
			const ok = await confirm({
				message: $_('admin.users.confirm.ban', { values: { name: u.login } }),
				confirmLabel: $_('admin.users.action.ban'),
				danger: true
			});
			if (!ok) return;
		}
		listError = '';
		try {
			await updateAdminUserStatus(u.id, status);
			users = await listAdminUsers();
			refreshPendingUsersBanner();
			toast($_('common.saved'));
		} catch (err) {
			listError = formatAuthUserApiError(err);
		}
	}

	async function remove(id: string, name: string) {
		const ok = await confirm({
			message: $_('admin.users.confirm.delete', { values: { name } }),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		listError = '';
		try {
			await deleteAdminUser(id);
			users = await listAdminUsers();
			toast($_('common.deleted'));
		} catch (err) {
			listError = formatAuthUserApiError(err);
		}
	}

	function rowActions(u: AdminUser): RowAction[] {
		const isSelf = u.id === $user?.id;
		const actions: RowAction[] = [];

		if (!isSelf) {
			if (u.status === 'pending') {
				actions.push({
					icon: 'create',
					label: $_('admin.users.action.activate'),
					onclick: () => void changeStatus(u, 'active')
				});
			}
			if (u.status === 'active') {
				actions.push({
					icon: 'archive',
					label: $_('admin.users.action.ban'),
					variant: 'danger',
					onclick: () => void changeStatus(u, 'banned')
				});
			}
			if (u.status === 'banned') {
				actions.push({
					icon: 'save',
					label: $_('admin.users.action.unblock'),
					onclick: () => void changeStatus(u, 'active')
				});
			}
			if (u.status === 'pending') {
				actions.push({
					icon: 'archive',
					label: $_('admin.users.action.ban'),
					variant: 'danger',
					onclick: () => void changeStatus(u, 'banned')
				});
			}
		}

		actions.push({
			icon: 'edit',
			label: $_('admin.users.resetPassword'),
			onclick: () => openResetPassword(u)
		});

		if (!isSelf) {
			actions.push({
				icon: 'delete',
				label: $_('common.delete'),
				variant: 'danger',
				onclick: () => void remove(u.id, u.login)
			});
		}

		return actions;
	}
</script>

<div class="space-y-4">
	<form class="card space-y-4" onsubmit={submit}>
		<h2 class="text-lg font-medium">{$_('admin.users.create.title')}</h2>
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="login">{$_('login.login')}</label>
				<input id="login" class="input" bind:value={login} minlength="3" required />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="display"
					>{$_('register.display_name')}</label
				>
				<input id="display" class="input" bind:value={displayName} />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="password">{$_('login.password')}</label
				>
				<input
					id="password"
					class="input"
					type="password"
					bind:value={password}
					minlength="8"
					autocomplete="new-password"
					required
				/>
				<p class="mt-1 text-xs" style:color="var(--text-muted)">
					{$_('auth.password.requirements')}
				</p>
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="password-confirm"
					>{$_('admin.users.passwordConfirm')}</label
				>
				<input
					id="password-confirm"
					class="input"
					type="password"
					bind:value={passwordConfirm}
					minlength="8"
					autocomplete="new-password"
					required
				/>
				{#if passwordConfirm.length > 0 && !passwordsMatch}
					<p class="mt-1 text-xs" style:color="var(--danger)">
						{$_('admin.users.passwordMismatch')}
					</p>
				{/if}
			</div>
			<div class="flex items-center justify-between gap-4 sm:col-span-2">
				<span class="text-sm">{$_('admin.users.roleAdmin')}</span>
				<ToggleSwitch
					checked={isAdmin}
					label={$_('admin.users.roleAdmin')}
					onchange={() => (isAdmin = !isAdmin)}
				/>
			</div>
		</div>
		<button type="submit" class="btn-primary" disabled={loading || !formValid}>
			{$_('common.create')}
		</button>
		<FormFeedback error={formError} />
	</form>

	<FormFeedback error={listError} />

	<div class="card md:overflow-x-auto">
		<div class="hidden md:block">
			<table class="w-full table-fixed text-left text-sm">
				<colgroup>
					<col />
					<col />
					<col class="w-[11rem]" />
					<col class="w-[9.5rem]" />
					<col class="w-12" />
				</colgroup>
				<thead>
					<tr style:color="var(--text-muted)">
						<th class="p-3">{$_('login.login')}</th>
						<th class="p-3">{$_('register.display_name')}</th>
						<th class="p-3 whitespace-nowrap">{$_('admin.users.role')}</th>
						<th class="p-3 whitespace-nowrap">{$_('admin.users.status')}</th>
						<th class="p-3 w-0"></th>
					</tr>
				</thead>
				<tbody>
					{#each users as u (u.id)}
						<tr class="border-t" style:border-color="var(--border)">
							<td class="p-3 align-middle">{u.login}</td>
							<td class="p-3 align-middle">{u.display_name}</td>
							<td class="p-3 align-middle whitespace-nowrap">{roleLabel(u.is_admin)}</td>
							<td class="p-3 align-middle whitespace-nowrap">{statusLabel(u.status)}</td>
							<td class="p-3 w-0 align-middle text-right whitespace-nowrap">
								<RowActionsMenu actions={rowActions(u)} />
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="space-y-3 md:hidden">
			{#each users as u (u.id)}
				<article class="rounded-xl border p-4" style:border-color="var(--border)">
					<div class="flex items-start justify-between gap-2">
						<p class="font-medium">{u.login}</p>
						<RowActionsMenu actions={rowActions(u)} />
					</div>
					<dl class="mt-2 grid gap-2 text-sm">
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('register.display_name')}</dt>
							<dd>{u.display_name}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('admin.users.role')}</dt>
							<dd>{roleLabel(u.is_admin)}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('admin.users.status')}</dt>
							<dd>{statusLabel(u.status)}</dd>
						</div>
					</dl>
				</article>
			{/each}
		</div>
	</div>
</div>

{#if resetOpen && resetUser}
	<ModalShell
		bind:open={resetOpen}
		title={$_('admin.users.reset.title')}
		onclose={closeResetPassword}
	>
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">
				{$_('admin.users.reset.forUser', { values: { login: resetUser.login } })}
			</p>
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)"
					>{$_('admin.users.reset.newPassword')}</span
				>
				<input
					class="input w-full"
					type="password"
					bind:value={resetPassword}
					minlength="8"
					autocomplete="new-password"
					required
				/>
			</label>
			<label class="block space-y-1">
				<span class="text-sm" style:color="var(--text-muted)"
					>{$_('admin.users.passwordConfirm')}</span
				>
				<input
					class="input w-full"
					type="password"
					bind:value={resetPasswordConfirm}
					minlength="8"
					autocomplete="new-password"
					required
				/>
			</label>
			{#if resetPasswordConfirm.length > 0 && !resetPasswordsMatch}
				<p class="text-sm" style:color="var(--danger)">{$_('admin.users.passwordMismatch')}</p>
			{/if}
			<p class="text-xs" style:color="var(--text-muted)">{$_('auth.password.requirements')}</p>
			<FormFeedback error={resetError} />
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={closeResetPassword}>
				{$_('common.cancel')}
			</button>
			<button
				type="button"
				class="btn-primary"
				disabled={resetLoading || !resetFormValid}
				onclick={() => void submitResetPassword()}
			>
				{resetLoading ? $_('common.loading') : $_('common.save')}
			</button>
		{/snippet}
	</ModalShell>
{/if}

{#if moderateOpen && moderateUser}
	<ModalShell
		bind:open={moderateOpen}
		title={$_('admin.userModeration.title')}
		onclose={closeModeration}
	>
		<div class="space-y-4">
			<p class="text-sm" style:color="var(--text-muted)">
				{$_('admin.userModeration.forUser', {
					values: {
						login: moderateUser.login,
						name: moderateUser.display_name || moderateUser.login
					}
				})}
			</p>
			<FormFeedback error={moderateError} />
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={closeModeration}>
				{$_('common.cancel')}
			</button>
			<button
				type="button"
				class="btn-ghost"
				style:color="var(--danger)"
				disabled={moderateLoading}
				onclick={() => void submitModeration('banned')}
			>
				{moderateLoading ? $_('common.loading') : $_('admin.users.action.ban')}
			</button>
			<button
				type="button"
				class="btn-primary"
				disabled={moderateLoading}
				onclick={() => void submitModeration('active')}
			>
				{moderateLoading ? $_('common.loading') : $_('admin.users.action.activate')}
			</button>
		{/snippet}
	</ModalShell>
{/if}
