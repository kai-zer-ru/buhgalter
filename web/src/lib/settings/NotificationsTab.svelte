<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import {
		getNotificationSettings,
		getUserSettings,
		previewNotificationTemplate,
		putNotificationSettings,
		resetNotificationTemplates,
		sendNotificationTest,
		type NotificationTemplate
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let timezone = $state('Europe/Moscow');
	let notificationsLoaded = $state(false);
	let notificationSecretConfigured = $state(false);
	let telegramEnabled = $state(false);
	let telegramConfigured = $state(false);
	let telegramTokenStored = $state(false);
	let telegramBotToken = $state('');
	let telegramChatId = $state('');
	let maxEnabled = $state(false);
	let maxConfigured = $state(false);
	let maxTokenStored = $state(false);
	let maxProvider = $state<'a161' | 'official'>('a161');
	let maxToken = $state('');
	let maxUserId = $state('');
	let maxRecipientId = $state('');
	let triggerDebt = $state(true);
	let triggerCredit = $state(true);
	let triggerPlanned = $state(true);
	let triggerPasswordReset = $state(true);
	let debtDaysBefore = $state(1);
	let myDebtOverdueDaysLimit = $state(7);
	let owedDebtOverdueStartAfterDays = $state(0);
	let owedDebtOverdueDaysLimit = $state(7);
	let creditDaysBefore = $state(1);
	let notificationTimeLocal = $state('00:00');
	let templates = $state<NotificationTemplate[]>([]);
	let templatesDirty = $state<Record<string, string>>({});
	let previewText = $state<Record<string, string>>({});
	let persistedNotificationState = $state({
		telegramEnabled: false,
		telegramChatId: '',
		maxEnabled: false,
		maxProvider: 'a161' as 'a161' | 'official',
		maxUserId: '',
		maxRecipientId: ''
	});

	const triggerOrder = [
		'debt_overdue',
		'debt_due_soon',
		'credit_payment',
		'planned_operation',
		'password_reset',
		'test'
	];

	onMount(() => {
		void (async () => {
			try {
				const s = await getUserSettings();
				timezone = s.timezone;
				await loadNotifications();
			} catch (err) {
				toast.fromError(err);
			}
		})();
	});

	async function loadNotifications() {
		const data = await getNotificationSettings();
		notificationsLoaded = true;
		notificationSecretConfigured = data.secret_key_configured === true;
		telegramEnabled = data.telegram_enabled;
		telegramConfigured = data.telegram_configured;
		telegramTokenStored = data.telegram_configured;
		telegramChatId = data.telegram_chat_id ?? '';
		maxEnabled = data.max_enabled;
		maxConfigured = data.max_configured;
		maxTokenStored = data.max_configured;
		maxProvider = data.max_provider === 'official' ? 'official' : 'a161';
		maxUserId = data.max_user_id ? String(data.max_user_id) : '';
		maxRecipientId = data.max_recipient_id ? String(data.max_recipient_id) : '';
		triggerDebt = data.trigger_debt;
		triggerCredit = data.trigger_credit;
		triggerPlanned = data.trigger_planned;
		triggerPasswordReset = data.trigger_password_reset ?? true;
		debtDaysBefore = data.debt_days_before;
		myDebtOverdueDaysLimit = data.my_debt_overdue_days_limit ?? 7;
		owedDebtOverdueStartAfterDays = data.owed_debt_overdue_start_after_days ?? 0;
		owedDebtOverdueDaysLimit = data.owed_debt_overdue_days_limit ?? 7;
		creditDaysBefore = data.credit_days_before;
		notificationTimeLocal = data.notification_time_local ?? '00:00';
		templates = data.templates;
		templatesDirty = {};
		previewText = {};
		persistedNotificationState = {
			telegramEnabled: data.telegram_enabled,
			telegramChatId: data.telegram_chat_id ?? '',
			maxEnabled: data.max_enabled,
			maxProvider: data.max_provider === 'official' ? 'official' : 'a161',
			maxUserId: data.max_user_id ? String(data.max_user_id) : '',
			maxRecipientId: data.max_recipient_id ? String(data.max_recipient_id) : ''
		};
	}

	function channelHasUnsavedChanges(channel: 'telegram' | 'max') {
		if (channel === 'telegram') {
			return (
				telegramBotToken.trim() !== '' ||
				telegramEnabled !== persistedNotificationState.telegramEnabled ||
				telegramChatId.trim() !== persistedNotificationState.telegramChatId
			);
		}
		return (
			maxToken.trim() !== '' ||
			maxEnabled !== persistedNotificationState.maxEnabled ||
			maxProvider !== persistedNotificationState.maxProvider ||
			maxUserId.trim() !== persistedNotificationState.maxUserId ||
			maxRecipientId.trim() !== persistedNotificationState.maxRecipientId
		);
	}

	function templateValue(triggerType: string, fallbackValue: string) {
		return templatesDirty[triggerType] ?? fallbackValue;
	}

	function templateTextareaId(triggerType: string) {
		return `tpl-${triggerType}`;
	}

	function triggerLabel(triggerType: string) {
		return $_(`settings.notifications.trigger.${triggerType}`);
	}

	function orderedTemplates(list: NotificationTemplate[]) {
		return [...list].sort((a, b) => {
			const ai = triggerOrder.indexOf(a.trigger_type);
			const bi = triggerOrder.indexOf(b.trigger_type);
			return (ai < 0 ? Number.MAX_SAFE_INTEGER : ai) - (bi < 0 ? Number.MAX_SAFE_INTEGER : bi);
		});
	}

	function validateTemplateText(template: string) {
		if (template.trim().length === 0) return $_('settings.notifications.error.template_empty');
		if (template.length > 500) return $_('settings.notifications.error.template_too_long');
		return '';
	}

	function validateDaysRange(value: number) {
		return Number.isFinite(value) && value >= 0 && value <= 30;
	}

	function validateOverdueDaysLimit(value: number) {
		return Number.isFinite(value) && value >= 0 && value <= 365;
	}

	function validateLocalTime(value: string) {
		return /^([01]\d|2[0-3]):([0-5]\d)$/.test(value.trim());
	}

	function updateTemplate(triggerType: string, value: string) {
		templatesDirty = { ...templatesDirty, [triggerType]: value };
	}

	async function insertPlaceholder(triggerType: string, placeholder: string) {
		const current =
			templatesDirty[triggerType] ??
			templates.find((t) => t.trigger_type === triggerType)?.template ??
			'';
		const textarea = document.getElementById(
			templateTextareaId(triggerType)
		) as HTMLTextAreaElement | null;
		const token = `{${placeholder}}`;
		const start = textarea ? (textarea.selectionStart ?? current.length) : current.length;
		const end = textarea ? (textarea.selectionEnd ?? current.length) : current.length;
		const next = `${current.slice(0, start)}${token}${current.slice(end)}`;
		updateTemplate(triggerType, next);
		await tick();
		const updated = document.getElementById(
			templateTextareaId(triggerType)
		) as HTMLTextAreaElement | null;
		if (updated) {
			const cursor = start + token.length;
			updated.focus();
			updated.setSelectionRange(cursor, cursor);
		}
	}

	async function saveNotifications(e: Event) {
		e.preventDefault();
		if (!notificationSecretConfigured) return;
		if (
			!validateDaysRange(debtDaysBefore) ||
			!validateDaysRange(creditDaysBefore) ||
			!validateOverdueDaysLimit(myDebtOverdueDaysLimit) ||
			!validateOverdueDaysLimit(owedDebtOverdueStartAfterDays) ||
			!validateOverdueDaysLimit(owedDebtOverdueDaysLimit)
		) {
			toast.error($_('settings.notifications.error.policy_range'));
			return;
		}
		if (!validateLocalTime(notificationTimeLocal)) {
			toast.error($_('settings.notifications.error.time_format'));
			return;
		}
		loading = true;
		try {
			await putNotificationSettings({
				telegram_enabled: telegramEnabled,
				telegram_bot_token: telegramBotToken.trim() || undefined,
				telegram_chat_id: telegramChatId.trim() || undefined,
				max_enabled: maxEnabled,
				max_provider: maxProvider,
				max_token: maxToken.trim() || undefined,
				max_user_id: maxUserId.trim() ? Number(maxUserId) : null,
				max_recipient_id: maxRecipientId.trim() ? Number(maxRecipientId) : null,
				trigger_debt: triggerDebt,
				trigger_credit: triggerCredit,
				trigger_planned: triggerPlanned,
				trigger_password_reset: $user?.is_admin ? triggerPasswordReset : undefined,
				debt_days_before: debtDaysBefore,
				my_debt_overdue_days_limit: myDebtOverdueDaysLimit,
				owed_debt_overdue_start_after_days: owedDebtOverdueStartAfterDays,
				owed_debt_overdue_days_limit: owedDebtOverdueDaysLimit,
				credit_days_before: creditDaysBefore,
				notification_time_local: notificationTimeLocal.trim()
			});
			telegramBotToken = '';
			maxToken = '';
			await loadNotifications();
			toast($_('settings.notifications.success.saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function runChannelTest(channel: 'telegram' | 'max') {
		if (!notificationSecretConfigured) return;
		if (channelHasUnsavedChanges(channel)) {
			toast.error($_('settings.notifications.error.save_before_test'));
			return;
		}
		loading = true;
		try {
			await sendNotificationTest(channel);
			toast($_('settings.notifications.success.test_sent', { values: { channel } }));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveTelegramBlock() {
		if (!notificationSecretConfigured) return;
		loading = true;
		try {
			await putNotificationSettings({
				telegram_enabled: telegramEnabled,
				telegram_bot_token: telegramBotToken.trim() || undefined,
				telegram_chat_id: telegramChatId.trim() || undefined
			});
			telegramBotToken = '';
			await loadNotifications();
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveMaxBlock() {
		if (!notificationSecretConfigured) return;
		loading = true;
		try {
			await putNotificationSettings({
				max_enabled: maxEnabled,
				max_provider: maxProvider,
				max_token: maxToken.trim() || undefined,
				max_user_id: maxUserId.trim() ? Number(maxUserId) : null,
				max_recipient_id: maxRecipientId.trim() ? Number(maxRecipientId) : null
			});
			maxToken = '';
			await loadNotifications();
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveTriggerTypesBlock() {
		if (!notificationSecretConfigured) return;
		loading = true;
		try {
			await putNotificationSettings({
				trigger_debt: triggerDebt,
				trigger_credit: triggerCredit,
				trigger_planned: triggerPlanned,
				trigger_password_reset: $user?.is_admin ? triggerPasswordReset : undefined
			});
			await loadNotifications();
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveTriggerPolicyBlock() {
		if (!notificationSecretConfigured) return;
		if (
			!validateDaysRange(debtDaysBefore) ||
			!validateDaysRange(creditDaysBefore) ||
			!validateOverdueDaysLimit(myDebtOverdueDaysLimit) ||
			!validateOverdueDaysLimit(owedDebtOverdueStartAfterDays) ||
			!validateOverdueDaysLimit(owedDebtOverdueDaysLimit)
		) {
			toast.error($_('settings.notifications.error.policy_range'));
			return;
		}
		if (!validateLocalTime(notificationTimeLocal)) {
			toast.error($_('settings.notifications.error.time_format'));
			return;
		}
		loading = true;
		try {
			await putNotificationSettings({
				debt_days_before: debtDaysBefore,
				my_debt_overdue_days_limit: myDebtOverdueDaysLimit,
				owed_debt_overdue_start_after_days: owedDebtOverdueStartAfterDays,
				owed_debt_overdue_days_limit: owedDebtOverdueDaysLimit,
				credit_days_before: creditDaysBefore,
				notification_time_local: notificationTimeLocal.trim()
			});
			await loadNotifications();
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function saveTemplate(triggerType: string, template: string) {
		if (!notificationSecretConfigured) return;
		const templateError = validateTemplateText(template);
		if (templateError) {
			toast.error(templateError);
			return;
		}
		loading = true;
		try {
			await putNotificationSettings({
				templates: [{ trigger_type: triggerType, template }]
			});
			await loadNotifications();
			const success = $_('settings.notifications.success.template_saved', {
				values: { trigger: triggerLabel(triggerType) }
			});
			toast(success);
		} catch (err) {
			toast.fromError(err);
		} finally {
			loading = false;
		}
	}

	async function previewTemplate(triggerType: string, template: string) {
		if (!notificationSecretConfigured) return;
		const templateError = validateTemplateText(template);
		if (templateError) {
			toast.error(templateError);
			return;
		}
		try {
			const result = await previewNotificationTemplate({ trigger_type: triggerType, template });
			previewText = { ...previewText, [triggerType]: result.text };
		} catch (err) {
			toast.fromError(err);
		}
	}

	async function resetTemplate(triggerType: string) {
		if (!notificationSecretConfigured) return;
		try {
			await resetNotificationTemplates(triggerType);
			await loadNotifications();
			toast(
				$_('settings.notifications.success.template_reset', {
					values: { trigger: triggerLabel(triggerType) }
				})
			);
		} catch (err) {
			toast.fromError(err);
		}
	}
</script>

{#if !notificationsLoaded}
	<div class="card">{$_('common.loading')}</div>
{:else}
	{#if !notificationSecretConfigured}
		<div
			class="card mb-4 space-y-3 border-2 p-5"
			style:border-color="color-mix(in srgb, var(--danger) 45%, var(--border))"
			role="alert"
		>
			<h3 class="text-lg font-semibold">{$_('settings.notifications.secret_missing.title')}</h3>
			{#if $user?.is_admin}
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('settings.notifications.secret_missing.body')}
				</p>
				<a href={resolve('/admin')} class="btn-primary inline-flex w-fit">
					{$_('settings.notifications.secret_missing.link')}
				</a>
			{:else}
				<p class="text-sm" style:color="var(--text-muted)">
					{$_('settings.notifications.secret_missing.body_non_admin')}
				</p>
			{/if}
		</div>
	{/if}

	<div
		class="space-y-6"
		class:pointer-events-none={!notificationSecretConfigured}
		class:opacity-50={!notificationSecretConfigured}
		class:select-none={!notificationSecretConfigured}
		aria-hidden={!notificationSecretConfigured ? true : undefined}
		inert={!notificationSecretConfigured}
	>
		<form class="space-y-6" onsubmit={saveNotifications}>
			<div class="space-y-6">
				<div class="card space-y-4">
					<h3 class="text-lg font-semibold">{$_('settings.notifications.channel.telegram')}</h3>
					<div class="flex items-center justify-between gap-4">
						<span class="text-sm">{$_('settings.notifications.telegram.enable')}</span>
						<ToggleSwitch
							checked={telegramEnabled}
							label={$_('settings.notifications.telegram.enable')}
							onchange={() => (telegramEnabled = !telegramEnabled)}
						/>
					</div>
					<div class="grid gap-3 md:grid-cols-2">
						<input
							class="input"
							placeholder={telegramTokenStored && telegramBotToken.trim() === ''
								? '********'
								: $_('settings.notifications.telegram.bot_token')}
							bind:value={telegramBotToken}
						/>
						<input
							class="input"
							placeholder={$_('settings.notifications.telegram.chat_id')}
							bind:value={telegramChatId}
						/>
					</div>
					{#if telegramTokenStored}
						<p class="text-xs" style:color="var(--text-muted)">
							{$_('settings.notifications.token_masked_hint')}
						</p>
					{/if}
					<div class="space-y-1 text-xs" style:color="var(--text-muted)">
						<p>{$_('settings.notifications.telegram.chat_id_help.title')}</p>
						<p>{$_('settings.notifications.telegram.chat_id_help.step1')}</p>
						<p>
							{$_('settings.notifications.telegram.chat_id_help.step2_prefix')}
							<a
								href="https://api.telegram.org/bot&lt;BOT_TOKEN&gt;/getUpdates"
								target="_blank"
								rel="noreferrer noopener"
								class="underline">api.telegram.org/bot&lt;BOT_TOKEN&gt;/getUpdates</a
							>.
						</p>
						<p>{$_('settings.notifications.telegram.chat_id_help.step3')}</p>
					</div>
					<div class="flex items-center gap-2 text-sm" style:color="var(--text-muted)">
						<span>
							{$_('settings.notifications.status.label')}
							{telegramConfigured
								? $_('settings.notifications.status.configured')
								: $_('settings.notifications.status.not_configured')}
						</span>
						<button
							type="button"
							class="btn-ghost"
							onclick={() => runChannelTest('telegram')}
							disabled={loading}
						>
							{$_('settings.notifications.test_send')}
						</button>
						<button
							type="button"
							class="btn-primary"
							onclick={saveTelegramBlock}
							disabled={loading}
						>
							{$_('settings.notifications.block_save')}
						</button>
					</div>
				</div>

				<div class="card space-y-4">
					<h3 class="text-lg font-semibold">{$_('settings.notifications.channel.max')}</h3>
					<div class="flex items-center justify-between gap-4">
						<span class="text-sm">{$_('settings.notifications.max.enable')}</span>
						<ToggleSwitch
							checked={maxEnabled}
							label={$_('settings.notifications.max.enable')}
							onchange={() => (maxEnabled = !maxEnabled)}
						/>
					</div>
					<div class="grid gap-3 md:grid-cols-2">
						<Select
							label=""
							controlled
							value={maxProvider}
							onchange={(next) => (maxProvider = next as 'a161' | 'official')}
							options={[
								{ value: 'a161', label: 'a161' },
								{ value: 'official', label: 'official' }
							]}
						/>
						<input
							class="input"
							placeholder={maxTokenStored && maxToken.trim() === ''
								? '********'
								: $_('settings.notifications.max.token')}
							bind:value={maxToken}
						/>
						{#if maxProvider === 'a161'}
							<input
								class="input"
								placeholder={$_('settings.notifications.max.user_id')}
								bind:value={maxUserId}
							/>
						{:else}
							<input
								class="input"
								placeholder={$_('settings.notifications.max.recipient_id')}
								bind:value={maxRecipientId}
							/>
						{/if}
					</div>
					{#if maxTokenStored}
						<p class="text-xs" style:color="var(--text-muted)">
							{$_('settings.notifications.token_masked_hint')}
						</p>
					{/if}
					<p class="text-xs" style:color="var(--text-muted)">
						{$_('settings.notifications.max.a161_link_prefix')}
						<a
							href="https://notify.a161.ru"
							target="_blank"
							rel="noreferrer noopener"
							class="underline">notify.a161.ru</a
						>.
					</p>
					<div class="flex items-center gap-2 text-sm" style:color="var(--text-muted)">
						<span>
							{$_('settings.notifications.status.label')}
							{maxConfigured
								? $_('settings.notifications.status.configured')
								: $_('settings.notifications.status.not_configured')}
						</span>
						<button
							type="button"
							class="btn-ghost"
							onclick={() => runChannelTest('max')}
							disabled={loading}
						>
							{$_('settings.notifications.test_send')}
						</button>
						<button type="button" class="btn-primary" onclick={saveMaxBlock} disabled={loading}>
							{$_('settings.notifications.block_save')}
						</button>
					</div>
				</div>

				<div class="card space-y-4">
					<h3 class="text-lg font-semibold">{$_('settings.notifications.triggers.title')}</h3>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('settings.notifications.triggers.types_hint')}
					</p>
					<div class="grid w-full max-w-md gap-2">
						<div class="grid grid-cols-[1fr_auto] items-center gap-3">
							<span class="text-sm leading-tight">{$_('settings.notifications.triggers.debt')}</span
							>
							<ToggleSwitch
								checked={triggerDebt}
								label={$_('settings.notifications.triggers.debt')}
								onchange={() => (triggerDebt = !triggerDebt)}
							/>
						</div>
						<div class="grid grid-cols-[1fr_auto] items-center gap-3">
							<span class="text-sm leading-tight"
								>{$_('settings.notifications.triggers.credit')}</span
							>
							<ToggleSwitch
								checked={triggerCredit}
								label={$_('settings.notifications.triggers.credit')}
								onchange={() => (triggerCredit = !triggerCredit)}
							/>
						</div>
						<div class="grid grid-cols-[1fr_auto] items-center gap-3">
							<span class="text-sm leading-tight"
								>{$_('settings.notifications.triggers.planned')}</span
							>
							<ToggleSwitch
								checked={triggerPlanned}
								label={$_('settings.notifications.triggers.planned')}
								onchange={() => (triggerPlanned = !triggerPlanned)}
							/>
						</div>
						{#if $user?.is_admin}
							<div class="grid grid-cols-[1fr_auto] items-center gap-3">
								<span class="text-sm leading-tight"
									>{$_('settings.notifications.triggers.passwordReset')}</span
								>
								<ToggleSwitch
									checked={triggerPasswordReset}
									label={$_('settings.notifications.triggers.passwordReset')}
									onchange={() => (triggerPasswordReset = !triggerPasswordReset)}
								/>
							</div>
						{/if}
					</div>
					<div class="flex items-center gap-2">
						<button
							type="button"
							class="btn-primary"
							onclick={saveTriggerTypesBlock}
							disabled={loading}
						>
							{$_('settings.notifications.block_save')}
						</button>
					</div>
				</div>

				<div class="card space-y-6">
					<div class="space-y-1">
						<h3 class="text-lg font-semibold">
							{$_('settings.notifications.triggers.policy_title')}
						</h3>
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('settings.notifications.triggers.policy_hint')}
						</p>
					</div>

					<section class="space-y-3">
						<h4 class="text-sm font-medium">
							{$_('settings.notifications.triggers.debt_policy_title')}
						</h4>
						<div class="grid gap-4 sm:grid-cols-2">
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug"
									>{$_('settings.notifications.triggers.my_debt_before_days')}</span
								>
								<input
									class="input w-full"
									type="number"
									min="0"
									max="30"
									bind:value={debtDaysBefore}
								/>
							</label>
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug"
									>{$_('settings.notifications.triggers.my_debt_overdue_limit_days')}</span
								>
								<input
									class="input w-full"
									type="number"
									min="0"
									max="365"
									bind:value={myDebtOverdueDaysLimit}
								/>
							</label>
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug"
									>{$_('settings.notifications.triggers.owed_debt_start_after_days')}</span
								>
								<input
									class="input w-full"
									type="number"
									min="0"
									max="365"
									bind:value={owedDebtOverdueStartAfterDays}
								/>
							</label>
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug"
									>{$_('settings.notifications.triggers.owed_debt_overdue_limit_days')}</span
								>
								<input
									class="input w-full"
									type="number"
									min="0"
									max="365"
									bind:value={owedDebtOverdueDaysLimit}
								/>
							</label>
						</div>
					</section>

					<section class="space-y-3">
						<h4 class="text-sm font-medium">
							{$_('settings.notifications.triggers.credit_policy_title')}
						</h4>
						<label class="block max-w-xs space-y-1.5 text-sm">
							<span class="block leading-snug"
								>{$_('settings.notifications.triggers.credit_days')}</span
							>
							<input
								class="input w-full"
								type="number"
								min="0"
								max="30"
								bind:value={creditDaysBefore}
							/>
						</label>
					</section>

					<section class="space-y-3 border-t pt-6" style:border-color="var(--border)">
						<h4 class="text-sm font-medium">
							{$_('settings.notifications.triggers.schedule_title')}
						</h4>
						<label class="block max-w-xs space-y-1.5 text-sm">
							<span class="block leading-snug"
								>{$_('settings.notifications.triggers.send_time')}</span
							>
							<input class="input w-full" type="time" bind:value={notificationTimeLocal} />
						</label>
						<p class="max-w-xl text-xs leading-relaxed" style:color="var(--text-muted)">
							{$_('settings.notifications.triggers.send_time_hint', { values: { timezone } })}
						</p>
					</section>

					<div class="flex items-center gap-2 border-t pt-4" style:border-color="var(--border)">
						<button
							type="button"
							class="btn-primary"
							onclick={saveTriggerPolicyBlock}
							disabled={loading}
						>
							{$_('settings.notifications.block_save')}
						</button>
					</div>
				</div>

				{#each orderedTemplates(templates) as tpl (tpl.trigger_type)}
					<div class="card space-y-4">
						<h3 class="text-lg font-semibold">{triggerLabel(tpl.trigger_type)}</h3>
						<textarea
							id={templateTextareaId(tpl.trigger_type)}
							class="input min-h-[88px]"
							value={templateValue(tpl.trigger_type, tpl.template)}
							oninput={(e) =>
								updateTemplate(tpl.trigger_type, (e.currentTarget as HTMLTextAreaElement).value)}
						></textarea>
						<p class="text-xs" style:color="var(--text-muted)">
							{$_('settings.notifications.templates.placeholders_hint')}
						</p>
						<div class="flex flex-wrap gap-2">
							{#each tpl.placeholders as placeholder (placeholder)}
								<button
									type="button"
									class="btn-ghost"
									onclick={() => insertPlaceholder(tpl.trigger_type, placeholder)}
								>
									{`{${placeholder}}`}
								</button>
							{/each}
						</div>
						<div class="flex flex-wrap items-center gap-2">
							<button
								type="button"
								class="btn-ghost"
								onclick={() =>
									previewTemplate(tpl.trigger_type, templateValue(tpl.trigger_type, tpl.template))}
							>
								{$_('settings.notifications.templates.preview')}
							</button>
							<button
								type="button"
								class="btn-ghost"
								onclick={() => resetTemplate(tpl.trigger_type)}
							>
								{$_('settings.notifications.templates.reset')}
							</button>
							<button
								type="button"
								class="btn-primary"
								onclick={() =>
									saveTemplate(tpl.trigger_type, templateValue(tpl.trigger_type, tpl.template))}
								disabled={loading}
							>
								{$_('settings.notifications.block_save')}
							</button>
						</div>
						{#if previewText[tpl.trigger_type]}
							<div
								class="rounded border p-2 text-sm"
								style:border-color="var(--border); color: var(--text-muted);"
							>
								<p class="mb-1 text-xs font-medium">
									{$_('settings.notifications.templates.preview_result')}
								</p>
								<p>{previewText[tpl.trigger_type]}</p>
							</div>
						{/if}
					</div>
				{/each}
				<button
					type="submit"
					class="btn-primary"
					disabled={loading || !notificationSecretConfigured}
					>{$_('settings.notifications.save_all')}</button
				>
			</div>
		</form>
	</div>
{/if}
