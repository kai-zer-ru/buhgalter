<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ApiError,
		createAdminUser,
		deleteAdminUser,
		listAdminUsers,
		type AdminUser
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { confirm } from '$lib/confirm';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
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

	const passwordsMatch = $derived(passwordConfirm.length === 0 || password === passwordConfirm);
	const formValid = $derived(
		login.trim().length >= 3 &&
			validatePasswordPolicy(password, login) &&
			password === passwordConfirm
	);

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		users = await listAdminUsers();
	});

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

<form class="card mb-6 space-y-4" onsubmit={submit}>
	<h2 class="text-lg font-medium">Создать пользователя</h2>
	<div class="grid gap-4 sm:grid-cols-2">
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="login">Логин</label>
			<input id="login" class="input" bind:value={login} minlength="3" required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="display">Имя</label>
			<input id="display" class="input" bind:value={displayName} />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="password">Пароль</label>
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
				>Подтверждение пароля</label
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
				<p class="mt-1 text-xs" style:color="var(--danger)">Пароли не совпадают</p>
			{/if}
		</div>
		<label class="flex items-center gap-2 text-sm sm:col-span-2">
			<input type="checkbox" bind:checked={isAdmin} />
			Администратор
		</label>
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
					<th class="pb-3 pr-4">Логин</th>
					<th class="pb-3 pr-4">Имя</th>
					<th class="pb-3 pr-4">Роль</th>
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
							{#if u.id !== $user?.id}
								<button type="button" class="btn-ghost" onclick={() => remove(u.id, u.login)}>
									{$_('common.delete')}
								</button>
							{/if}
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
						<dt style:color="var(--text-muted)">Имя</dt>
						<dd>{u.display_name}</dd>
					</div>
					<div class="flex justify-between gap-2">
						<dt style:color="var(--text-muted)">Роль</dt>
						<dd>{u.is_admin ? 'admin' : 'user'}</dd>
					</div>
				</dl>
				{#if u.id !== $user?.id}
					<button type="button" class="btn-ghost mt-3 w-full" onclick={() => remove(u.id, u.login)}>
						{$_('common.delete')}
					</button>
				{/if}
			</article>
		{/each}
	</div>
</div>
