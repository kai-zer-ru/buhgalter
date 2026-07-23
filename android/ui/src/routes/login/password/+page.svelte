<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { login, requestPasswordReset } from '$lib/api/client';
	import { formatAuthUserApiError } from '$lib/auth/api-errors';
	import { user, markSessionHint, persistLastUser } from '$lib/stores/auth';
	import { clearAuthToken, setAuthToken } from '$lib/platform/auth-token';
	import { hasServerUrl } from '$lib/platform/server-url';
	import { syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import { toast } from '$lib/toast';

	let loginName = $state('');
	let password = $state('');
	let loading = $state(false);
	let resetOpen = $state(false);
	let resetLogin = $state('');
	let resetLoading = $state(false);
	let resetSent = $state(false);

	onMount(async () => {
		if (!hasServerUrl()) {
			await goto(resolve('/server-setup'));
		}
	});

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			await clearAuthToken();
			const res = await login(loginName.trim(), password);
			await setAuthToken(res.token, 'session');
			user.set(res.user);
			persistLastUser(res.user);
			markSessionHint();
			setLocale(res.user.language);
			syncThemeFromUser(res.user.theme);
			await goto(resolve('/'));
		} catch (err) {
			await clearAuthToken();
			toast.error(formatAuthUserApiError(err));
		} finally {
			loading = false;
		}
	}

	function openResetRequest() {
		resetLogin = loginName.trim();
		resetSent = false;
		resetOpen = true;
	}

	async function submitResetRequest() {
		const name = resetLogin.trim();
		if (name.length < 3) {
			toast.error($_('login.reset.loginRequired'));
			return;
		}
		resetLoading = true;
		try {
			await requestPasswordReset(name);
			resetSent = true;
			toast($_('login.reset.sent'));
		} catch (err) {
			toast.error(formatAuthUserApiError(err));
		} finally {
			resetLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('login.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-2 text-2xl font-semibold">{$_('login.title')}</h1>
		<p class="mb-6 text-sm" style:color="var(--text-muted)">{$_('androidAuth.method.password')}</p>
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
			<button type="submit" class="btn-primary w-full" disabled={loading}>
				{loading ? $_('common.loading') : $_('login.submit')}
			</button>
		</form>
		<p class="mt-4 text-center text-sm">
			<button
				type="button"
				class="font-medium hover:underline"
				style:color="var(--primary)"
				onclick={openResetRequest}
			>
				{$_('login.reset.request')}
			</button>
		</p>
		<p class="mt-4 text-center text-sm" style:color="var(--text-muted)">
			<a href={resolve('/login')} class="hover:underline" style:color="var(--primary)">
				{$_('androidAuth.backToMethods')}
			</a>
		</p>
	</div>

	<ModalShell
		bind:open={resetOpen}
		title={$_('login.reset.title')}
		onclose={() => (resetOpen = false)}
	>
		<div class="space-y-4">
			{#if resetSent}
				<p class="text-sm" style:color="var(--text-muted)">{$_('login.reset.sentHint')}</p>
			{:else}
				<p class="text-sm" style:color="var(--text-muted)">{$_('login.reset.hint')}</p>
				<label class="block space-y-1">
					<span class="text-sm" style:color="var(--text-muted)">{$_('login.login')}</span>
					<input class="input w-full" bind:value={resetLogin} autocomplete="username" required />
				</label>
			{/if}
		</div>
		{#snippet footer()}
			<button type="button" class="btn-ghost" onclick={() => (resetOpen = false)}>
				{$_('common.close')}
			</button>
			{#if !resetSent}
				<button
					type="button"
					class="btn-primary"
					disabled={resetLoading}
					onclick={() => void submitResetRequest()}
				>
					{resetLoading ? $_('common.loading') : $_('login.reset.submit')}
				</button>
			{/if}
		{/snippet}
	</ModalShell>
</div>
