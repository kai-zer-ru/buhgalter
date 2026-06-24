<script lang="ts">
	import type { Snippet } from 'svelte';
	import { pushModalEscape } from '$lib/modal-escape';

	let {
		open = $bindable(false),
		title,
		maxWidth = 'max-w-lg',
		onclose,
		children,
		footer
	}: {
		open?: boolean;
		title: string;
		maxWidth?: string;
		onclose: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	$effect(() => {
		if (!open) return;
		return pushModalEscape(onclose);
	});
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="presentation"
		onclick={onclose}
	>
		<div
			class="card flex max-h-[min(92vh,44rem)] w-full flex-col overflow-hidden {maxWidth}"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => event.stopPropagation()}
		>
			<div class="shrink-0 border-b px-6 py-4" style:border-color="var(--border)">
				<h2 class="text-lg font-semibold">{title}</h2>
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto px-6 py-4">
				{@render children()}
			</div>
			{#if footer}
				<div
					class="flex shrink-0 justify-end gap-2 border-t px-6 py-4"
					style:border-color="var(--border)"
				>
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
