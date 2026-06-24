<script lang="ts">
	import type { DragGhostView } from '$lib/drag-reorder';

	let { ghost }: { ghost: DragGhostView } = $props();
</script>

<div
	class="pointer-events-none fixed z-[70] box-border overflow-hidden"
	style:left="{ghost.x - ghost.offsetX}px"
	style:top="{ghost.y - ghost.offsetY}px"
	style:width="{ghost.width}px"
	style:min-width="{ghost.width}px"
	style:max-width="{ghost.width}px"
	style:height="{ghost.height}px"
	aria-hidden="true"
>
	<div
		class="pointer-events-none h-full w-full opacity-95 [&>*]:box-border [&>*]:h-full [&>*]:w-full"
		style:box-shadow="0 20px 40px color-mix(in srgb, #000 35%, transparent)"
	>
		<!-- eslint-disable-next-line svelte/no-at-html-tags -- cloned DOM from same page only -->
		{@html ghost.html}
	</div>
</div>
