<script lang="ts">
	import { _ } from 'svelte-i18n';
	import {
		failedOutboxCount,
		hasFailedOutbox,
		outboxTick,
		pendingOutboxCount
	} from '$lib/offline/store';
	import { serverReachability } from '$lib/offline/server-connectivity';
	import { pullFromServer, pullStatus, syncState } from '$lib/offline/sync';

	type Props = {
		ondone?: () => void;
	};

	let { ondone }: Props = $props();

	const syncing = $derived($syncState === 'syncing');
	const offline = $derived($serverReachability === 'offline');

	const statusText = $derived.by(() => {
		void $outboxTick;
		const pending = pendingOutboxCount();
		const failed = failedOutboxCount();
		const s = $pullStatus;

		if (s.kind === 'syncing') return $_('offline.syncing');
		if (offline) {
			if (failed > 0) {
				return $_('offline.drawer.offlineFailed', { values: { count: failed } });
			}
			if (pending > 0) {
				return $_('offline.drawer.offlinePending', { values: { count: pending } });
			}
			return $_('offline.status.offline');
		}
		if (s.kind === 'error') return $_('offline.status.pullFailed');
		if (s.kind === 'failed' || failed > 0) return $_('offline.syncFailed');
		if (s.kind === 'ok') return $_('offline.status.ok');
		if (pending > 0) {
			return $_('offline.status.pendingCount', { values: { count: pending } });
		}
		return $_('offline.status.idle');
	});

	async function sync() {
		await pullFromServer();
		ondone?.();
	}
</script>

<div class="android-drawer-sync space-y-2 px-1 pb-2">
	<p class="text-xs leading-snug" style:color="var(--text-muted)">{statusText}</p>
	<button
		type="button"
		class="btn-primary w-full text-sm"
		disabled={syncing}
		onclick={() => void sync()}
	>
		{syncing ? $_('offline.syncing') : $_('offline.syncNow')}
	</button>
	{#if hasFailedOutbox() && !offline}
		<p class="text-xs" style:color="var(--danger)">{$_('offline.syncFailed')}</p>
	{/if}
</div>
