<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		label,
		description = '',
		badge = '',
		badgeTone = 'muted',
		children
	}: {
		label: string;
		description?: string;
		badge?: string;
		badgeTone?: 'success' | 'muted' | 'warning';
		children: Snippet;
	} = $props();
</script>

<!-- account-group-panel / account-group-summary: shared chevron styles and the e2e helper. -->
<details class="account-group-panel settings-panel">
	<summary class="account-group-summary settings-panel-summary">
		<span class="min-w-0">
			<span class="flex flex-wrap items-center gap-2">
				<span class="font-medium">{label}</span>
				{#if badge}
					<span
						class="badge"
						class:badge-success={badgeTone === 'success'}
						class:settings-panel-badge-warning={badgeTone === 'warning'}>{badge}</span
					>
				{/if}
			</span>
			{#if description}
				<span class="mt-0.5 block text-sm leading-snug" style:color="var(--text-muted)">
					{description}
				</span>
			{/if}
		</span>
		<svg
			class="account-group-chevron"
			aria-hidden="true"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<path d="m6 9 6 6 6-6" />
		</svg>
	</summary>
	<div class="settings-panel-body">
		{@render children()}
	</div>
</details>

<style>
	.settings-panel {
		border: 1px solid var(--border);
		border-radius: var(--ui-radius-card, 1rem);
		background-color: var(--bg-elevated);
		box-shadow: var(--ui-elevation-1, 0 1px 2px rgb(0 0 0 / 0.05));
	}

	.settings-panel-summary {
		padding: 0.875rem 1rem;
		border-radius: inherit;
		transition: background-color 120ms;
	}

	.settings-panel-summary:hover {
		background-color: color-mix(in srgb, var(--border) 30%, transparent);
	}

	.settings-panel[open] > .settings-panel-summary {
		border-bottom-left-radius: 0;
		border-bottom-right-radius: 0;
	}

	.settings-panel-body {
		border-top: 1px solid var(--border);
		padding: 1rem;
	}

	@media (min-width: 640px) {
		.settings-panel-summary {
			padding: 1rem 1.25rem;
		}

		.settings-panel-body {
			padding: 1.25rem;
		}
	}

	.settings-panel-badge-warning {
		background-color: color-mix(in srgb, var(--warning) 14%, transparent);
		color: var(--warning);
		border-color: color-mix(in srgb, var(--warning) 30%, transparent);
	}
</style>
