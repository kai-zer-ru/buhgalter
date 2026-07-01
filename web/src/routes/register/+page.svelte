<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { register } from '$lib/api/client';
	import { formatAuthUserApiError, authUserApiField } from '$lib/auth/api-errors';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import { toast } from '$lib/toast';

	let loginName = $state('');
	let displayName = $state('');
	let password = $state('');
	let passwordConfirm = $state('');
	let loginError = $state('');
	let passwordError = $state('');
	let loading = $state(false);

	const passwordsMatch = $derived(passwordConfirm.length === 0 || password === passwordConfirm);
	const formValid = $derived(
		loginName.trim().length >= 3 &&
			validatePasswordPolicy(password, loginName) &&
			password === passwordConfirm
	);

	async function submit(e: Event) {
		e.preventDefault();
		loginError = '';
		passwordError = '';
		if (!formValid) {
			toast.error($_('admin.users.passwordMismatch'));
			return;
		}
		loading = true;
		try {
			await register(
				loginName.trim(),
				password,
				passwordConfirm,
				displayName.trim() || loginName.trim()
			);
			toast($_('register.pending'));
			await goto(resolve('/login'));
		} catch (err) {
			const message = formatAuthUserApiError(err, 'common.error', false);
			const field = authUserApiField(err);
			if (field === 'login') loginError = message;
			else if (field === 'password') passwordError = message;
			else toast.error(message);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('register.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-6 text-2xl font-semibold">{$_('register.title')}</h1>
		<form class="space-y-4" onsubmit={submit}>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="login">{$_('login.login')}</label>
				<input id="login" class="input" bind:value={loginName} minlength="3" required />
				{#if loginError}
					<p class="mt-1 text-xs" style:color="var(--danger)">{loginError}</p>
				{/if}
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
					type="password"
					class="input"
					bind:value={password}
					minlength="8"
					autocomplete="new-password"
					required
				/>
				<p class="mt-1 text-xs" style:color="var(--text-muted)">
					{$_('auth.password.requirements')}
				</p>
				{#if passwordError}
					<p class="mt-1 text-xs" style:color="var(--danger)">{passwordError}</p>
				{/if}
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="password-confirm"
					>{$_('admin.users.passwordConfirm')}</label
				>
				<input
					id="password-confirm"
					type="password"
					class="input"
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
			<button type="submit" class="btn-primary w-full" disabled={loading || !formValid}>
				{loading ? $_('common.loading') : $_('register.submit')}
			</button>
		</form>
		<p class="mt-6 text-center text-sm" style:color="var(--text-muted)">
			<a href={resolve('/login')} style:color="var(--primary)">{$_('login.title')}</a>
		</p>
	</div>
</div>
