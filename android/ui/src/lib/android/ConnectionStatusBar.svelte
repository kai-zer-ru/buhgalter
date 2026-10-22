<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { failedOutboxCount, outboxTick, pendingOutboxCount } from '$lib/offline/store';
	import { serverReachability } from '$lib/offline/server-connectivity';

	const offline = $derived($serverReachability === 'offline');

	const message = $derived.by(() => {
		if (!offline) return null;
		void $outboxTick;
		const pending = pendingOutboxCount();
		const failed = failedOutboxCount();
		if (failed > 0) {
			return $_('offline.bar.offlineFailed', { values: { count: failed } });
		}
		if (pending > 0) {
			return $_('offline.bar.offlinePending', { values: { count: pending } });
		}
		return $_('offline.noConnection');
	});
</script>

{#if message}
	<div class="android-connection-bar" role="status" aria-live="polite">
		{message}
	</div>
{/if}

<style>
	.android-connection-bar {
		flex-shrink: 0;
		padding: 0.5rem 1rem;
		padding-bottom: max(0.5rem, env(safe-area-inset-bottom));
		text-align: center;
		font-size: 0.875rem;
		background: color-mix(in srgb, var(--danger) 18%, var(--bg-elevated));
		color: var(--danger);
		border-top: 1px solid color-mix(in srgb, var(--danger) 35%, var(--border));
	}
</style>
