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
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import GroupHeading from '$lib/components/GroupHeading.svelte';
	import EmptyStateCard from '$lib/components/EmptyStateCard.svelte';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { reportPageLoadFailure } from '$lib/page-load';
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
	let loading = $state(false);
	let pageLoading = $state(true);
	let loadError = $state<string | null>(null);
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
		pageLoading = true;
		try {
			const [list, settings] = await Promise.all([listBackups(), getBackupSettings()]);
			files = list;
			backupEnabled = settings.backup_enabled;
			backupTime = settings.backup_time;
			backupRetention = settings.backup_retention;
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: files.length > 0 });
			if (msg) loadError = msg;
		} finally {
			pageLoading = false;
		}
	}

	async function saveSettings(e: Event) {
		e.preventDefault();
		loading = true;
		try {
			await putBackupSettings({
				backup_enabled: backupEnabled,
				backup_time: backupTime,
				backup_retention: backupRetention
			});
			toast($_('admin.backups.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function manualBackup() {
		loading = true;
		try {
			await runBackup();
			await refresh();
			toast($_('admin.backups.created'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function handleRestore(e: Event) {
		e.preventDefault();
		if (!restoreFile) {
			toast.error($_('admin.backups.pick_file'));
			return;
		}
		if (restoreConfirm !== 'RESTORE') {
			toast.error($_('admin.backups.restore_type'));
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
			toast.fromError(err);
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

<PageLoadGate loading={pageLoading} error={loadError} onretry={() => void refresh()}>
	<div class="space-y-8">
		<form class="card space-y-4" onsubmit={saveSettings}>
			<div class="space-y-1">
				<h2 class="font-semibold">{$_('admin.backups.schedule')}</h2>
				<p class="text-sm" style:color="var(--text-muted)">{$_('admin.backups.schedule_hint')}</p>
			</div>
			<div class="flex items-center justify-between gap-4">
				<span class="text-sm font-medium">{$_('admin.backups.auto')}</span>
				<ToggleSwitch
					checked={backupEnabled}
					label={$_('admin.backups.auto')}
					onchange={() => (backupEnabled = !backupEnabled)}
				/>
			</div>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="field-label" for="time">{$_('admin.backups.time')}</label>
					<input id="time" type="time" class="input" bind:value={backupTime} />
				</div>
				<div>
					<label class="field-label" for="retention">{$_('admin.backups.retention')}</label>
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
			<div
				class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
				style:border-color="var(--border)"
			>
				<button
					type="button"
					class="btn-ghost sm:mr-auto"
					disabled={loading}
					onclick={manualBackup}
				>
					{$_('admin.backups.run_now')}
				</button>
				<button type="submit" class="btn-primary" disabled={loading}>{$_('common.save')}</button>
			</div>
		</form>

		<section class="space-y-3">
			<GroupHeading
				title={$_('admin.backups.group.archive', { values: { count: files.length } })}
				hint={$_('admin.backups.group.archive_hint')}
			/>
			{#if files.length === 0}
				<EmptyStateCard message={$_('admin.backups.empty')} />
			{:else}
				<div class="card overflow-hidden p-0 sm:p-0">
					<table class="hidden w-full text-left text-sm md:table">
						<thead>
							<tr class="text-xs uppercase tracking-wide" style:color="var(--text-muted)">
								<th class="px-4 py-3 font-medium">{$_('admin.backups.col.file')}</th>
								<th class="px-3 py-3 font-medium">{$_('admin.backups.col.size')}</th>
								<th class="px-3 py-3 font-medium">{$_('admin.backups.col.date')}</th>
								<th class="px-3 py-3"></th>
							</tr>
						</thead>
						<tbody>
							{#each files as f (f.filename)}
								<tr class="border-t" style:border-color="var(--border)">
									<td class="px-4 py-2.5 font-mono text-xs">{f.filename}</td>
									<td class="px-3 py-2.5 tabular-nums">{formatSize(f.size)}</td>
									<td class="px-3 py-2.5">{formatAPIDateTimeForDisplay(f.created_at, tz)}</td>
									<td class="px-3 py-2 text-right">
										<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- API download endpoint -->
										<a class="btn-ghost inline-block" href={backupDownloadUrl(f.filename)}>
											{$_('admin.backups.download')}
										</a>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
					<ul class="list-divided md:hidden">
						{#each files as f (f.filename)}
							<li class="flex items-center gap-3 px-4 py-2.5">
								<div class="min-w-0 flex-1">
									<p class="break-all font-mono text-xs">{f.filename}</p>
									<p class="mt-0.5 text-xs" style:color="var(--text-muted)">
										{formatAPIDateTimeForDisplay(f.created_at, tz)} · {formatSize(f.size)}
									</p>
								</div>
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- API download endpoint -->
								<a class="btn-ghost shrink-0" href={backupDownloadUrl(f.filename)}>
									{$_('admin.backups.download')}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		</section>

		<section class="space-y-3">
			<GroupHeading title={$_('settings.profile.section.danger')} />
			<form
				class="card space-y-4"
				style:border-color="color-mix(in srgb, var(--danger) 40%, var(--border))"
				onsubmit={handleRestore}
			>
				<div class="space-y-1">
					<h3 class="font-semibold">{$_('admin.backups.restore')}</h3>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('admin.backups.restore.hint')}
					</p>
				</div>

				<div class="space-y-2">
					<label class="block text-sm font-medium" for="restore-file">
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
						<button type="button" class="btn-ghost" onclick={pickRestoreFile}>
							{$_('admin.backups.restore.upload')}
						</button>
						{#if restoreFile}
							<span class="font-mono text-sm" style:color="var(--text-muted)"
								>{restoreFile.name}</span
							>
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
					<label class="field-label" for="restore-confirm">
						{$_('admin.backups.restore.confirm')}
					</label>
					<input
						id="restore-confirm"
						class="input font-mono"
						placeholder="RESTORE"
						bind:value={restoreConfirm}
						autocomplete="off"
					/>
				</div>

				<div class="flex justify-end border-t pt-4" style:border-color="var(--border)">
					<button type="submit" class="btn-danger" disabled={loading || !restoreReady}>
						{$_('admin.backups.restore.submit')}
					</button>
				</div>
			</form>
		</section>
	</div>
</PageLoadGate>
