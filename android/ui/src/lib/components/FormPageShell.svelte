<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { leaveForm } from '$lib/android/form-nav';
	import { shellHeader } from '$lib/android/shell-header';

	let {
		title,
		backHref,
		onback,
		children,
		footer
	}: {
		title: string;
		backHref?: string;
		onback?: () => void;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	function goBack() {
		if (onback) {
			onback();
			return;
		}
		if (backHref) {
			void leaveForm(backHref);
			return;
		}
		window.history.back();
	}

	$effect(() => {
		shellHeader.set({ title, onBack: goBack });
	});

	onMount(() => {
		return () => shellHeader.set(null);
	});
</script>

<div class="form-page">
	<div class="form-page-scroll">
		{@render children()}
	</div>

	{#if footer}
		<footer class="form-page-footer">
			{@render footer()}
		</footer>
	{/if}
</div>

<style>
	/* Fill shell main; footer stays pinned — scroll only the middle section. */
	.form-page {
		display: flex;
		flex: 1 1 0;
		flex-direction: column;
		min-height: 0;
		width: 100%;
		overflow: hidden;
	}

	.form-page-scroll {
		flex: 1 1 0;
		min-height: 0;
		overflow-y: auto;
		-webkit-overflow-scrolling: touch;
		padding: 1rem 1rem 0.5rem;
	}

	.form-page-footer {
		flex-shrink: 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
		border-top: 1px solid var(--border);
		background-color: var(--bg-elevated);
		padding: 0.75rem 1rem;
		padding-bottom: max(0.75rem, env(safe-area-inset-bottom));
	}

	.form-page-footer :global(.btn-primary),
	.form-page-footer :global(.btn-ghost) {
		width: 100%;
		min-height: 2.75rem;
	}
</style>
