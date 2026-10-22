<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { hasServerUrl } from '$lib/platform/server-url';

	onMount(async () => {
		if (!hasServerUrl()) {
			await goto(resolve('/server-setup'));
		}
	});
</script>

<svelte:head>
	<title>{$_('androidAuth.chooseTitle')} — {$_('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="card w-full max-w-md">
		<h1 class="mb-2 text-2xl font-semibold">{$_('androidAuth.chooseTitle')}</h1>
		<p class="mb-6 text-sm" style:color="var(--text-muted)">{$_('androidAuth.chooseSubtitle')}</p>
		<div class="space-y-3">
			<a
				href={resolve('/login/password')}
				class="btn-primary flex w-full items-center justify-center"
			>
				{$_('androidAuth.method.password')}
			</a>
			<a href={resolve('/login/token')} class="btn-ghost flex w-full items-center justify-center">
				{$_('androidAuth.method.token')}
			</a>
		</div>
		<p class="mt-4 text-center text-sm" style:color="var(--text-muted)">
			<a href={resolve('/server-setup')} class="hover:underline" style:color="var(--primary)">
				{$_('serverSetup.changeServer')}
			</a>
		</p>
	</div>
</div>
