<script lang="ts">
	import { onMount } from 'svelte';
	import { _ } from 'svelte-i18n';
	import { getAdminDiagnostics, type AdminDiagnostics } from '$lib/api/client';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import { formatAPIDateTimeForDisplay } from '$lib/dates';
	import { reportPageLoadFailure } from '$lib/page-load';
	import { toast } from '$lib/toast';
	import { user } from '$lib/stores/auth';

	let diagnostics = $state<AdminDiagnostics | null>(null);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	const tz = $derived($user?.timezone ?? 'Europe/Moscow');

	const orderedFields: Array<[keyof AdminDiagnostics, string]> = [
		['app_version', 'app_version'],
		['build_commit', 'build_commit'],
		['build_time', 'build_time'],
		['db_migration_version', 'db_migration_version'],
		['install_method', 'install_method'],
		['previous_app_version', 'previous_app_version'],
		['go_version', 'go_version'],
		['os', 'os'],
		['arch', 'arch'],
		['uptime_seconds', 'uptime_seconds'],
		['db_size_bytes', 'db_size_bytes'],
		['users_count', 'users_count'],
		['data_dir', 'data_dir'],
		['log_dir', 'log_dir'],
		['addr', 'addr'],
		['static_embed', 'static_embed'],
		['external_url', 'external_url']
	];

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			diagnostics = await getAdminDiagnostics();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: !!diagnostics });
			if (msg) loadError = msg;
		} finally {
			loading = false;
		}
	}

	function display(field: keyof AdminDiagnostics, value: unknown) {
		if (value === null || value === undefined || value === '') return '—';
		if (typeof value === 'boolean') return value ? 'true' : 'false';
		if (field === 'build_time' && typeof value === 'string') {
			return formatAPIDateTimeForDisplay(value, tz);
		}
		return String(value);
	}

	function detectProxy(externalURL: string): string {
		return externalURL.trim() ? 'nginx/custom' : 'Нет (прямое подключение к порту 8765)';
	}

	function buildReportText(data: AdminDiagnostics): string {
		const envLine = `${data.os}/${data.arch}, browser: <укажите браузер>`;
		const upgradeFrom = data.previous_app_version?.trim() || 'новая установка';
		const configLines = Object.entries(data.env)
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([k, v]) => `${k}=${v}`)
			.join('\n');
		return [
			`${$_('admin.diagnostics.summary.version')}: ${data.app_version || 'unknown'}`,
			`${$_('admin.diagnostics.summary.upgrade_from')}: ${upgradeFrom}`,
			`${$_('admin.diagnostics.summary.install_method')}: ${data.install_method || 'unknown'}`,
			`${$_('admin.diagnostics.summary.environment')}: ${envLine}`,
			`${$_('admin.diagnostics.summary.proxy')}: ${detectProxy(data.external_url || '')}`,
			`${$_('admin.diagnostics.summary.config')}:\n${configLines || '—'}`
		].join('\n');
	}

	async function copyForReport() {
		if (!diagnostics) return;
		try {
			await navigator.clipboard.writeText(buildReportText(diagnostics));
			toast($_('admin.diagnostics.copied'));
		} catch {
			toast.error($_('admin.diagnostics.failed_copy'));
		}
	}
</script>

<PageLoadGate {loading} error={loadError} onretry={() => void load()}>
	{#if diagnostics}
		{@const envText = Object.entries(diagnostics.env)
			.sort(([a], [b]) => a.localeCompare(b))
			.map(([k, v]) => `${k}=${v}`)
			.join('\n')}
		<div class="space-y-8">
			<section class="space-y-3">
				<div class="flex flex-wrap items-end justify-between gap-3">
					<div class="space-y-1 px-1">
						<h2 class="text-base font-semibold">{$_('admin.diagnostics.title')}</h2>
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('admin.diagnostics.hint')}
						</p>
					</div>
					<button type="button" class="btn-primary" onclick={copyForReport}>
						{$_('admin.diagnostics.copy')}
					</button>
				</div>
				<div class="card overflow-hidden p-0 sm:p-0">
					<table class="hidden w-full text-left text-sm md:table">
						<tbody>
							{#each orderedFields as [field, label] (field)}
								<tr class="border-t first:border-t-0" style:border-color="var(--border)">
									<td class="w-64 px-4 py-2.5 font-mono text-xs" style:color="var(--text-muted)"
										>{label}</td
									>
									<td class="break-all px-4 py-2.5">{display(field, diagnostics[field])}</td>
								</tr>
							{/each}
						</tbody>
					</table>
					<dl class="list-divided md:hidden">
						{#each orderedFields as [field, label] (field)}
							<div class="px-4 py-2.5">
								<dt class="font-mono text-xs" style:color="var(--text-muted)">{label}</dt>
								<dd class="mt-0.5 break-all text-sm">{display(field, diagnostics[field])}</dd>
							</div>
						{/each}
					</dl>
				</div>
			</section>

			<section class="space-y-3">
				<div class="space-y-1 px-1">
					<h2 class="text-base font-semibold">{$_('admin.diagnostics.env')}</h2>
					<p class="text-sm" style:color="var(--text-muted)">{$_('admin.diagnostics.env_hint')}</p>
				</div>
				<div class="card p-0 sm:p-0">
					<pre class="overflow-x-auto p-4 text-xs leading-relaxed">{envText}</pre>
				</div>
			</section>
		</div>
	{/if}
</PageLoadGate>
