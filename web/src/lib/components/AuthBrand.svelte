<script lang="ts">
	import { _ } from 'svelte-i18n';
	import AppIcon from '$lib/components/AppIcon.svelte';
</script>

<div class="mb-6 flex items-center gap-3">
	<AppIcon size={44} class="rounded-xl" />
	<div class="min-w-0">
		<p class="font-semibold">{$_('app.title')}</p>
		<p class="text-sm" style:color="var(--text-muted)">{$_('app.tagline')}</p>
	</div>
</div>
