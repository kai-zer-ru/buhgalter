<script lang="ts">
	import SectionNavHub from '$lib/components/SectionNavHub.svelte';
	import { androidSettingsNavItems } from '$lib/android/nav-items';
	import { featureFlags } from '$lib/features';

	const items = $derived.by(() => {
		void $featureFlags;
		return androidSettingsNavItems();
	});
</script>

<SectionNavHub {items} />
