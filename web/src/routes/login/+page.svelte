<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, getRegistrationEnabled, login } from '$lib/api/client';
	import { user, markSessionHint } from '$lib/stores/auth';
	import { syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';

	let loginName = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);
	let registrationEnabled = $state(false);

	onMount(async () => {
		registrationEnabled = await getRegistrationEnabled();
	});

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			const res = await login(loginName.trim(), password);
			user.set(res.user);
			markSessionHint();
			setLocale(res.user.language);
			syncThemeFromUser(res.user.theme);
			await goto(resolve('/'));
		} catch (err) {
			error = err instanceof ApiError ? err.message : $_('common.error');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('login.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-6 text-2xl font-semibold">{$_('login.title')}</h1>
		<form class="space-y-4" onsubmit={submit}>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="login">{$_('login.login')}</label>
				<input id="login" class="input" bind:value={loginName} autocomplete="username" required />
			</div>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="password">{$_('login.password')}</label
				>
				<input
					id="password"
					type="password"
					class="input"
					bind:value={password}
					autocomplete="current-password"
					required
				/>
			</div>
			{#if error}
				<p class="text-sm" style:color="var(--danger)">{error}</p>
			{/if}
			<button type="submit" class="btn-primary w-full" disabled={loading}>
				{loading ? $_('common.loading') : $_('login.submit')}
			</button>
		</form>
		{#if registrationEnabled}
			<p class="mt-6 text-center text-sm" style:color="var(--text-muted)">
				{$_('login.no_account')}
				<a href={resolve('/register')} class="font-medium" style:color="var(--primary)"
					>{$_('login.register')}</a
				>
			</p>
		{/if}
	</div>
</div>
