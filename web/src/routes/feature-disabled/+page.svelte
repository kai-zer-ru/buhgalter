<script lang="ts">
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
</script>

<div class="mx-auto max-w-lg space-y-3 py-10 text-center">
	<h1 class="text-xl font-semibold">{$_('features.disabled.title')}</h1>
	<p class="text-sm" style:color="var(--text-muted)">{$_('features.disabled.body')}</p>
	<a class="btn-primary inline-flex" href={resolve('/')}>{$_('features.disabled.home')}</a>
</div>
