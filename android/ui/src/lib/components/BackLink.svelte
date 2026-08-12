<script lang="ts">
	import { resolve } from '$app/paths';

	export type BackLinkHref =
		| '/'
		| '/accounts'
		| '/accounts/new'
		| '/settings'
		| '/settings/profile'
		| '/settings/password'
		| '/settings/security'
		| '/settings/server'
		| '/settings/tokens'
		| '/settings/notifications'
		| '/settings/bank-notifications'
		| '/settings/bank-notifications/drafts'
		| '/settings/bank-notifications/history'
		| '/settings/import'
		| '/settings/transaction-templates'
		| '/admin'
		| '/admin/system'
		| '/admin/backups'
		| '/admin/diagnostics'
		| '/debts'
		| '/debtors'
		| '/credits'
		| '/subscriptions'
		| '/categories'
		| '/merchants'
		| '/tags'
		| '/recurring-operations'
		| '/transactions'
		| '/stats'
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
		// BackLinkHref is a curated set of static routes; casting to full Pathname
		// pulls in param routes and breaks resolve() overload selection.
		return resolve(item.href as '/');
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
					{:else}
						<span class="breadcrumb-current" aria-current="page">{item.label}</span>
					{/if}
					{#if index < items.length - 1}
						<span class="breadcrumb-sep" aria-hidden="true">/</span>
					{/if}
				</li>
			{/each}
		</ol>
	</nav>
{/if}
