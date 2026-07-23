<script lang="ts">
	import { _ } from 'svelte-i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import type { AppVersionInfo } from '$lib/version-check';

	type Props = {
		info: AppVersionInfo;
		onclose: () => void;
	};

	let { info, onclose }: Props = $props();

	const displayApp = $derived(`v${info.appVersion}`);
	const displayServer = $derived(
		info.serverVersion ? `v${info.serverVersion}` : $_('update.app.serverUnknown')
	);
</script>

<ModalShell open={true} title={$_('update.app.title')} {onclose}>
	<div class="space-y-4 text-sm">
		<dl class="space-y-3">
			<div class="flex items-baseline justify-between gap-4">
				<dt style:color="var(--text-muted)">{$_('update.app.serverVersion')}</dt>
				<dd class="font-medium tabular-nums">{displayServer}</dd>
			</div>
			<div class="flex items-baseline justify-between gap-4">
				<dt style:color="var(--text-muted)">{$_('update.app.clientVersion')}</dt>
				<dd class="font-medium tabular-nums" class:update-app-version-warn={info.versionMismatch}>
					{displayApp}
					{#if info.versionMismatch}
						<span class="update-app-version-warn-icon" aria-hidden="true">!</span>
					{/if}
				</dd>
			</div>
		</dl>

		{#if info.versionMismatch}
			<p
				class="rounded-lg border px-3 py-2.5"
				style:border-color="color-mix(in srgb, var(--warning) 45%, var(--border))"
				style:background-color="color-mix(in srgb, var(--warning) 10%, var(--bg-elevated))"
				style:color="var(--warning)"
				role="alert"
			>
				{$_('update.app.mismatchWarning')}
			</p>
		{/if}
	</div>
	{#snippet footer()}
		{#if info.releaseUrl}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- external release URL -->
			<a
				href={info.releaseUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="btn-primary text-center"
			>
				{$_('update.app.downloadApk')}
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		<button type="button" class="btn-ghost" onclick={onclose}>
			{$_('common.close')}
		</button>
	{/snippet}
</ModalShell>

<style>
	.update-app-version-warn {
		color: var(--warning);
	}

	.update-app-version-warn-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		margin-left: 0.25rem;
		min-width: 1.125rem;
		height: 1.125rem;
		border-radius: 9999px;
		background-color: color-mix(in srgb, var(--warning) 22%, transparent);
		font-size: 0.75rem;
		font-weight: 700;
		line-height: 1;
		vertical-align: middle;
	}
</style>
