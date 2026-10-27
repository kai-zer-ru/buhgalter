<script lang="ts">
	import SectionNavHub from '$lib/components/SectionNavHub.svelte';
	import { androidSettingsNavItems } from '$lib/android/nav-items';
	import { featureFlags } from '$lib/features';
	import { householdMe, householdNavLabelKey } from '$lib/household/membership';

	const items = $derived.by(() => {
		void $featureFlags;
		return androidSettingsNavItems({ householdLabelKey: householdNavLabelKey($householdMe) });
	});
</script>

<SectionNavHub {items} />
