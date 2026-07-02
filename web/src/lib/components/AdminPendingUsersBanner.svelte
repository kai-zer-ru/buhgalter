<script lang="ts">
	import { afterNavigate, goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import { listAdminUsers, type AdminUser } from '$lib/api/client';
	import { pendingUsersBannerTick } from '$lib/stores/admin-pending-users';
	import { user } from '$lib/stores/auth';

	let pendingUsers = $state<AdminUser[]>([]);

	afterNavigate(() => {
		if (get(user)?.is_admin) {
			void load();
		}
	});

	$effect(() => {
		if (!$user?.is_admin) {
			pendingUsers = [];
			return;
		}
		void $pendingUsersBannerTick;
		void $page.url.pathname;
		void load();
	});

	async function load() {
		if (!$user?.is_admin) {
			pendingUsers = [];
			return;
		}
		try {
			const users = await listAdminUsers();
			pendingUsers = users.filter((u) => u.status === 'pending');
		} catch {
			pendingUsers = [];
		}
	}

	function displayName(u: AdminUser): string {
		return u.display_name?.trim() || u.login;
	}

	async function openModeration(u: AdminUser) {
		await goto(resolve(`/admin/users?moderate=${u.id}`));
	}
</script>

{#if $user?.is_admin && pendingUsers.length > 0}
	<div class="mb-4 space-y-2">
		{#each pendingUsers as u (u.id)}
			<div
				class="flex flex-wrap items-center justify-between gap-3 rounded-xl border px-4 py-3 text-sm"
				style:border-color="var(--border)"
				style:background-color="color-mix(in srgb, var(--primary) 8%, var(--bg-elevated))"
			>
				<p>
					{$_('admin.userModeration.notice', { values: { name: displayName(u) } })}
				</p>
				<button type="button" class="btn-primary shrink-0" onclick={() => void openModeration(u)}>
					{$_('admin.userModeration.action')}
				</button>
			</div>
		{/each}
	</div>
{/if}
