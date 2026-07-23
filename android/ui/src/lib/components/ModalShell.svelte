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
		class="modal-backdrop fixed inset-0 z-50 flex items-end justify-center bg-black/50 sm:items-center sm:p-4"
		role="presentation"
		onclick={onclose}
	>
		<div
			class="modal-panel card flex max-h-[min(92dvh,44rem)] w-full flex-col overflow-hidden sm:max-h-[min(92vh,44rem)] {maxWidth}"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => event.stopPropagation()}
		>
			<div class="shrink-0 border-b px-4 py-3 sm:px-6 sm:py-4" style:border-color="var(--border)">
				<h2 class="text-lg font-semibold">{title}</h2>
			</div>
			<div class="min-h-0 flex-1 overflow-y-auto px-4 py-4 sm:px-6">
				{@render children()}
			</div>
			{#if footer}
				<div
					class="modal-footer flex shrink-0 flex-col-reverse gap-2 border-t px-4 py-3 sm:flex-row sm:justify-end sm:px-6 sm:py-4"
					style:border-color="var(--border)"
				>
					{@render footer()}
				</div>
			{/if}
		</div>
	</div>
{/if}
