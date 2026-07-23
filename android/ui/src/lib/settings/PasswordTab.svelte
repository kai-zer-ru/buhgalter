<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { changePassword } from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');

	async function savePassword(e: Event) {
		e.preventDefault();
		if (newPassword !== confirmPassword) {
			toast.error($_('errors.PASSWORDS_MISMATCH'));
			return;
		}
		if (!validatePasswordPolicy(newPassword, $user?.login ?? '')) {
			toast.error($_('auth.password.requirements'));
			return;
		}
		if (oldPassword === newPassword) {
			toast.error($_('errors.PASSWORD_UNCHANGED'));
			return;
		}
		loading = true;
		try {
			await changePassword(oldPassword, newPassword, confirmPassword);
			oldPassword = '';
			newPassword = '';
			confirmPassword = '';
			toast($_('settings.password.changed'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}
</script>

<form class="card max-w-lg space-y-4" onsubmit={savePassword}>
	<div>
		<label class="mb-1.5 block text-sm font-medium" for="old"
			>{$_('settings.password.current')}</label
		>
		<input id="old" type="password" class="input" bind:value={oldPassword} required />
	</div>
	<div>
		<label class="mb-1.5 block text-sm font-medium" for="new">{$_('settings.password.new')}</label>
		<input id="new" type="password" class="input" bind:value={newPassword} minlength="8" required />
		<p class="mt-1 text-xs" style:color="var(--text-muted)">
			{$_('auth.password.requirements')}
		</p>
	</div>
	<div>
		<label class="mb-1.5 block text-sm font-medium" for="confirm"
			>{$_('settings.password.confirm')}</label
		>
		<input id="confirm" type="password" class="input" bind:value={confirmPassword} required />
	</div>
	<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
</form>
