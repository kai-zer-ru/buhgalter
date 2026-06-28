<script lang="ts">
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import type { PendingVersionUpdate } from '$lib/version-check';
	import { dismissVersionUpdate } from '$lib/version-check';

	let {
		update,
		onclose
	}: {
		update: PendingVersionUpdate;
		onclose: () => void;
	} = $props();

	const backupsUrl = `${resolve('/settings')}?tab=admin&admin_tab=backups`;

	function handleDismiss() {
		if (update.latest_version) {
			dismissVersionUpdate(update.latest_version);
		}
		onclose();
	}
</script>

<ModalShell open={true} title={$_('update.title')} {onclose}>
	<div class="space-y-3 text-sm">
		<p>
			{$_('update.message', { values: { version: update.latest_version ?? '' } })}
		</p>
		<ul class="list-disc space-y-1 pl-5" style:color="var(--text-muted)">
			<li>
				{$_('update.backup')}
				{$_('update.backup_or')}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path -->
				<a href={backupsUrl} class="hover:underline" style:color="var(--primary)">
					{$_('update.backup_link')}
				</a>.
			</li>
			<li>{$_('update.release_notes')}</li>
		</ul>
	</div>
	{#snippet footer()}
		{#if update.release_url}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- external release URL -->
			<a
				href={update.release_url}
				target="_blank"
				rel="noopener noreferrer"
				class="btn-primary text-center"
			>
				{$_('update.open_release')}
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/if}
		<button type="button" class="btn-ghost" onclick={handleDismiss}>
			{$_('update.dismiss')}
		</button>
	{/snippet}
</ModalShell>
