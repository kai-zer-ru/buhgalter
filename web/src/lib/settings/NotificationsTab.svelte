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
		type NotificationInlineKeyboard,
		type NotificationTemplate
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import PageLoadGate from '$lib/components/PageLoadGate.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import Select from '$lib/components/Select.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { featureFlags, isFeatureEnabled } from '$lib/features';
	import { reportPageLoadFailure } from '$lib/page-load';
	import KeyboardBuilder from '$lib/settings/KeyboardBuilder.svelte';
	import {
		emptyInlineKeyboard,
		ensureKeyboardIds,
		normalizeInlineKeyboard,
		validateInlineKeyboard,
		type KeyboardProvider,
		type KeyboardValidationError
	} from '$lib/settings/keyboard';
	import SettingsPanel from '$lib/settings/SettingsPanel.svelte';
	import { toast } from '$lib/toast';

	let loading = $state(false);
	let loadError = $state<string | null>(null);
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
	let triggerSubscription = $state(true);
	let triggerPlanned = $state(true);
	let triggerNegativeBalance = $state(true);
	let triggerBudget = $state(true);
	let triggerAutoTopupDisabled = $state(true);
	let triggerPasswordReset = $state(true);
	let debtDaysBefore = $state(1);
	let myDebtOverdueDaysLimit = $state(7);
	let owedDebtOverdueStartAfterDays = $state(0);
	let owedDebtOverdueDaysLimit = $state(7);
	let creditDaysBefore = $state(1);
	let subscriptionDaysBefore = $state(1);
	let notificationTimeLocal = $state('00:00');
	let templates = $state<NotificationTemplate[]>([]);
	let templatesDirty = $state<Record<string, string>>({});
	let templateKeyboards = $state<
		Record<string, { telegram: NotificationInlineKeyboard; max: NotificationInlineKeyboard }>
	>({});
	let templateKeyboardTab = $state<Record<string, KeyboardProvider>>({});
	let previewText = $state<Record<string, string>>({});
	let previewKeyboards = $state<
		Record<string, { telegram: NotificationInlineKeyboard; max: NotificationInlineKeyboard }>
	>({});
	let globalKeyboardTelegram = $state<NotificationInlineKeyboard>(emptyInlineKeyboard());
	let globalKeyboardMax = $state<NotificationInlineKeyboard>(emptyInlineKeyboard());
	/** True when this session minted ids for global buttons that are not stored yet. */
	let globalIdsDirty = $state(false);
	let globalKeyboardTab = $state<KeyboardProvider>('telegram');
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
		'subscription_charge',
		'planned_operation',
		'balance_shortfall',
		'budget_threshold',
		'auto_topup_disabled',
		'password_reset',
		'test'
	];

	type NotificationTriggerKey =
		| 'debt'
		| 'credit'
		| 'subscription'
		| 'planned'
		| 'negativeBalance'
		| 'budget'
		| 'autoTopupDisabled'
		| 'passwordReset';

	const templateSettingKey: Partial<Record<string, NotificationTriggerKey>> = {
		debt_overdue: 'debt',
		debt_due_soon: 'debt',
		credit_payment: 'credit',
		subscription_charge: 'subscription',
		planned_operation: 'planned',
		balance_shortfall: 'negativeBalance',
		budget_threshold: 'budget',
		auto_topup_disabled: 'autoTopupDisabled',
		password_reset: 'passwordReset'
	};

	function isNotificationTriggerEnabled(key: NotificationTriggerKey) {
		switch (key) {
			case 'debt':
				return triggerDebt;
			case 'credit':
				return triggerCredit;
			case 'subscription':
				return triggerSubscription;
			case 'planned':
				return triggerPlanned;
			case 'negativeBalance':
				return triggerNegativeBalance;
			case 'budget':
				return triggerBudget;
			case 'autoTopupDisabled':
				return triggerAutoTopupDisabled;
			case 'passwordReset':
				return triggerPasswordReset;
		}
	}

	function toggleNotificationTrigger(key: NotificationTriggerKey) {
		switch (key) {
			case 'debt':
				triggerDebt = !triggerDebt;
				break;
			case 'credit':
				triggerCredit = !triggerCredit;
				break;
			case 'subscription':
				triggerSubscription = !triggerSubscription;
				break;
			case 'planned':
				triggerPlanned = !triggerPlanned;
				break;
			case 'negativeBalance':
				triggerNegativeBalance = !triggerNegativeBalance;
				break;
			case 'budget':
				triggerBudget = !triggerBudget;
				break;
			case 'autoTopupDisabled':
				triggerAutoTopupDisabled = !triggerAutoTopupDisabled;
				break;
			case 'passwordReset':
				triggerPasswordReset = !triggerPasswordReset;
				break;
		}
	}

	function notificationTriggerRows(): Array<{ key: NotificationTriggerKey; hintKey: string }> {
		const flags = $featureFlags;
		const rows: Array<{ key: NotificationTriggerKey; hintKey: string }> = [];
		if (isFeatureEnabled('debts', flags)) rows.push({ key: 'debt', hintKey: 'debt_hint' });
		if (isFeatureEnabled('credits', flags)) rows.push({ key: 'credit', hintKey: 'credit_hint' });
		if (isFeatureEnabled('subscriptions', flags)) {
			rows.push({ key: 'subscription', hintKey: 'subscription_hint' });
		}
		rows.push({ key: 'planned', hintKey: 'planned_hint' });
		rows.push({ key: 'negativeBalance', hintKey: 'negativeBalance_hint' });
		if (isFeatureEnabled('budget', flags)) rows.push({ key: 'budget', hintKey: 'budget_hint' });
		if (isFeatureEnabled('balance_maintenance', flags)) {
			rows.push({ key: 'autoTopupDisabled', hintKey: 'autoTopupDisabled_hint' });
		}
		if ($user?.is_admin) {
			rows.push({ key: 'passwordReset', hintKey: 'passwordReset_hint' });
		}
		return rows;
	}

	onMount(() => {
		void reloadPage();
	});

	async function reloadPage() {
		try {
			const s = await getUserSettings();
			timezone = s.timezone;
			await loadNotifications();
			loadError = null;
		} catch (err) {
			const msg = reportPageLoadFailure(err, { hasData: notificationsLoaded });
			if (msg) loadError = msg;
		}
	}

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
		triggerSubscription =
			'trigger_subscription' in data ? (data.trigger_subscription ?? true) : true;
		triggerPlanned = data.trigger_planned;
		triggerNegativeBalance =
			'trigger_negative_balance' in data ? data.trigger_negative_balance : true;
		triggerBudget = 'trigger_budget' in data ? data.trigger_budget : true;
		triggerAutoTopupDisabled =
			'trigger_auto_topup_disabled' in data ? data.trigger_auto_topup_disabled : true;
		triggerPasswordReset = data.trigger_password_reset ?? true;
		debtDaysBefore = data.debt_days_before;
		myDebtOverdueDaysLimit = data.my_debt_overdue_days_limit ?? 7;
		owedDebtOverdueStartAfterDays = data.owed_debt_overdue_start_after_days ?? 0;
		owedDebtOverdueDaysLimit = data.owed_debt_overdue_days_limit ?? 7;
		creditDaysBefore = data.credit_days_before;
		subscriptionDaysBefore = data.subscription_days_before ?? 1;
		notificationTimeLocal = data.notification_time_local ?? '00:00';
		templates = data.templates;
		templatesDirty = {};
		templateKeyboards = Object.fromEntries(
			data.templates.map((tpl) => [
				tpl.trigger_type,
				{
					telegram: ensureKeyboardIds(tpl.keyboard_telegram),
					max: ensureKeyboardIds(tpl.keyboard_max)
				}
			])
		);
		templateKeyboardTab = {};
		previewText = {};
		previewKeyboards = {};
		globalIdsDirty =
			keyboardNeedsIds(data.keyboard_telegram) || keyboardNeedsIds(data.keyboard_max);
		globalKeyboardTelegram = ensureKeyboardIds(data.keyboard_telegram);
		globalKeyboardMax = ensureKeyboardIds(data.keyboard_max);
		persistedNotificationState = {
			telegramEnabled: data.telegram_enabled,
			telegramChatId: data.telegram_chat_id ?? '',
			maxEnabled: data.max_enabled,
			maxProvider: data.max_provider === 'official' ? 'official' : 'a161',
			maxUserId: data.max_user_id ? String(data.max_user_id) : '',
			maxRecipientId: data.max_recipient_id ? String(data.max_recipient_id) : ''
		};
	}

	function keyboardNeedsIds(kb: NotificationInlineKeyboard | null | undefined): boolean {
		return (kb?.rows ?? []).some((row) => row.some((btn) => !btn.id?.trim()));
	}

	function keyboardValidationMessage(err: KeyboardValidationError): string {
		return $_(`settings.notifications.keyboard.error.${err.code}`, {
			values: err.values ?? {}
		});
	}

	function ensureTemplateKeyboards(triggerType: string) {
		if (templateKeyboards[triggerType]) return templateKeyboards[triggerType];
		const tpl = templates.find((t) => t.trigger_type === triggerType);
		const next = {
			telegram: ensureKeyboardIds(tpl?.keyboard_telegram),
			max: ensureKeyboardIds(tpl?.keyboard_max)
		};
		templateKeyboards = { ...templateKeyboards, [triggerType]: next };
		return next;
	}

	function templateKeyboardTabFor(triggerType: string): KeyboardProvider {
		const preferred = templateKeyboardTab[triggerType] ?? 'telegram';
		if (preferred === 'telegram' && telegramEnabled) return 'telegram';
		if (preferred === 'max' && maxEnabled) return 'max';
		if (telegramEnabled) return 'telegram';
		return 'max';
	}

	function globalKeyboardTabFor(): KeyboardProvider {
		if (globalKeyboardTab === 'telegram' && telegramEnabled) return 'telegram';
		if (globalKeyboardTab === 'max' && maxEnabled) return 'max';
		if (telegramEnabled) return 'telegram';
		return 'max';
	}

	function setTemplateKeyboardTab(triggerType: string, tab: KeyboardProvider) {
		templateKeyboardTab = { ...templateKeyboardTab, [triggerType]: tab };
	}

	function setTemplateKeyboard(
		triggerType: string,
		provider: KeyboardProvider,
		kb: NotificationInlineKeyboard
	) {
		const current = ensureTemplateKeyboards(triggerType);
		templateKeyboards = {
			...templateKeyboards,
			[triggerType]: {
				...current,
				[provider]: normalizeInlineKeyboard(kb)
			}
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

	/** Header badge reflects the stored state, not unsaved edits in the form. */
	function channelBadge(
		configured: boolean,
		enabled: boolean
	): { label: string; tone: 'success' | 'muted' | 'warning' } {
		if (!configured) {
			return { label: $_('settings.notifications.channel_state.not_configured'), tone: 'warning' };
		}
		return enabled
			? { label: $_('settings.notifications.channel_state.on'), tone: 'success' }
			: { label: $_('settings.notifications.channel_state.off'), tone: 'muted' };
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

	function templateEditable(triggerType: string) {
		const settingKey = templateSettingKey[triggerType];
		if (!settingKey) return true;
		return isNotificationTriggerEnabled(settingKey);
	}

	function templateDisabledHint(triggerType: string) {
		const settingKey = templateSettingKey[triggerType];
		if (!settingKey) return '';
		return $_('settings.notifications.templates.disabled_setting', {
			values: { setting: $_(`settings.notifications.triggers.${settingKey}`) }
		});
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
				trigger_subscription: triggerSubscription,
				trigger_planned: triggerPlanned,
				trigger_negative_balance: triggerNegativeBalance,
				trigger_budget: triggerBudget,
				trigger_auto_topup_disabled: triggerAutoTopupDisabled,
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
			!validateDaysRange(subscriptionDaysBefore) ||
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
				subscription_days_before: subscriptionDaysBefore,
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

	async function saveGlobalKeyboards() {
		if (!notificationSecretConfigured) return;
		const payload: {
			keyboard_telegram?: NotificationInlineKeyboard;
			keyboard_max?: NotificationInlineKeyboard;
		} = {};
		if (telegramEnabled) {
			const tgErr = validateInlineKeyboard('telegram', globalKeyboardTelegram);
			if (tgErr) {
				toast.error(keyboardValidationMessage(tgErr));
				return;
			}
			payload.keyboard_telegram = normalizeInlineKeyboard(globalKeyboardTelegram);
		}
		if (maxEnabled) {
			const maxErr = validateInlineKeyboard('max', globalKeyboardMax);
			if (maxErr) {
				toast.error(keyboardValidationMessage(maxErr));
				return;
			}
			payload.keyboard_max = normalizeInlineKeyboard(globalKeyboardMax);
		}
		loading = true;
		try {
			await putNotificationSettings(payload);
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
		if (!templateEditable(triggerType)) return;
		const templateError = validateTemplateText(template);
		if (templateError) {
			toast.error(templateError);
			return;
		}
		const kbs = ensureTemplateKeyboards(triggerType);
		const tgErr = validateInlineKeyboard('telegram', kbs.telegram);
		if (tgErr) {
			toast.error(keyboardValidationMessage(tgErr));
			return;
		}
		const maxErr = validateInlineKeyboard('max', kbs.max);
		if (maxErr) {
			toast.error(keyboardValidationMessage(maxErr));
			return;
		}
		loading = true;
		try {
			await putNotificationSettings({
				...(globalIdsDirty
					? {
							keyboard_telegram: normalizeInlineKeyboard(globalKeyboardTelegram),
							keyboard_max: normalizeInlineKeyboard(globalKeyboardMax)
						}
					: {}),
				templates: [
					{
						trigger_type: triggerType,
						template,
						keyboard_telegram: normalizeInlineKeyboard(kbs.telegram),
						keyboard_max: normalizeInlineKeyboard(kbs.max)
					}
				]
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
		if (!templateEditable(triggerType)) return;
		const templateError = validateTemplateText(template);
		if (templateError) {
			toast.error(templateError);
			return;
		}
		const kbs = ensureTemplateKeyboards(triggerType);
		const tgErr = validateInlineKeyboard('telegram', kbs.telegram);
		if (tgErr) {
			toast.error(keyboardValidationMessage(tgErr));
			return;
		}
		const maxErr = validateInlineKeyboard('max', kbs.max);
		if (maxErr) {
			toast.error(keyboardValidationMessage(maxErr));
			return;
		}
		try {
			const result = await previewNotificationTemplate({
				trigger_type: triggerType,
				template,
				keyboard_telegram: normalizeInlineKeyboard(kbs.telegram),
				keyboard_max: normalizeInlineKeyboard(kbs.max)
			});
			previewText = { ...previewText, [triggerType]: result.text };
			previewKeyboards = {
				...previewKeyboards,
				[triggerType]: {
					telegram: normalizeInlineKeyboard(result.keyboard_telegram),
					max: normalizeInlineKeyboard(result.keyboard_max)
				}
			};
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

{#snippet groupHeading(title: string, hint: string)}
	<div class="space-y-1 px-1">
		<h2 class="text-base font-semibold">{title}</h2>
		<p class="text-sm" style:color="var(--text-muted)">{hint}</p>
	</div>
{/snippet}

<PageLoadGate
	loading={!notificationsLoaded && !loadError}
	error={loadError}
	onretry={() => void reloadPage()}
>
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
		class="space-y-8"
		class:pointer-events-none={!notificationSecretConfigured}
		class:opacity-50={!notificationSecretConfigured}
		class:select-none={!notificationSecretConfigured}
		aria-hidden={!notificationSecretConfigured ? true : undefined}
		inert={!notificationSecretConfigured}
	>
		<section class="space-y-3">
			{@render groupHeading(
				$_('settings.notifications.group.channels'),
				$_('settings.notifications.group.channels_hint')
			)}

			<SettingsPanel
				label={$_('settings.notifications.channel.telegram')}
				badge={channelBadge(telegramConfigured, telegramEnabled).label}
				badgeTone={channelBadge(telegramConfigured, telegramEnabled).tone}
			>
				<div class="space-y-5">
					<div class="flex items-center justify-between gap-4">
						<span class="text-sm font-medium">{$_('settings.notifications.telegram.enable')}</span>
						<ToggleSwitch
							checked={telegramEnabled}
							label={$_('settings.notifications.telegram.enable')}
							onchange={() => (telegramEnabled = !telegramEnabled)}
						/>
					</div>
					<div class="grid gap-4 md:grid-cols-2">
						<label class="block space-y-1.5 text-sm">
							<span class="block" style:color="var(--text-muted)"
								>{$_('settings.notifications.telegram.bot_token')}</span
							>
							<input
								class="input"
								autocomplete="off"
								placeholder={telegramTokenStored && telegramBotToken.trim() === ''
									? '********'
									: ''}
								bind:value={telegramBotToken}
							/>
						</label>
						<label class="block space-y-1.5 text-sm">
							<span class="block" style:color="var(--text-muted)"
								>{$_('settings.notifications.telegram.chat_id')}</span
							>
							<input class="input" autocomplete="off" bind:value={telegramChatId} />
						</label>
					</div>
					{#if telegramTokenStored}
						<p class="text-xs" style:color="var(--text-muted)">
							{$_('settings.notifications.token_masked_hint')}
						</p>
					{/if}
					<div
						class="space-y-1 rounded-xl border p-3 text-xs leading-relaxed"
						style:border-color="var(--border)"
						style:color="var(--text-muted)"
					>
						<p class="font-medium" style:color="var(--text)">
							{$_('settings.notifications.telegram.chat_id_help.title')}
						</p>
						<p>{$_('settings.notifications.telegram.chat_id_help.step1')}</p>
						<p class="break-words">
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
					<div
						class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
						style:border-color="var(--border)"
					>
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
			</SettingsPanel>

			<SettingsPanel
				label={$_('settings.notifications.channel.max')}
				badge={channelBadge(maxConfigured, maxEnabled).label}
				badgeTone={channelBadge(maxConfigured, maxEnabled).tone}
			>
				<div class="space-y-5">
					<div class="flex items-center justify-between gap-4">
						<span class="text-sm font-medium">{$_('settings.notifications.max.enable')}</span>
						<ToggleSwitch
							checked={maxEnabled}
							label={$_('settings.notifications.max.enable')}
							onchange={() => (maxEnabled = !maxEnabled)}
						/>
					</div>
					<div class="grid gap-4 md:grid-cols-3">
						<Select
							label={$_('settings.notifications.max.provider')}
							controlled
							value={maxProvider}
							onchange={(next) => (maxProvider = next as 'a161' | 'official')}
							options={[
								{ value: 'a161', label: 'a161' },
								{ value: 'official', label: 'official' }
							]}
						/>
						<label class="block space-y-1.5 text-sm">
							<span class="block" style:color="var(--text-muted)"
								>{$_('settings.notifications.max.token')}</span
							>
							<input
								class="input"
								autocomplete="off"
								placeholder={maxTokenStored && maxToken.trim() === '' ? '********' : ''}
								bind:value={maxToken}
							/>
						</label>
						{#if maxProvider === 'a161'}
							<label class="block space-y-1.5 text-sm">
								<span class="block" style:color="var(--text-muted)"
									>{$_('settings.notifications.max.user_id')}</span
								>
								<input class="input" inputmode="numeric" bind:value={maxUserId} />
							</label>
						{:else}
							<label class="block space-y-1.5 text-sm">
								<span class="block" style:color="var(--text-muted)"
									>{$_('settings.notifications.max.recipient_id')}</span
								>
								<input class="input" inputmode="numeric" bind:value={maxRecipientId} />
							</label>
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
					<div
						class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
						style:border-color="var(--border)"
					>
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
			</SettingsPanel>
		</section>

		<section class="space-y-3">
			{@render groupHeading(
				$_('settings.notifications.group.schedule'),
				$_('settings.notifications.group.schedule_hint')
			)}

			<SettingsPanel
				label={$_('settings.notifications.triggers.title')}
				description={$_('settings.notifications.triggers.types_hint')}
			>
				<div class="space-y-4">
					<ul class="divide-y" style:border-color="var(--border)">
						{#each notificationTriggerRows() as row (row.key)}
							<li
								class="flex items-start justify-between gap-4 py-3 first:pt-0"
								style:border-color="var(--border)"
							>
								<div class="min-w-0">
									<p class="text-sm font-medium leading-snug">
										{$_(`settings.notifications.triggers.${row.key}`)}
									</p>
									<p class="mt-0.5 text-sm leading-relaxed" style:color="var(--text-muted)">
										{$_(`settings.notifications.triggers.${row.hintKey}`)}
									</p>
								</div>
								<ToggleSwitch
									checked={isNotificationTriggerEnabled(row.key)}
									label={$_(`settings.notifications.triggers.${row.key}`)}
									onchange={() => toggleNotificationTrigger(row.key)}
								/>
							</li>
						{/each}
					</ul>
					<div
						class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
						style:border-color="var(--border)"
					>
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
			</SettingsPanel>

			<SettingsPanel
				label={$_('settings.notifications.triggers.policy_title')}
				description={$_('settings.notifications.triggers.policy_hint')}
			>
				<div class="space-y-6">
					<section class="space-y-3">
						<h3 class="text-sm font-semibold">
							{$_('settings.notifications.triggers.debt_policy_title')}
						</h3>
						<div class="grid gap-4 sm:grid-cols-2">
							<label class="flex flex-col gap-1.5 text-sm">
								<span class="leading-snug" style:color="var(--text-muted)"
									>{$_('settings.notifications.triggers.my_debt_before_days')}</span
								>
								<input
									class="input mt-auto w-full"
									type="number"
									min="0"
									max="30"
									bind:value={debtDaysBefore}
								/>
							</label>
							<label class="flex flex-col gap-1.5 text-sm">
								<span class="leading-snug" style:color="var(--text-muted)"
									>{$_('settings.notifications.triggers.my_debt_overdue_limit_days')}</span
								>
								<input
									class="input mt-auto w-full"
									type="number"
									min="0"
									max="365"
									bind:value={myDebtOverdueDaysLimit}
								/>
							</label>
							<label class="flex flex-col gap-1.5 text-sm">
								<span class="leading-snug" style:color="var(--text-muted)"
									>{$_('settings.notifications.triggers.owed_debt_start_after_days')}</span
								>
								<input
									class="input mt-auto w-full"
									type="number"
									min="0"
									max="365"
									bind:value={owedDebtOverdueStartAfterDays}
								/>
							</label>
							<label class="flex flex-col gap-1.5 text-sm">
								<span class="leading-snug" style:color="var(--text-muted)"
									>{$_('settings.notifications.triggers.owed_debt_overdue_limit_days')}</span
								>
								<input
									class="input mt-auto w-full"
									type="number"
									min="0"
									max="365"
									bind:value={owedDebtOverdueDaysLimit}
								/>
							</label>
						</div>
					</section>

					<div class="grid gap-6 border-t pt-6 sm:grid-cols-2" style:border-color="var(--border)">
						<section class="space-y-3">
							<h3 class="text-sm font-semibold">
								{$_('settings.notifications.triggers.credit_policy_title')}
							</h3>
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug" style:color="var(--text-muted)"
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

						<section class="space-y-3">
							<h3 class="text-sm font-semibold">
								{$_('settings.notifications.triggers.subscription_policy_title')}
							</h3>
							<label class="block space-y-1.5 text-sm">
								<span class="block leading-snug" style:color="var(--text-muted)"
									>{$_('settings.notifications.triggers.subscription_days')}</span
								>
								<input
									class="input w-full"
									type="number"
									min="0"
									max="30"
									bind:value={subscriptionDaysBefore}
								/>
							</label>
						</section>
					</div>

					<section class="space-y-3 border-t pt-6" style:border-color="var(--border)">
						<h3 class="text-sm font-semibold">
							{$_('settings.notifications.triggers.schedule_title')}
						</h3>
						<label class="block max-w-xs space-y-1.5 text-sm">
							<span class="block leading-snug" style:color="var(--text-muted)"
								>{$_('settings.notifications.triggers.send_time')}</span
							>
							<input class="input w-full" type="time" bind:value={notificationTimeLocal} />
						</label>
						<p class="max-w-xl text-xs leading-relaxed" style:color="var(--text-muted)">
							{$_('settings.notifications.triggers.send_time_hint', { values: { timezone } })}
						</p>
					</section>

					<div
						class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
						style:border-color="var(--border)"
					>
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
			</SettingsPanel>
		</section>

		<section class="space-y-3">
			{@render groupHeading(
				$_('settings.notifications.group.messages'),
				$_('settings.notifications.group.messages_hint')
			)}

			<div data-testid="notifications-global-keyboards">
				<SettingsPanel
					label={$_('settings.notifications.keyboard.global_title')}
					description={$_('settings.notifications.keyboard.global_short')}
				>
					<div class="space-y-4">
						<p class="text-sm" style:color="var(--text-muted)">
							{$_('settings.notifications.keyboard.global_hint')}
						</p>
						{#if telegramEnabled && maxEnabled}
							<PageTabs
								active={globalKeyboardTabFor()}
								tabs={[
									{
										id: 'telegram',
										label: $_('settings.notifications.channel.telegram')
									},
									{ id: 'max', label: $_('settings.notifications.channel.max') }
								]}
								onchange={(next) => (globalKeyboardTab = next as KeyboardProvider)}
							/>
						{/if}
						{#if globalKeyboardTabFor() === 'telegram' && telegramEnabled}
							<KeyboardBuilder
								provider="telegram"
								bind:keyboard={globalKeyboardTelegram}
								scope="global-tg"
								triggerType={null}
							/>
						{:else if globalKeyboardTabFor() === 'max' && maxEnabled}
							<KeyboardBuilder
								provider="max"
								bind:keyboard={globalKeyboardMax}
								scope="global-max"
								triggerType={null}
							/>
						{/if}
						<div
							class="flex flex-wrap items-center justify-end gap-2 border-t pt-4"
							style:border-color="var(--border)"
						>
							<button
								type="button"
								class="btn-primary"
								onclick={saveGlobalKeyboards}
								disabled={loading}
								data-testid="notifications-global-keyboards-save"
							>
								{$_('settings.notifications.block_save')}
							</button>
						</div>
					</div>
				</SettingsPanel>
			</div>

			{#each orderedTemplates(templates) as tpl (tpl.trigger_type)}
				{@const editable = templateEditable(tpl.trigger_type)}
				{@const tplKb = templateKeyboards[tpl.trigger_type] ?? {
					telegram: normalizeInlineKeyboard(tpl.keyboard_telegram),
					max: normalizeInlineKeyboard(tpl.keyboard_max)
				}}
				{@const tplKbTab = templateKeyboardTabFor(tpl.trigger_type)}
				<div data-testid={`notifications-template-${tpl.trigger_type}`}>
					<SettingsPanel
						label={triggerLabel(tpl.trigger_type)}
						badge={!editable
							? $_('settings.notifications.templates.state.disabled')
							: tpl.is_custom
								? $_('settings.notifications.templates.state.custom')
								: ''}
						badgeTone={!editable ? 'muted' : 'success'}
					>
						<div class="space-y-5">
							{#if !editable}
								<p
									class="rounded-xl border p-3 text-sm"
									style:border-color="color-mix(in srgb, var(--warning) 35%, var(--border))"
									style:color="var(--text-muted)"
								>
									{templateDisabledHint(tpl.trigger_type)}
								</p>
							{/if}
							<div
								class="space-y-5"
								class:opacity-50={!editable}
								class:pointer-events-none={!editable}
								class:select-none={!editable}
							>
								<div class="space-y-2">
									<label
										for={templateTextareaId(tpl.trigger_type)}
										class="block text-sm font-semibold"
									>
										{$_('settings.notifications.templates.text_label')}
									</label>
									<textarea
										id={templateTextareaId(tpl.trigger_type)}
										class="input min-h-[96px] py-3"
										disabled={!editable}
										readonly={!editable}
										value={templateValue(tpl.trigger_type, tpl.template)}
										oninput={(e) =>
											updateTemplate(
												tpl.trigger_type,
												(e.currentTarget as HTMLTextAreaElement).value
											)}></textarea>
									<p class="text-xs" style:color="var(--text-muted)">
										{$_('settings.notifications.templates.placeholders_hint')}
									</p>
									<div class="flex flex-wrap gap-1.5">
										{#each tpl.placeholders as placeholder (placeholder)}
											<button
												type="button"
												class="btn-ghost px-2 py-1 font-mono text-xs"
												disabled={!editable}
												onclick={() => insertPlaceholder(tpl.trigger_type, placeholder)}
											>
												{`{${placeholder}}`}
											</button>
										{/each}
									</div>
								</div>

								{#if telegramEnabled || maxEnabled}
									<div class="space-y-3 border-t pt-5" style:border-color="var(--border)">
										<div class="space-y-1">
											<p class="text-sm font-semibold">
												{$_('settings.notifications.keyboard.template_title')}
											</p>
											<p class="text-xs" style:color="var(--text-muted)">
												{$_('settings.notifications.keyboard.template_global_hint')}
											</p>
										</div>
										{#if telegramEnabled && maxEnabled}
											<PageTabs
												active={tplKbTab}
												tabs={[
													{
														id: 'telegram',
														label: $_('settings.notifications.channel.telegram')
													},
													{ id: 'max', label: $_('settings.notifications.channel.max') }
												]}
												onchange={(next) =>
													setTemplateKeyboardTab(tpl.trigger_type, next as KeyboardProvider)}
											/>
										{/if}
										{#if tplKbTab === 'telegram' && telegramEnabled}
											<KeyboardBuilder
												provider="telegram"
												keyboard={tplKb.telegram}
												includeGlobal={globalKeyboardTelegram}
												disabled={!editable}
												scope={`tpl-${tpl.trigger_type}-tg`}
												triggerType={tpl.trigger_type}
												onchange={(kb) => setTemplateKeyboard(tpl.trigger_type, 'telegram', kb)}
											/>
										{:else if maxEnabled}
											<KeyboardBuilder
												provider="max"
												keyboard={tplKb.max}
												includeGlobal={globalKeyboardMax}
												disabled={!editable}
												scope={`tpl-${tpl.trigger_type}-max`}
												triggerType={tpl.trigger_type}
												onchange={(kb) => setTemplateKeyboard(tpl.trigger_type, 'max', kb)}
											/>
										{/if}
									</div>
								{/if}

								{#if previewText[tpl.trigger_type]}
									<div
										class="space-y-2 rounded-xl border p-3 text-sm"
										style:border-color="var(--border)"
										style:background-color="color-mix(in srgb, var(--border) 20%, transparent)"
									>
										<p class="text-xs font-medium" style:color="var(--text-muted)">
											{$_('settings.notifications.templates.preview_result')}
										</p>
										<p class="whitespace-pre-wrap">{previewText[tpl.trigger_type]}</p>
										{#if previewKeyboards[tpl.trigger_type]}
											<div class="grid gap-2 sm:grid-cols-2">
												<div class="min-w-0">
													<p class="text-xs font-medium" style:color="var(--text-muted)">
														{$_('settings.notifications.keyboard.preview_telegram')}
													</p>
													<pre
														class="mt-1 overflow-x-auto rounded border p-2 text-xs"
														style:border-color="var(--border)">{JSON.stringify(
															previewKeyboards[tpl.trigger_type].telegram,
															null,
															2
														)}</pre>
												</div>
												<div class="min-w-0">
													<p class="text-xs font-medium" style:color="var(--text-muted)">
														{$_('settings.notifications.keyboard.preview_max')}
													</p>
													<pre
														class="mt-1 overflow-x-auto rounded border p-2 text-xs"
														style:border-color="var(--border)">{JSON.stringify(
															previewKeyboards[tpl.trigger_type].max,
															null,
															2
														)}</pre>
												</div>
											</div>
										{/if}
									</div>
								{/if}

								<div
									class="flex flex-wrap items-center gap-2 border-t pt-4"
									style:border-color="var(--border)"
								>
									<button
										type="button"
										class="btn-ghost sm:mr-auto"
										disabled={!editable}
										onclick={() => resetTemplate(tpl.trigger_type)}
									>
										{$_('settings.notifications.templates.reset')}
									</button>
									<button
										type="button"
										class="btn-ghost"
										disabled={!editable}
										onclick={() =>
											previewTemplate(
												tpl.trigger_type,
												templateValue(tpl.trigger_type, tpl.template)
											)}
									>
										{$_('settings.notifications.templates.preview')}
									</button>
									<button
										type="button"
										class="btn-primary"
										disabled={loading || !editable}
										onclick={() =>
											saveTemplate(tpl.trigger_type, templateValue(tpl.trigger_type, tpl.template))}
									>
										{$_('settings.notifications.block_save')}
									</button>
								</div>
							</div>
						</div>
					</SettingsPanel>
				</div>
			{/each}
		</section>
	</div>
</PageLoadGate>
