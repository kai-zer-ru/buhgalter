<script lang="ts">
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		ackPasswordResetRequest,
		listPasswordResetRequests,
		type PasswordResetRequest
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';

	let requests = $state<PasswordResetRequest[]>([]);

	onMount(() => {
		if (get(user)?.is_admin) {
			void load();
		}
	});

	async function load() {
		if (!$user?.is_admin) {
			requests = [];
			return;
		}
		try {
			requests = await listPasswordResetRequests();
		} catch {
			requests = [];
		}
	}

	function displayName(req: PasswordResetRequest): string {
		return req.display_name?.trim() || req.login;
	}

	async function acknowledge(req: PasswordResetRequest) {
		await ackPasswordResetRequest(req.id);
		requests = requests.filter((item) => item.id !== req.id);
		await goto(resolve(`/admin/users?reset=${req.user_id}`));
	}
</script>

{#if $user?.is_admin && requests.length > 0}
	<div class="mb-4 space-y-2">
		{#each requests as req (req.id)}
			<div
				class="flex flex-wrap items-center justify-between gap-3 rounded-xl border px-4 py-3 text-sm"
				style:border-color="var(--border)"
				style:background-color="color-mix(in srgb, var(--primary) 8%, var(--bg-elevated))"
			>
				<p>
					{$_('admin.passwordReset.notice', { values: { name: displayName(req) } })}
				</p>
				<button type="button" class="btn-primary shrink-0" onclick={() => void acknowledge(req)}>
					{$_('common.ok')}
				</button>
			</div>
		{/each}
	</div>
{/if}
