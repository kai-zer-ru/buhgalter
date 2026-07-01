<script lang="ts">
	import { resolve } from '$app/paths';

	export type BackLinkHref =
		| '/'
		| '/accounts'
		| '/settings'
		| '/admin'
		| '/admin/users'
		| '/admin/backups'
		| '/admin/diagnostics'
		| '/debts'
		| '/credits'
		| '/transactions'
		| '/stats'
		| '/debtors'
		| '/accounts/new'
		| '/recurring-operations'
		| '/budget';

	export type BreadcrumbItem = {
		href: BackLinkHref;
		label: string;
		search?: string;
	};

	let {
		items = []
	}: {
		items: BreadcrumbItem[];
	} = $props();

	function target(item: BreadcrumbItem): string {
		return item.search ? `${resolve(item.href)}?${item.search}` : resolve(item.href);
	}
</script>

{#if items.length > 0}
	<nav class="breadcrumbs" aria-label="Breadcrumbs">
		<ol class="flex flex-wrap items-center gap-1.5">
			{#each items as item, index (`${index}:${item.href}:${item.label}:${item.search ?? ''}`)}
				<li class="inline-flex items-center gap-1.5">
					{#if index < items.length - 1}
						<!-- eslint-disable svelte/no-navigation-without-resolve -- path built with resolve() helper -->
						<a href={target(item)} class="breadcrumb-link">{item.label}</a>
						<!-- eslint-enable svelte/no-navigation-without-resolve -->
						<span class="breadcrumb-separator" aria-hidden="true">/</span>
					{:else}
						<span class="breadcrumb-current" aria-current="page">{item.label}</span>
					{/if}
				</li>
			{/each}
		</ol>
	</nav>
{/if}
