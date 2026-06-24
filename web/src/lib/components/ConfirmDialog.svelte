<script lang="ts">
	import { onDestroy } from 'svelte';
	import { locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import { confirmStore, resolveConfirm, type ConfirmState } from '$lib/confirm';
	import { pushModalEscape } from '$lib/modal-escape';

	let state = $state<ConfirmState>({ open: false, options: { message: '' } });

	const unsubscribe = confirmStore.subscribe((value) => {
		state = value;
	});

	onDestroy(unsubscribe);

	$effect(() => {
		if (!state.open) return;
		return pushModalEscape(() => resolveConfirm(false));
	});

	function onBackdropClick() {
		resolveConfirm(false);
	}

	function onConfirm() {
		resolveConfirm(true);
	}

	const title = $derived.by(() => {
		void $locale;
		return state.options.title ?? tr('common.confirm.title');
	});
	const confirmLabel = $derived.by(() => {
		void $locale;
		return state.options.confirmLabel ?? tr('common.confirm.confirm');
	});
	const cancelLabel = $derived.by(() => {
		void $locale;
		return state.options.cancelLabel ?? tr('common.cancel');
	});
</script>

{#if state.open}
	<div
		class="fixed inset-0 z-[60] flex items-center justify-center p-4"
		style:background-color="color-mix(in srgb, #000 55%, transparent)"
		role="presentation"
		onclick={onBackdropClick}
	>
		<div
			class="card w-full max-w-md shadow-xl"
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="confirm-dialog-title"
			aria-describedby="confirm-dialog-message"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<h2 id="confirm-dialog-title" class="text-lg font-semibold">{title}</h2>
			<p id="confirm-dialog-message" class="mt-2 text-sm" style:color="var(--text-muted)">
				{state.options.message}
			</p>
			<div class="mt-6 flex flex-wrap justify-end gap-2">
				<button type="button" class="btn-ghost" onclick={() => resolveConfirm(false)}>
					{cancelLabel}
				</button>
				<button
					type="button"
					class={state.options.danger ? 'btn-danger' : 'btn-primary'}
					onclick={onConfirm}
				>
					{confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
