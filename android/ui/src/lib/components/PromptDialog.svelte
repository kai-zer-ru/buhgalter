<script lang="ts">
	import { onDestroy } from 'svelte';
	import { locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import { promptStore, resolvePrompt, type PromptState } from '$lib/prompt';
	import { pushModalEscape } from '$lib/modal-escape';

	let dialog = $state<PromptState>({ open: false, options: {} });
	let value = $state('');

	const unsubscribe = promptStore.subscribe((next) => {
		dialog = next;
		if (next.open) {
			value = next.options.defaultValue ?? '';
		}
	});

	onDestroy(unsubscribe);

	$effect(() => {
		if (!dialog.open) return;
		return pushModalEscape(() => resolvePrompt(null));
	});

	const title = $derived.by(() => {
		void $locale;
		return dialog.options.title ?? tr('common.confirm.title');
	});
	const confirmLabel = $derived.by(() => {
		void $locale;
		return dialog.options.confirmLabel ?? tr('common.save');
	});
	const cancelLabel = $derived.by(() => {
		void $locale;
		return dialog.options.cancelLabel ?? tr('common.cancel');
	});

	function onConfirm() {
		const trimmed = value.trim();
		if (!trimmed) return;
		resolvePrompt(trimmed);
	}
</script>

{#if dialog.open}
	<div
		class="modal-backdrop fixed inset-0 z-[60] flex items-end justify-center sm:items-center sm:p-4"
		style:background-color="color-mix(in srgb, #000 55%, transparent)"
		role="presentation"
		onclick={() => resolvePrompt(null)}
	>
		<div
			class="modal-panel card w-full max-w-md shadow-xl sm:rounded-2xl"
			role="dialog"
			aria-modal="true"
			aria-labelledby="prompt-dialog-title"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<form
				class="p-4 sm:p-6"
				onsubmit={(e) => {
					e.preventDefault();
					onConfirm();
				}}
			>
				<h2 id="prompt-dialog-title" class="text-lg font-semibold">{title}</h2>
				{#if dialog.options.message}
					<p class="mt-2 text-sm" style:color="var(--text-muted)">{dialog.options.message}</p>
				{/if}
				<input class="input mt-4 w-full" bind:value maxlength={dialog.options.maxLength ?? 80} />
				<div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
					<button
						type="button"
						class="btn-ghost w-full sm:w-auto"
						onclick={() => resolvePrompt(null)}
					>
						{cancelLabel}
					</button>
					<button type="submit" class="btn-primary w-full sm:w-auto" disabled={!value.trim()}>
						{confirmLabel}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
