<script lang="ts">
	import { categoryIconUrl } from '$lib/finance';

	let { icon, size = 32 }: { icon: string; size?: number } = $props();

	const src = $derived(categoryIconUrl(icon));
	let failed = $state(false);

	$effect(() => {
		void icon;
		failed = false;
	});
</script>

<img
	src={failed ? categoryIconUrl('default') : src}
	alt=""
	class="inline-block shrink-0 object-cover"
	width={size}
	height={size}
	onerror={() => (failed = true)}
/>
