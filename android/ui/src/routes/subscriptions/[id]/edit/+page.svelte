<script lang="ts">
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { listSubscriptions, type Subscription } from '$lib/api/client';
	import SubscriptionForm from '$lib/components/SubscriptionForm.svelte';
	import { leaveForm } from '$lib/android/form-nav';
	import { parseFormReturnPath } from '$lib/android/form-routes';
	import { dataRefreshTick } from '$lib/offline/sync';

	const id = $derived($page.params.id ?? '');
	const returnTo = $derived(
		parseFormReturnPath($page.url.searchParams.get('from'), '/subscriptions')
	);

	let subscription = $state<Subscription | null>(null);
	let ready = $state(false);

	$effect(() => {
		if (!id) return;
		ready = false;
		void load(id);
	});

	async function load(subId: string) {
		try {
			const subs = await listSubscriptions();
			subscription = subs.find((item) => item.id === subId) ?? null;
		} catch {
			subscription = null;
		} finally {
			ready = true;
		}
	}

	function finish() {
		dataRefreshTick.update((n) => n + 1);
		void leaveForm(returnTo);
	}
</script>

{#if ready && subscription}
	<SubscriptionForm backHref={returnTo} {subscription} onclose={finish} onsaved={finish} />
{:else if ready}
	<p class="p-4 text-sm" style:color="var(--text-muted)">{$_('common.notFound')}</p>
{/if}
