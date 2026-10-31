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

<form class="card max-w-lg space-y-6" onsubmit={savePassword}>
	<section class="space-y-4">
		<div class="space-y-1">
			<h2 class="font-semibold">{$_('settings.password.section')}</h2>
			<p class="text-sm" style:color="var(--text-muted)">{$_('settings.password.section_hint')}</p>
		</div>
		<div>
			<label class="field-label" for="old">{$_('settings.password.current')}</label>
			<input
				id="old"
				type="password"
				class="input"
				autocomplete="current-password"
				bind:value={oldPassword}
				required
			/>
		</div>
	</section>
	<section class="space-y-4 border-t pt-6" style:border-color="var(--border)">
		<div>
			<label class="field-label" for="new">{$_('settings.password.new')}</label>
			<input
				id="new"
				type="password"
				class="input"
				autocomplete="new-password"
				bind:value={newPassword}
				minlength="8"
				required
			/>
			<p class="mt-1 text-xs" style:color="var(--text-muted)">
				{$_('auth.password.requirements')}
			</p>
		</div>
		<div>
			<label class="field-label" for="confirm">{$_('settings.password.confirm')}</label>
			<input
				id="confirm"
				type="password"
				class="input"
				autocomplete="new-password"
				bind:value={confirmPassword}
				required
			/>
		</div>
	</section>
	<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
		<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
	</div>
</form>
