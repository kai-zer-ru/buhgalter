<script lang="ts">
	import { _ } from 'svelte-i18n';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
</script>

<EmptyStateCard message={$_('settings.notifications.web_only_banner')}>
	<p class="text-sm" style:color="var(--text-muted)">
		{$_('settings.notifications.web_only_hint')}
	</p>
</EmptyStateCard>
