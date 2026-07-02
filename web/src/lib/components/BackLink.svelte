<script lang="ts">
	import { resolve } from '$app/paths';

	export type BackLinkHref =
		| '/'
		| '/accounts'
		| '/settings'
		| '/settings/password'
		| '/settings/tokens'
		| '/settings/notifications'
		| '/settings/categories'
		| '/settings/import'
		| '/settings/recurring-operations'
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
		| '/budget';

	export type BreadcrumbItem = {
		href: BackLinkHref;
		label: string;
	};

	let {
		items = []
	}: {
		items: BreadcrumbItem[];
	} = $props();

	function target(item: BreadcrumbItem): string {
		return resolve(item.href);
	}
</script>

{#if items.length > 0}
	<nav class="breadcrumbs" aria-label="Breadcrumbs">
		<ol class="flex flex-wrap items-center gap-1.5">
			{#each items as item, index (`${index}:${item.href}:${item.label}`)}
				<li class="inline-flex items-center gap-1.5">
					{#if index < items.length - 1}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolve() in target() -->
						<a href={target(item)} class="breadcrumb-link">{item.label}</a>
						<span class="breadcrumb-separator" aria-hidden="true">/</span>
					{:else}
						<span class="breadcrumb-current" aria-current="page">{item.label}</span>
					{/if}
				</li>
			{/each}
		</ol>
	</nav>
{/if}
