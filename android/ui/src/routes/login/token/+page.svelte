<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { ApiError, getMe } from '$lib/api/client';
	import { formatAuthUserApiError } from '$lib/auth/api-errors';
	import { user, markSessionHint, persistLastUser } from '$lib/stores/auth';
	import { clearAuthToken, setAuthToken } from '$lib/platform/auth-token';
	import { hasServerUrl } from '$lib/platform/server-url';
	import { syncThemeFromUser } from '$lib/stores/theme';
	import { setLocale } from '$lib/i18n';
	import { toast } from '$lib/toast';

	let apiToken = $state('');
	let loading = $state(false);

	onMount(async () => {
		if (!hasServerUrl()) {
			await goto(resolve('/server-setup'));
		}
	});

	async function submit(e: Event) {
		e.preventDefault();
		const token = apiToken.trim();
		if (!token) {
			toast.error($_('androidAuth.tokenRequired'));
			return;
		}
		loading = true;
		try {
			await setAuthToken(token, 'api_token');
			const me = await getMe();
			user.set(me);
			persistLastUser(me);
			markSessionHint();
			setLocale(me.language);
			syncThemeFromUser(me.theme);
			await goto(resolve('/'));
		} catch (err) {
			await clearAuthToken();
			if (err instanceof ApiError && err.status === 401) {
				toast.error($_('androidAuth.invalidToken'));
			} else {
				toast.error(formatAuthUserApiError(err));
			}
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('androidAuth.title')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-2 text-2xl font-semibold">{$_('androidAuth.title')}</h1>
		<p class="mb-6 text-sm" style:color="var(--text-muted)">{$_('androidAuth.subtitle')}</p>
		<form class="space-y-4" onsubmit={submit}>
			<div>
				<label class="mb-1.5 block text-sm font-medium" for="api-token">
					{$_('androidAuth.tokenLabel')}
				</label>
				<input
					id="api-token"
					class="input font-mono text-sm"
					type="password"
					bind:value={apiToken}
					autocomplete="off"
					spellcheck="false"
					required
				/>
			</div>
			<button type="submit" class="btn-primary w-full" disabled={loading}>
				{loading ? $_('common.loading') : $_('androidAuth.submit')}
			</button>
		</form>
		<p class="mt-4 text-center text-sm" style:color="var(--text-muted)">
			<a href={resolve('/login')} class="hover:underline" style:color="var(--primary)">
				{$_('androidAuth.backToMethods')}
			</a>
		</p>
	</div>
</div>
