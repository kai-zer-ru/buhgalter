<script lang="ts">
	import SectionNavHub from '$lib/components/SectionNavHub.svelte';
	import { androidSettingsNavItems } from '$lib/android/nav-items';
</script>

<SectionNavHub items={androidSettingsNavItems()} />
