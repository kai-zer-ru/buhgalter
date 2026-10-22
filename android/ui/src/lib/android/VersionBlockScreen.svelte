<script lang="ts">
	import { _ } from 'svelte-i18n';
	import AppIcon from '$lib/components/AppIcon.svelte';
	import type { AppVersionInfo } from '$lib/version-check';

	type Props = {
		info: AppVersionInfo;
	};

	let { info }: Props = $props();

	const displayApp = $derived(`v${info.appVersion}`);
	const displayServer = $derived(
		info.serverVersion ? `v${info.serverVersion}` : $_('update.app.serverUnknown')
	);
</script>

<div class="flex min-h-screen flex-col items-center justify-center gap-6 px-6 py-10 text-center">
	<AppIcon size={64} class="h-16 w-16" />
	<div class="max-w-md space-y-3">
		<h1 class="text-xl font-semibold">{$_('update.app.blockTitle')}</h1>
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('update.app.blockMessage', {
				values: { appVersion: displayApp, serverVersion: displayServer }
			})}
		</p>
		<dl
			class="space-y-2 rounded-lg border px-4 py-3 text-left text-sm"
			style:border-color="var(--border)"
		>
			<div class="flex items-baseline justify-between gap-4">
				<dt style:color="var(--text-muted)">{$_('update.app.serverVersion')}</dt>
				<dd class="font-medium tabular-nums">{displayServer}</dd>
			</div>
			<div class="flex items-baseline justify-between gap-4">
				<dt style:color="var(--text-muted)">{$_('update.app.clientVersion')}</dt>
				<dd class="font-medium tabular-nums" style:color="var(--warning)">{displayApp}</dd>
			</div>
		</dl>
		<p
			class="rounded-lg border px-3 py-2.5 text-sm"
			style:border-color="color-mix(in srgb, var(--warning) 45%, var(--border))"
			style:background-color="color-mix(in srgb, var(--warning) 10%, var(--bg-elevated))"
			style:color="var(--warning)"
			role="alert"
		>
			{$_('update.app.blockHint')}
		</p>
	</div>
	{#if info.releaseUrl}
		<!-- eslint-disable svelte/no-navigation-without-resolve -- external release URL -->
		<a
			href={info.releaseUrl}
			target="_blank"
			rel="noopener noreferrer"
			class="btn-primary w-full max-w-md text-center"
		>
			{$_('update.app.downloadApk')}
		</a>
		<!-- eslint-enable svelte/no-navigation-without-resolve -->
	{/if}
</div>
