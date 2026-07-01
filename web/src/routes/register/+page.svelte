<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { register } from '$lib/api/client';
	import { user, markSessionHint } from '$lib/stores/auth';
	import { syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import { toast } from '$lib/toast';

	let loginName = $state('');
	let displayName = $state('');
	let password = $state('');
	let passwordConfirm = $state('');
	let loading = $state(false);

	const passwordsMatch = $derived(passwordConfirm.length === 0 || password === passwordConfirm);
	const formValid = $derived(
		loginName.trim().length >= 3 &&
			validatePasswordPolicy(password, loginName) &&
			password === passwordConfirm
	);

	async function submit(e: Event) {
		e.preventDefault();
		if (!formValid) {
			toast.error('Пароли не совпадают или слишком короткие');
			return;
		}
		loading = true;
		try {
			const res = await register(
				loginName.trim(),
				password,
				passwordConfirm,
				displayName.trim() || loginName.trim()
			);
			user.set(res.user);
			markSessionHint();
			setLocale(res.user.language);
			syncThemeFromUser(res.user.theme);
			await goto(resolve('/'));
		} catch (err) {
			toast.fromError(err);
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
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="password-confirm"
					>Подтверждение пароля</label
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
					<p class="mt-1 text-xs" style:color="var(--danger)">Пароли не совпадают</p>
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
