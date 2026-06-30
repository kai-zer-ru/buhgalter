<script lang="ts">
	import { onMount } from 'svelte';
	import { goto, replaceState } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		createAdminUser,
		deleteAdminUser,
		listAdminUsers,
		resetAdminUserPassword,
		type AdminUser
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { confirm } from '$lib/confirm';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { toast } from '$lib/toast';
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

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		users = await listAdminUsers();
		openResetFromQuery();
	});

	$effect(() => {
		if (users.length > 0 && $page.url.searchParams.get('reset')) {
			openResetFromQuery();
		}
	});

	function openResetFromQuery() {
		const userId = $page.url.searchParams.get('reset');
		if (!userId) return;
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
		resetOpen = false;
		resetUser = null;
		if ($page.url.searchParams.has('reset')) {
			const url = new URL($page.url);
			url.searchParams.delete('reset');
			const search = url.searchParams.toString();
			const adminUsersUrl = search
				? `${resolve('/admin/users')}?${search}`
				: resolve('/admin/users');
			// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
			replaceState(adminUsersUrl, {});
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		formError = '';
		if (!formValid) {
			formError = 'Пароли не совпадают или слишком короткие';
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
			formError = err instanceof ApiError ? err.message : $_('common.error');
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
			resetError = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			resetLoading = false;
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
			listError = err instanceof ApiError ? err.message : $_('common.error');
		}
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
			<table class="w-full text-left text-sm">
				<thead>
					<tr style:color="var(--text-muted)">
						<th class="pb-3 pr-4">{$_('login.login')}</th>
						<th class="pb-3 pr-4">{$_('register.display_name')}</th>
						<th class="pb-3 pr-4">{$_('admin.users.role')}</th>
						<th class="pb-3"></th>
					</tr>
				</thead>
				<tbody>
					{#each users as u (u.id)}
						<tr class="border-t" style:border-color="var(--border)">
							<td class="py-3 pr-4">{u.login}</td>
							<td class="py-3 pr-4">{u.display_name}</td>
							<td class="py-3 pr-4">{u.is_admin ? 'admin' : 'user'}</td>
							<td class="py-3 text-right">
								<div class="flex justify-end gap-2">
									<button type="button" class="btn-ghost" onclick={() => openResetPassword(u)}>
										{$_('admin.users.resetPassword')}
									</button>
									{#if u.id !== $user?.id}
										<button type="button" class="btn-ghost" onclick={() => remove(u.id, u.login)}>
											{$_('common.delete')}
										</button>
									{/if}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="space-y-3 md:hidden">
			{#each users as u (u.id)}
				<article class="rounded-xl border p-4" style:border-color="var(--border)">
					<p class="font-medium">{u.login}</p>
					<dl class="mt-2 grid gap-2 text-sm">
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('register.display_name')}</dt>
							<dd>{u.display_name}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('admin.users.role')}</dt>
							<dd>{u.is_admin ? 'admin' : 'user'}</dd>
						</div>
					</dl>
					<div class="mt-3 flex flex-col gap-2">
						<button type="button" class="btn-ghost w-full" onclick={() => openResetPassword(u)}>
							{$_('admin.users.resetPassword')}
						</button>
						{#if u.id !== $user?.id}
							<button type="button" class="btn-ghost w-full" onclick={() => remove(u.id, u.login)}>
								{$_('common.delete')}
							</button>
						{/if}
					</div>
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
