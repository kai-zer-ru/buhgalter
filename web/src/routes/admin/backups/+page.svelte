<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		backupDownloadUrl,
		getBackupSettings,
		listBackups,
		putBackupSettings,
		restoreBackup,
		runBackup,
		type BackupFile
	} from '$lib/api/client';
	import { formatApiError } from '$lib/api/errors';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { logout, user } from '$lib/stores/auth';
	import { confirm } from '$lib/confirm';
	import { toast } from '$lib/toast';

	let files = $state<BackupFile[]>([]);
	let backupEnabled = $state(false);
	let backupTime = $state('03:00');
	let backupRetention = $state(7);
	let restoreFile = $state<File | null>(null);
	let restoreFileInput: HTMLInputElement | undefined = $state();
	let restoreConfirm = $state('');
	let scheduleFeedback = $state({ error: '', success: '' });
	let restoreFeedback = $state({ error: '', success: '' });
	let loading = $state(false);
	const tz = $derived($user?.timezone ?? 'Europe/Moscow');
	const restoreReady = $derived(restoreConfirm === 'RESTORE' && restoreFile !== null);

	onMount(async () => {
		if (!$user?.is_admin) {
			await goto(resolve('/'));
			return;
		}
		await refresh();
	});

	async function refresh() {
		const [list, settings] = await Promise.all([listBackups(), getBackupSettings()]);
		files = list;
		backupEnabled = settings.backup_enabled;
		backupTime = settings.backup_time;
		backupRetention = settings.backup_retention;
	}

	async function saveSettings(e: Event) {
		e.preventDefault();
		scheduleFeedback = { error: '', success: '' };
		loading = true;
		try {
			await putBackupSettings({
				backup_enabled: backupEnabled,
				backup_time: backupTime,
				backup_retention: backupRetention
			});
			scheduleFeedback = { error: '', success: $_('admin.backups.saved') };
			toast($_('admin.backups.saved'));
		} catch (err) {
			scheduleFeedback = { error: formatApiError(err), success: '' };
		} finally {
			loading = false;
		}
	}

	async function manualBackup() {
		scheduleFeedback = { error: '', success: '' };
		loading = true;
		try {
			await runBackup();
			await refresh();
			scheduleFeedback = { error: '', success: $_('admin.backups.created') };
			toast($_('admin.backups.created'));
		} catch (err) {
			scheduleFeedback = { error: formatApiError(err), success: '' };
		} finally {
			loading = false;
		}
	}

	async function handleRestore(e: Event) {
		e.preventDefault();
		restoreFeedback = { error: '', success: '' };
		if (!restoreFile) {
			restoreFeedback = { error: $_('admin.backups.pick_file'), success: '' };
			return;
		}
		if (restoreConfirm !== 'RESTORE') {
			restoreFeedback = { error: $_('admin.backups.restore_type'), success: '' };
			return;
		}
		const ok = await confirm({
			message: $_('admin.backups.confirm.restore'),
			confirmLabel: $_('common.confirm.confirm'),
			danger: true
		});
		if (!ok) return;

		loading = true;
		try {
			await restoreBackup(restoreFile, restoreConfirm);
			await logout();
			window.location.href = resolve('/login');
		} catch (err) {
			restoreFeedback = { error: formatApiError(err), success: '' };
		} finally {
			loading = false;
		}
	}

	function onFileChange(e: Event) {
		const input = e.target as HTMLInputElement;
		restoreFile = input.files?.[0] ?? null;
	}

	function pickRestoreFile() {
		restoreFileInput?.click();
	}

	function clearRestoreFile() {
		restoreFile = null;
		if (restoreFileInput) {
			restoreFileInput.value = '';
		}
	}

	function formatSize(bytes: number) {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

<div class="space-y-4">
	<form class="card space-y-4" onsubmit={saveSettings}>
		<h2 class="text-lg font-medium">{$_('admin.backups.schedule')}</h2>
		<div class="flex items-center justify-between gap-4">
			<span class="text-sm">{$_('admin.backups.auto')}</span>
			<button
				type="button"
				role="switch"
				aria-label={$_('admin.backups.auto')}
				aria-checked={backupEnabled}
				class="relative h-6 w-11 shrink-0 rounded-full transition-colors"
				style:background-color={backupEnabled ? 'var(--primary)' : 'var(--border)'}
				onclick={() => (backupEnabled = !backupEnabled)}
			>
				<span
					class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform"
					class:translate-x-5={backupEnabled}
				></span>
			</button>
		</div>
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<label class="mb-1.5 block text-sm" for="time">{$_('admin.backups.time')}</label>
				<input id="time" type="time" class="input" bind:value={backupTime} />
			</div>
			<div>
				<label class="mb-1.5 block text-sm" for="retention">{$_('admin.backups.retention')}</label>
				<input
					id="retention"
					type="number"
					class="input"
					min="1"
					max="365"
					bind:value={backupRetention}
				/>
			</div>
		</div>
		<div class="flex flex-wrap gap-2">
			<button type="submit" class="btn-primary" disabled={loading}>{$_('common.save')}</button>
			<button type="button" class="btn-primary" disabled={loading} onclick={manualBackup}>
				{$_('admin.backups.run_now')}
			</button>
		</div>
		<FormFeedback error={scheduleFeedback.error} success={scheduleFeedback.success} />
	</form>

	<div class="card">
		<h2 class="mb-4 text-lg font-medium">{$_('admin.backups.archive')}</h2>
		<div class="hidden md:block md:overflow-x-auto">
			<table class="w-full text-left text-sm">
				<thead>
					<tr style:color="var(--text-muted)">
						<th class="pb-3 pr-4">{$_('admin.backups.col.file')}</th>
						<th class="pb-3 pr-4">{$_('admin.backups.col.size')}</th>
						<th class="pb-3 pr-4">{$_('admin.backups.col.date')}</th>
						<th class="pb-3"></th>
					</tr>
				</thead>
				<tbody>
					{#each files as f (f.filename)}
						<tr class="border-t" style:border-color="var(--border)">
							<td class="py-3 pr-4 font-mono text-xs">{f.filename}</td>
							<td class="py-3 pr-4">{formatSize(f.size)}</td>
							<td class="py-3 pr-4">{formatAPIDateTimeForDisplay(f.created_at, tz)}</td>
							<td class="py-3 text-right">
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- API download endpoint -->
								<a class="btn-ghost inline-block" href={backupDownloadUrl(f.filename)}>
									{$_('admin.backups.download')}
								</a>
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="4" class="py-4" style:color="var(--text-muted)"
								>{$_('admin.backups.empty')}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<div class="space-y-3 md:hidden">
			{#each files as f (f.filename)}
				<article class="rounded-xl border p-4" style:border-color="var(--border)">
					<p class="break-all font-mono text-xs">{f.filename}</p>
					<dl class="mt-2 grid gap-2 text-sm">
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('admin.backups.col.size')}</dt>
							<dd>{formatSize(f.size)}</dd>
						</div>
						<div class="flex justify-between gap-2">
							<dt style:color="var(--text-muted)">{$_('admin.backups.col.date')}</dt>
							<dd>{formatAPIDateTimeForDisplay(f.created_at, tz)}</dd>
						</div>
					</dl>
					<!-- eslint-disable svelte/no-navigation-without-resolve -- API download endpoint, not app route -->
					<a
						class="btn-ghost mt-3 inline-block w-full text-center"
						href={backupDownloadUrl(f.filename)}
					>
						{$_('admin.backups.download')}
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				</article>
			{:else}
				<p class="py-4 text-sm" style:color="var(--text-muted)">{$_('admin.backups.empty')}</p>
			{/each}
		</div>
	</div>

	<form class="card space-y-4" onsubmit={handleRestore}>
		<h2 class="text-lg font-medium">{$_('admin.backups.restore')}</h2>
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('admin.backups.restore.hint')}
		</p>

		<div class="space-y-3">
			<label class="mb-1.5 block text-sm font-medium" for="restore-file">
				{$_('admin.backups.restore.file')}
			</label>
			<input
				id="restore-file"
				bind:this={restoreFileInput}
				type="file"
				accept=".db"
				class="sr-only"
				onchange={onFileChange}
			/>
			<div class="flex flex-wrap items-center gap-3">
				<button type="button" class="btn-primary" onclick={pickRestoreFile}>
					{$_('admin.backups.restore.upload')}
				</button>
				{#if restoreFile}
					<span class="text-sm font-mono" style:color="var(--text-muted)">{restoreFile.name}</span>
					<button type="button" class="btn-ghost" onclick={clearRestoreFile}>
						{$_('admin.backups.restore.remove')}
					</button>
				{:else}
					<span class="text-sm" style:color="var(--text-muted)"
						>{$_('admin.backups.restore.no_file')}</span
					>
				{/if}
			</div>
		</div>

		<div class="max-w-xs">
			<label class="mb-1.5 block text-sm font-medium" for="restore-confirm">
				{$_('admin.backups.restore.confirm')}
			</label>
			<input
				id="restore-confirm"
				class="input"
				placeholder="RESTORE"
				bind:value={restoreConfirm}
				autocomplete="off"
			/>
		</div>

		<FormFeedback error={restoreFeedback.error} success={restoreFeedback.success} />
		<button type="submit" class="btn-primary" disabled={loading || !restoreReady}>
			{$_('admin.backups.restore.submit')}
		</button>
	</form>
</div>
