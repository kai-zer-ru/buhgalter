<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { getSetupStatus, postSetup, postSetupRestore } from '$lib/api/client';
	import { formatApiError } from '$lib/api/errors';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import AppIcon from '$lib/components/AppIcon.svelte';

	let adminLogin = $state('admin');
	let adminDisplayName = $state('');
	let adminPassword = $state('');
	let adminPasswordConfirm = $state('');
	let registrationEnabled = $state(false);
	let externalURL = $state('');
	let error = $state('');
	let loading = $state(false);
	let showPassword = $state(false);
	let restoreFile = $state<File | null>(null);
	let restoreLoading = $state(false);
	let restoreError = $state('');
	let restoreSuccess = $state('');

	const passwordOk = $derived(validatePasswordPolicy(adminPassword, adminLogin));
	const passwordsMatch = $derived(
		adminPasswordConfirm.length === 0 || adminPassword === adminPasswordConfirm
	);
	const formValid = $derived(
		passwordOk && adminPassword === adminPasswordConfirm && adminDisplayName.trim().length > 0
	);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (!formValid) {
			error = $_('errors.PASSWORDS_MISMATCH');
			return;
		}
		loading = true;
		try {
			await postSetup({
				admin_login: adminLogin.trim(),
				admin_display_name: adminDisplayName.trim(),
				admin_password: adminPassword,
				admin_password_confirm: adminPasswordConfirm,
				registration_enabled: registrationEnabled,
				external_url: externalURL.trim()
			});
			window.location.href = '/login';
		} catch (err) {
			error = formatApiError(err, 'setup.error');
		} finally {
			loading = false;
		}
	}

	function onRestoreFileChange(e: Event) {
		const target = e.target as HTMLInputElement;
		restoreFile = target.files?.[0] ?? null;
		restoreError = '';
		restoreSuccess = '';
	}

	async function submitRestore() {
		restoreError = '';
		restoreSuccess = '';
		if (!restoreFile) {
			restoreError = $_('setup.restore.file_required');
			return;
		}
		restoreLoading = true;
		try {
			const resp = await postSetupRestore(restoreFile);
			if (resp.configured) {
				window.location.href = '/login';
				return;
			}
			await getSetupStatus();
			restoreSuccess = $_('setup.restore.success_continue');
		} catch (err) {
			restoreError = formatApiError(err, 'setup.restore.error');
		} finally {
			restoreLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{$_('setup.title')} — {$_('app.title')}</title>
</svelte:head>

<div
	class="relative min-h-screen overflow-hidden bg-gradient-to-br from-slate-50 via-emerald-50/50 to-teal-50/30 px-4 py-10 sm:px-6"
>
	<div
		class="pointer-events-none absolute -left-32 -top-32 h-96 w-96 rounded-full bg-emerald-200/30 blur-3xl"
	></div>
	<div
		class="pointer-events-none absolute -bottom-24 -right-24 h-80 w-80 rounded-full bg-teal-200/40 blur-3xl"
	></div>

	<div class="relative mx-auto w-full max-w-lg">
		<div class="mb-8 text-center">
			<div class="mx-auto mb-4 flex justify-center">
				<AppIcon size={56} class="shadow-lg shadow-emerald-500/25" />
			</div>
			<h1 class="text-2xl font-bold tracking-tight text-slate-900 sm:text-3xl">
				{$_('app.title')}
			</h1>
			<p class="mt-2 text-sm text-slate-500">{$_('setup.subtitle')}</p>
		</div>

		<div
			class="rounded-2xl border border-white/60 bg-white/80 p-6 shadow-xl shadow-slate-200/50 backdrop-blur-sm sm:p-8"
		>
			<div class="mb-6">
				<h2 class="text-lg font-semibold text-slate-900">{$_('setup.heading')}</h2>
				<p class="mt-1 text-sm leading-relaxed text-slate-500">{$_('setup.heading_hint')}</p>
			</div>

			<form class="space-y-6" onsubmit={submit}>
				<div class="space-y-3">
					<p class="text-xs font-semibold uppercase tracking-wider text-slate-400">
						{$_('setup.section.database')}
					</p>
					<div
						class="flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/80 px-4 py-3"
					>
						<span
							class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-200/80 text-lg"
							>🗄️</span
						>
						<div>
							<p class="text-sm font-medium text-slate-800">{$_('setup.database.sqlite')}</p>
							<p class="text-xs text-slate-500">{$_('setup.database.hint')}</p>
						</div>
					</div>
					<div class="rounded-xl border border-slate-200 bg-white px-4 py-4">
						<p class="text-sm font-medium text-slate-800">{$_('setup.restore.title')}</p>
						<p class="mt-1 text-xs leading-relaxed text-slate-500">
							{$_('setup.restore.hint')}
						</p>
						<div class="mt-3 space-y-3">
							<input
								type="file"
								accept=".db,application/x-sqlite3,application/octet-stream"
								class="block w-full text-sm text-slate-700 file:mr-3 file:rounded-lg file:border-0 file:bg-slate-100 file:px-3 file:py-2 file:text-xs file:font-semibold file:text-slate-700 hover:file:bg-slate-200"
								onchange={onRestoreFileChange}
							/>
							<button
								type="button"
								class="inline-flex items-center justify-center rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
								disabled={restoreLoading || !restoreFile}
								onclick={submitRestore}
							>
								{#if restoreLoading}
									{$_('setup.restore.loading')}
								{:else}
									{$_('setup.restore.submit')}
								{/if}
							</button>
							{#if restoreError}
								<p class="text-xs text-red-600">{restoreError}</p>
							{/if}
							{#if restoreSuccess}
								<p class="text-xs text-emerald-700">{restoreSuccess}</p>
							{/if}
						</div>
					</div>
				</div>

				<hr class="border-slate-100" />

				<div class="space-y-4">
					<p class="text-xs font-semibold uppercase tracking-wider text-slate-400">
						{$_('setup.section.account')}
					</p>

					<div>
						<label class="mb-1.5 block text-sm font-medium text-slate-700" for="display-name">
							{$_('register.display_name')}
						</label>
						<input
							id="display-name"
							class="w-full rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
							bind:value={adminDisplayName}
							placeholder={$_('setup.display_name.placeholder')}
							maxlength="64"
							autocomplete="name"
							required
						/>
					</div>

					<div>
						<label class="mb-1.5 block text-sm font-medium text-slate-700" for="login">
							{$_('setup.login.label')}
						</label>
						<input
							id="login"
							class="w-full rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
							bind:value={adminLogin}
							placeholder="admin"
							minlength="3"
							maxlength="32"
							autocomplete="username"
							required
						/>
					</div>

					<div>
						<label class="mb-1.5 block text-sm font-medium text-slate-700" for="password">
							{$_('login.password')}
						</label>
						<div class="relative">
							<input
								id="password"
								type={showPassword ? 'text' : 'password'}
								class="w-full rounded-xl border border-slate-200 bg-white px-4 py-2.5 pr-11 text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
								bind:value={adminPassword}
								placeholder={$_('setup.password.placeholder')}
								minlength="8"
								autocomplete="new-password"
								required
							/>
							<button
								type="button"
								class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
								onclick={() => (showPassword = !showPassword)}
								aria-label={showPassword ? $_('setup.password.hide') : $_('setup.password.show')}
							>
								{showPassword ? '🙈' : '👁️'}
							</button>
						</div>
						{#if adminPassword.length > 0}
							<p class="mt-1.5 text-xs {passwordOk ? 'text-emerald-600' : 'text-amber-600'}">
								{passwordOk ? $_('setup.password.ok') : $_('auth.password.requirements')}
							</p>
						{/if}
						<p class="mt-1.5 text-xs text-slate-500">{$_('auth.password.requirements')}</p>
					</div>

					<div>
						<label class="mb-1.5 block text-sm font-medium text-slate-700" for="password-confirm">
							{$_('settings.password.confirm')}
						</label>
						<input
							id="password-confirm"
							type={showPassword ? 'text' : 'password'}
							class="w-full rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 {adminPasswordConfirm.length >
								0 && !passwordsMatch
								? 'border-red-300 focus:border-red-500 focus:ring-red-500/20'
								: ''}"
							bind:value={adminPasswordConfirm}
							placeholder={$_('setup.password_confirm.placeholder')}
							minlength="8"
							autocomplete="new-password"
							required
						/>
						{#if adminPasswordConfirm.length > 0}
							<p class="mt-1.5 text-xs {passwordsMatch ? 'text-emerald-600' : 'text-red-600'}">
								{passwordsMatch ? $_('setup.password_confirm.ok') : $_('errors.PASSWORDS_MISMATCH')}
							</p>
						{/if}
					</div>
				</div>

				<hr class="border-slate-100" />

				<div class="space-y-4">
					<p class="text-xs font-semibold uppercase tracking-wider text-slate-400">
						{$_('setup.section.options')}
					</p>

					<div
						class="flex items-center justify-between gap-4 rounded-xl border border-slate-200 bg-slate-50/50 px-4 py-3"
					>
						<div>
							<p class="text-sm font-medium text-slate-800">{$_('setup.registration.title')}</p>
							<p class="text-xs text-slate-500">{$_('setup.registration.hint')}</p>
						</div>
						<button
							type="button"
							role="switch"
							aria-checked={registrationEnabled}
							aria-label={$_('setup.registration.aria')}
							class="relative h-6 w-11 shrink-0 rounded-full transition-colors {registrationEnabled
								? 'bg-emerald-500'
								: 'bg-slate-300'}"
							onclick={() => (registrationEnabled = !registrationEnabled)}
						>
							<span
								class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform {registrationEnabled
									? 'translate-x-5'
									: ''}"
							></span>
						</button>
					</div>

					<div>
						<label class="mb-1.5 block text-sm font-medium text-slate-700" for="external">
							{$_('setup.external_url.label')}
							<span class="font-normal text-slate-400">{$_('setup.external_url.optional')}</span>
						</label>
						<input
							id="external"
							type="url"
							placeholder="https://buhgalter.example.com"
							class="w-full rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm transition placeholder:text-slate-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
							bind:value={externalURL}
						/>
						<p class="mt-1.5 text-xs text-slate-400">{$_('setup.external_url.hint')}</p>
					</div>
				</div>

				{#if error}
					<div
						class="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
						role="alert"
					>
						<span class="mt-0.5">⚠️</span>
						<span>{error}</span>
					</div>
				{/if}

				<button
					type="submit"
					class="flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 px-4 py-3 text-sm font-semibold text-white shadow-lg shadow-emerald-500/25 transition hover:from-emerald-500 hover:to-teal-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 disabled:cursor-not-allowed disabled:opacity-60"
					disabled={loading || !formValid}
				>
					{#if loading}
						<span
							class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"
						></span>
						{$_('setup.submit.loading')}
					{:else}
						{$_('setup.submit')}
					{/if}
				</button>
			</form>
		</div>

		<p class="mt-6 text-center text-xs text-slate-400">{$_('setup.footer')}</p>
	</div>
</div>
