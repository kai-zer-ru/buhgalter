<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import {
		failedOutboxCount,
		hasFailedOutbox,
		hasPendingOutbox,
		outboxTick,
		pendingOutboxCount
	} from '$lib/offline/store';
	import { serverReachability } from '$lib/offline/server-connectivity';
	import { syncState, scheduleSyncOutbox } from '$lib/offline/sync';

	let pending = $state(false);
	let failed = $state(false);
	let syncing = $state(false);

	const online = $derived($serverReachability === 'online');

	function refresh() {
		pending = hasPendingOutbox();
		failed = hasFailedOutbox();
	}

	onMount(() => {
		const unsubTick = outboxTick.subscribe(() => refresh());
		const unsubSync = syncState.subscribe((s) => {
			syncing = s === 'syncing';
		});
		refresh();
		return () => {
			unsubTick();
			unsubSync();
		};
	});
</script>

{#if online && (pending || failed)}
	<div
		class="border-b px-4 py-2 text-center text-sm sm:px-6"
		style:border-color="var(--border)"
		style:background-color="color-mix(in srgb, var(--primary) 8%, var(--bg-elevated))"
	>
		{#if syncing}
			<span style:color="var(--text-muted)">{$_('offline.syncing')}</span>
		{:else if failed}
			<span style:color="var(--danger)">
				{$_('offline.syncFailed')}
				{#if failedOutboxCount() > 1}
					({failedOutboxCount()})
				{/if}
			</span>
			<button type="button" class="ml-2 underline" onclick={() => scheduleSyncOutbox()}>
				{$_('common.retry')}
			</button>
		{:else}
			<span style:color="var(--text-muted)">
				{$_('offline.status.pendingCount', { values: { count: pendingOutboxCount() } })}
			</span>
			<button type="button" class="ml-2 underline" onclick={() => scheduleSyncOutbox()}>
				{$_('offline.syncNow')}
			</button>
		{/if}
	</div>
{/if}
