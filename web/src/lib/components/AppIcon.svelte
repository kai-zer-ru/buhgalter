<script lang="ts">
	let {
		size = 32,
		class: className = ''
	}: {
		size?: number;
		class?: string;
	} = $props();
</script>

<img
	src="/icon.png"
	alt=""
	width={size}
	height={size}
	class="shrink-0 rounded-lg object-cover {className}"
/>
