<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { get } from 'svelte/store';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import {
		ApiError,
		changePassword,
		createToken,
		deleteToken,
		getNotificationSettings,
		getRegistrationEnabled,
		getUserSettings,
		listTokens,
		previewNotificationTemplate,
		putUserSettings,
		putNotificationSettings,
		resetNotificationTemplates,
		sendNotificationTest,
		type APIToken,
		type APITokenCreated,
		type NotificationSettings,
		type NotificationTemplate
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { applyTheme } from '$lib/theme';
	import { setLocale } from '$lib/i18n';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import DateTimePicker from '$lib/components/DateTimePicker.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import Select from '$lib/components/Select.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import IntegerInput from '$lib/components/IntegerInput.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { confirm } from '$lib/confirm';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import { formatApiError } from '$lib/api/errors';
	import {
		apiDateTimeToRFC3339,
		defaultTokenExpiryLocal,
		formatAPIOperationDateTimeForDisplay,
		fromDateLocalEnd
	} from '$lib/dates';
	import { dateOnlyPicker } from '$lib/datetime-picker-standards';
	import { toast } from '$lib/toast';
	import CategoriesTab from '$lib/settings/CategoriesTab.svelte';
	import ImportTab from '$lib/settings/ImportTab.svelte';
	import AdminSystemTab from '../admin/+page.svelte';
	import AdminUsersTab from '../admin/users/+page.svelte';
	import AdminBackupsTab from '../admin/backups/+page.svelte';
	import AdminDiagnosticsTab from '../admin/diagnostics/+page.svelte';
	import AdminSupportLinks from '$lib/components/AdminSupportLinks.svelte';
	import BackLink, { type BreadcrumbItem } from '$lib/components/BackLink.svelte';

	type Tab =
		| 'profile'
		| 'password'
		| 'tokens'
		| 'notifications'
		| 'categories'
		| 'import'
		| 'admin';
	type AdminTab = 'system' | 'users' | 'backups' | 'diagnostics';

	function tabFromSearchParams(params: URLSearchParams): Tab {
		const value = params.get('tab');
		if (
			value === 'password' ||
			value === 'tokens' ||
			value === 'notifications' ||
			value === 'categories' ||
			value === 'import' ||
			value === 'admin'
		) {
			return value;
		}
		return 'profile';
	}

	function adminTabFromSearchParams(params: URLSearchParams): AdminTab {
		const value = params.get('admin_tab');
		if (value === 'users' || value === 'backups' || value === 'diagnostics') {
			return value;
		}
		return 'system';
	}

	let tab = $state<Tab>(tabFromSearchParams(get(page).url.searchParams));
	let adminTab = $state<AdminTab>(adminTabFromSearchParams(get(page).url.searchParams));
	let profileFeedback = $state({ error: '', success: '' });
	let passwordFeedback = $state({ error: '', success: '' });
	let loading = $state(false);

	let displayName = $state('');
	let language = $state('ru');
	let currency = $state('RUB');
	let timezone = $state('Europe/Moscow');
	let theme = $state<'light' | 'dark'>('light');

	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');

	let tokens = $state<APIToken[]>([]);
	let newTokenName = $state('');
	let newTokenExpiresAt = $state('');
	let newTokenNeverExpires = $state(false);
	let tokenFormError = $state('');
	let createdToken = $state<APITokenCreated | null>(null);

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
	let triggerNegativeBalance = $state(true);
	let triggerBudget = $state(true);
	let triggerPasswordReset = $state(true);
	let triggerUserRegistration = $state(true);
	let registrationEnabled = $state(false);
	let debtDaysBefore = $state(1);
	let myDebtOverdueDaysLimit = $state(7);
	let owedDebtOverdueStartAfterDays = $state(0);
	let owedDebtOverdueDaysLimit = $state(7);
	let creditDaysBefore = $state(1);
	let notificationTimeLocal = $state('00:00');
	let templates = $state<NotificationTemplate[]>([]);
	let templatesDirty = $state<Record<string, string>>({});
	let previewText = $state<Record<string, string>>({});
	let templateFeedback = $state<Record<string, { error: string; success: string }>>({});
	let channelFeedback = $state<Record<'telegram' | 'max', { error: string; success: string }>>({
		telegram: { error: '', success: '' },
		max: { error: '', success: '' }
	});
	let blockFeedback = $state<
		Record<
			'telegram' | 'max' | 'triggerTypes' | 'triggerPolicy',
			{ error: string; success: string }
		>
	>({
		telegram: { error: '', success: '' },
		max: { error: '', success: '' },
		triggerTypes: { error: '', success: '' },
		triggerPolicy: { error: '', success: '' }
	});
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
		'balance_shortfall',
		'budget_threshold',
		'password_reset',
		'user_registration',
		'test'
	];

	type NotificationTriggerKey =
		| 'debt'
		| 'credit'
		| 'planned'
		| 'negativeBalance'
		| 'budget'
		| 'passwordReset'
		| 'userRegistration';

	const templateSettingKey: Partial<Record<string, NotificationTriggerKey>> = {
		debt_overdue: 'debt',
		debt_due_soon: 'debt',
		credit_payment: 'credit',
		planned_operation: 'planned',
		balance_shortfall: 'negativeBalance',
		budget_threshold: 'budget',
		password_reset: 'passwordReset',
		user_registration: 'userRegistration'
	};

	function isNotificationTriggerEnabled(key: NotificationTriggerKey) {
		switch (key) {
			case 'debt':
				return triggerDebt;
			case 'credit':
				return triggerCredit;
			case 'planned':
				return triggerPlanned;
			case 'negativeBalance':
				return triggerNegativeBalance;
			case 'budget':
				return triggerBudget;
			case 'passwordReset':
				return triggerPasswordReset;
			case 'userRegistration':
				return triggerUserRegistration;
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
			case 'planned':
				triggerPlanned = !triggerPlanned;
				break;
			case 'negativeBalance':
				triggerNegativeBalance = !triggerNegativeBalance;
				break;
			case 'budget':
				triggerBudget = !triggerBudget;
				break;
			case 'passwordReset':
				triggerPasswordReset = !triggerPasswordReset;
				break;
			case 'userRegistration':
				triggerUserRegistration = !triggerUserRegistration;
				break;
		}
	}

	function notificationTriggerRows(): Array<{ key: NotificationTriggerKey; hintKey: string }> {
		const rows: Array<{ key: NotificationTriggerKey; hintKey: string }> = [
			{ key: 'debt', hintKey: 'debt_hint' },
			{ key: 'credit', hintKey: 'credit_hint' },
			{ key: 'planned', hintKey: 'planned_hint' },
			{ key: 'negativeBalance', hintKey: 'negativeBalance_hint' },
			{ key: 'budget', hintKey: 'budget_hint' }
		];
		if ($user?.is_admin) {
			rows.push({ key: 'passwordReset', hintKey: 'passwordReset_hint' });
			if (registrationEnabled) {
				rows.push({ key: 'userRegistration', hintKey: 'userRegistration_hint' });
			}
		}
		return rows;
	}

	type NotificationPolicyFieldKey =
		| 'debtDaysBefore'
		| 'myDebtOverdueDaysLimit'
		| 'owedDebtOverdueStartAfterDays'
		| 'owedDebtOverdueDaysLimit'
		| 'creditDaysBefore';

	function notificationPolicyRows(): Array<{
		key: NotificationPolicyFieldKey;
		triggerKey: NotificationTriggerKey;
		nameKey: string;
		hintKey: string;
		min: number;
		max: number;
	}> {
		return [
			{
				key: 'debtDaysBefore',
				triggerKey: 'debt',
				nameKey: 'myDebtBefore',
				hintKey: 'myDebtBefore_hint',
				min: 0,
				max: 30
			},
			{
				key: 'myDebtOverdueDaysLimit',
				triggerKey: 'debt',
				nameKey: 'myDebtOverdue',
				hintKey: 'myDebtOverdue_hint',
				min: 0,
				max: 365
			},
			{
				key: 'owedDebtOverdueStartAfterDays',
				triggerKey: 'debt',
				nameKey: 'owedDebtStart',
				hintKey: 'owedDebtStart_hint',
				min: 0,
				max: 365
			},
			{
				key: 'owedDebtOverdueDaysLimit',
				triggerKey: 'debt',
				nameKey: 'owedDebtOverdue',
				hintKey: 'owedDebtOverdue_hint',
				min: 0,
				max: 365
			},
			{
				key: 'creditDaysBefore',
				triggerKey: 'credit',
				nameKey: 'creditDays',
				hintKey: 'creditDays_hint',
				min: 0,
				max: 30
			}
		];
	}

	function policyFieldEditable(triggerKey: NotificationTriggerKey) {
		return isNotificationTriggerEnabled(triggerKey);
	}

	function policyScheduleEditable() {
		return triggerDebt || triggerCredit || triggerPlanned;
	}

	function policyDisabledHint(triggerKey: NotificationTriggerKey) {
		return $_('settings.notifications.templates.disabled_setting', {
			values: { setting: $_(`settings.notifications.triggers.${triggerKey}`) }
		});
	}

	function policyScheduleDisabledHint() {
		return $_('settings.notifications.policy.disabled_schedule');
	}

	function policyFieldValue(key: NotificationPolicyFieldKey): number {
		switch (key) {
			case 'debtDaysBefore':
				return debtDaysBefore;
			case 'myDebtOverdueDaysLimit':
				return myDebtOverdueDaysLimit;
			case 'owedDebtOverdueStartAfterDays':
				return owedDebtOverdueStartAfterDays;
			case 'owedDebtOverdueDaysLimit':
				return owedDebtOverdueDaysLimit;
			case 'creditDaysBefore':
				return creditDaysBefore;
		}
	}

	function setPolicyFieldValue(key: NotificationPolicyFieldKey, value: number) {
		switch (key) {
			case 'debtDaysBefore':
				debtDaysBefore = value;
				break;
			case 'myDebtOverdueDaysLimit':
				myDebtOverdueDaysLimit = value;
				break;
			case 'owedDebtOverdueStartAfterDays':
				owedDebtOverdueStartAfterDays = value;
				break;
			case 'owedDebtOverdueDaysLimit':
				owedDebtOverdueDaysLimit = value;
				break;
			case 'creditDaysBefore':
				creditDaysBefore = value;
				break;
		}
	}

	onMount(() => {
		tab = tabFromSearchParams(new URL(window.location.href).searchParams);
		adminTab = adminTabFromSearchParams(new URL(window.location.href).searchParams);
		const syncTabFromLocation = () => {
			const params = new URL(window.location.href).searchParams;
			if (params.get('tab') === 'accounts') {
				void goto(resolve('/accounts'), { replaceState: true });
				return;
			}
			tab = tabFromSearchParams(params);
			adminTab = adminTabFromSearchParams(params);
		};
		window.addEventListener('popstate', syncTabFromLocation);
		void (async () => {
			try {
				await loadProfile();
				await loadTokens();
			} catch (err) {
				profileFeedback = {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				};
			}
		})();
		return () => window.removeEventListener('popstate', syncTabFromLocation);
	});

	$effect(() => {
		if (tab !== 'tokens') return;
		if (!newTokenExpiresAt) {
			newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
		}
	});

	$effect(() => {
		if (tab !== 'notifications') return;
		void loadNotifications().catch((err) => {
			toast(err instanceof ApiError ? err.message : $_('common.error'), 'error');
		});
	});

	function selectTab(next: Tab) {
		if (next === tab) return;
		tab = next;
		const url = new URL(get(page).url);
		if (next === 'profile') {
			url.searchParams.delete('tab');
		} else {
			url.searchParams.set('tab', next);
		}
		if (next !== 'admin') {
			url.searchParams.delete('admin_tab');
		} else {
			url.searchParams.set('admin_tab', adminTab);
		}
		const search = url.searchParams.toString();
		const settingsUrl = search ? `${resolve('/settings')}?${search}` : resolve('/settings');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		void goto(settingsUrl, { replaceState: true, keepFocus: true, noScroll: true });
	}

	function selectAdminTab(next: AdminTab) {
		if (next === adminTab) return;
		adminTab = next;
		const url = new URL(get(page).url);
		url.searchParams.set('tab', 'admin');
		url.searchParams.set('admin_tab', next);
		const search = url.searchParams.toString();
		const settingsUrl = search ? `${resolve('/settings')}?${search}` : resolve('/settings');
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- query params after resolved base path
		void goto(settingsUrl, { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function loadProfile() {
		const s = await getUserSettings();
		displayName = s.display_name;
		language = s.language;
		currency = s.currency;
		timezone = s.timezone;
		theme = s.theme === 'dark' ? 'dark' : 'light';
	}

	async function loadTokens() {
		tokens = await listTokens();
	}

	function applyNotificationSettings(data: NotificationSettings) {
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
		triggerNegativeBalance =
			'trigger_negative_balance' in data ? data.trigger_negative_balance : true;
		triggerBudget = 'trigger_budget' in data ? data.trigger_budget : true;
		triggerPasswordReset =
			'trigger_password_reset' in data ? (data.trigger_password_reset ?? true) : true;
		triggerUserRegistration =
			'trigger_user_registration' in data ? (data.trigger_user_registration ?? true) : true;
		debtDaysBefore = data.debt_days_before;
		myDebtOverdueDaysLimit = data.my_debt_overdue_days_limit ?? 7;
		owedDebtOverdueStartAfterDays = data.owed_debt_overdue_start_after_days ?? 0;
		owedDebtOverdueDaysLimit = data.owed_debt_overdue_days_limit ?? 7;
		creditDaysBefore = data.credit_days_before;
		notificationTimeLocal = data.notification_time_local ?? '00:00';
		templates = data.templates;
		templatesDirty = {};
		previewText = {};
		templateFeedback = {};
		persistedNotificationState = {
			telegramEnabled: data.telegram_enabled,
			telegramChatId: data.telegram_chat_id ?? '',
			maxEnabled: data.max_enabled,
			maxProvider: data.max_provider === 'official' ? 'official' : 'a161',
			maxUserId: data.max_user_id ? String(data.max_user_id) : '',
			maxRecipientId: data.max_recipient_id ? String(data.max_recipient_id) : ''
		};
	}

	async function loadNotifications() {
		const [data, regEnabled] = await Promise.all([
			getNotificationSettings(),
			getRegistrationEnabled()
		]);
		registrationEnabled = regEnabled;
		applyNotificationSettings(data);
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

	function visibleTemplates(list: NotificationTemplate[]) {
		return orderedTemplates(list).filter(
			(tpl) => tpl.trigger_type !== 'user_registration' || registrationEnabled
		);
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
		channelFeedback = {
			...channelFeedback,
			[channel]: { error: '', success: '' }
		};
		if (channelHasUnsavedChanges(channel)) {
			channelFeedback = {
				...channelFeedback,
				[channel]: { error: $_('settings.notifications.error.save_before_test'), success: '' }
			};
			return;
		}
		loading = true;
		try {
			await sendNotificationTest(channel);
			channelFeedback = {
				...channelFeedback,
				[channel]: {
					error: '',
					success: $_('settings.notifications.success.test_sent', { values: { channel } })
				}
			};
		} catch (err) {
			channelFeedback = {
				...channelFeedback,
				[channel]: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		} finally {
			loading = false;
		}
	}

	async function saveTelegramBlock() {
		if (!notificationSecretConfigured) return;
		blockFeedback = { ...blockFeedback, telegram: { error: '', success: '' } };
		loading = true;
		try {
			const data = await putNotificationSettings({
				telegram_enabled: telegramEnabled,
				telegram_bot_token: telegramBotToken.trim() || undefined,
				telegram_chat_id: telegramChatId.trim() || undefined
			});
			telegramBotToken = '';
			applyNotificationSettings(data);
			blockFeedback = {
				...blockFeedback,
				telegram: { error: '', success: $_('settings.notifications.success.block_saved') }
			};
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			blockFeedback = {
				...blockFeedback,
				telegram: { error: err instanceof ApiError ? err.message : $_('common.error'), success: '' }
			};
		} finally {
			loading = false;
		}
	}

	async function saveMaxBlock() {
		if (!notificationSecretConfigured) return;
		blockFeedback = { ...blockFeedback, max: { error: '', success: '' } };
		loading = true;
		try {
			const data = await putNotificationSettings({
				max_enabled: maxEnabled,
				max_provider: maxProvider,
				max_token: maxToken.trim() || undefined,
				max_user_id: maxUserId.trim() ? Number(maxUserId) : null,
				max_recipient_id: maxRecipientId.trim() ? Number(maxRecipientId) : null
			});
			maxToken = '';
			applyNotificationSettings(data);
			blockFeedback = {
				...blockFeedback,
				max: { error: '', success: $_('settings.notifications.success.block_saved') }
			};
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			blockFeedback = {
				...blockFeedback,
				max: { error: err instanceof ApiError ? err.message : $_('common.error'), success: '' }
			};
		} finally {
			loading = false;
		}
	}

	async function saveTriggerTypesBlock() {
		if (!notificationSecretConfigured) return;
		blockFeedback = { ...blockFeedback, triggerTypes: { error: '', success: '' } };
		loading = true;
		try {
			const data = await putNotificationSettings({
				trigger_debt: triggerDebt,
				trigger_credit: triggerCredit,
				trigger_planned: triggerPlanned,
				trigger_negative_balance: triggerNegativeBalance,
				trigger_budget: triggerBudget,
				trigger_password_reset: $user?.is_admin ? triggerPasswordReset : undefined,
				trigger_user_registration:
					$user?.is_admin && registrationEnabled ? triggerUserRegistration : undefined
			});
			applyNotificationSettings(data);
			blockFeedback = {
				...blockFeedback,
				triggerTypes: { error: '', success: $_('settings.notifications.success.block_saved') }
			};
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			blockFeedback = {
				...blockFeedback,
				triggerTypes: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		} finally {
			loading = false;
		}
	}

	async function saveTriggerPolicyBlock() {
		if (!notificationSecretConfigured) return;
		blockFeedback = { ...blockFeedback, triggerPolicy: { error: '', success: '' } };
		if (
			!validateDaysRange(debtDaysBefore) ||
			!validateDaysRange(creditDaysBefore) ||
			!validateOverdueDaysLimit(myDebtOverdueDaysLimit) ||
			!validateOverdueDaysLimit(owedDebtOverdueStartAfterDays) ||
			!validateOverdueDaysLimit(owedDebtOverdueDaysLimit)
		) {
			blockFeedback = {
				...blockFeedback,
				triggerPolicy: { error: $_('settings.notifications.error.policy_range'), success: '' }
			};
			return;
		}
		if (!validateLocalTime(notificationTimeLocal)) {
			blockFeedback = {
				...blockFeedback,
				triggerPolicy: { error: $_('settings.notifications.error.time_format'), success: '' }
			};
			return;
		}
		loading = true;
		try {
			const data = await putNotificationSettings({
				debt_days_before: debtDaysBefore,
				my_debt_overdue_days_limit: myDebtOverdueDaysLimit,
				owed_debt_overdue_start_after_days: owedDebtOverdueStartAfterDays,
				owed_debt_overdue_days_limit: owedDebtOverdueDaysLimit,
				credit_days_before: creditDaysBefore,
				notification_time_local: notificationTimeLocal.trim()
			});
			applyNotificationSettings(data);
			blockFeedback = {
				...blockFeedback,
				triggerPolicy: { error: '', success: $_('settings.notifications.success.block_saved') }
			};
			toast($_('settings.notifications.success.block_saved'));
		} catch (err) {
			blockFeedback = {
				...blockFeedback,
				triggerPolicy: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		} finally {
			loading = false;
		}
	}

	async function saveTemplate(triggerType: string, template: string) {
		if (!notificationSecretConfigured) return;
		if (!templateEditable(triggerType)) return;
		templateFeedback = {
			...templateFeedback,
			[triggerType]: { error: '', success: '' }
		};
		const templateError = validateTemplateText(template);
		if (templateError) {
			templateFeedback = {
				...templateFeedback,
				[triggerType]: { error: templateError, success: '' }
			};
			return;
		}
		loading = true;
		try {
			const data = await putNotificationSettings({
				templates: [{ trigger_type: triggerType, template }]
			});
			applyNotificationSettings(data);
			const success = $_('settings.notifications.success.template_saved', {
				values: { trigger: triggerLabel(triggerType) }
			});
			templateFeedback = {
				...templateFeedback,
				[triggerType]: { error: '', success }
			};
			toast(success);
		} catch (err) {
			templateFeedback = {
				...templateFeedback,
				[triggerType]: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		} finally {
			loading = false;
		}
	}

	async function previewTemplate(triggerType: string, template: string) {
		if (!notificationSecretConfigured) return;
		if (!templateEditable(triggerType)) return;
		templateFeedback = {
			...templateFeedback,
			[triggerType]: { error: '', success: '' }
		};
		const templateError = validateTemplateText(template);
		if (templateError) {
			templateFeedback = {
				...templateFeedback,
				[triggerType]: { error: templateError, success: '' }
			};
			return;
		}
		try {
			const result = await previewNotificationTemplate({ trigger_type: triggerType, template });
			previewText = { ...previewText, [triggerType]: result.text };
		} catch (err) {
			templateFeedback = {
				...templateFeedback,
				[triggerType]: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		}
	}

	async function resetTemplate(triggerType: string) {
		if (!notificationSecretConfigured) return;
		if (!templateEditable(triggerType)) return;
		templateFeedback = {
			...templateFeedback,
			[triggerType]: { error: '', success: '' }
		};
		try {
			const data = await resetNotificationTemplates(triggerType);
			applyNotificationSettings(data);
			templateFeedback = {
				...templateFeedback,
				[triggerType]: {
					error: '',
					success: $_('settings.notifications.success.template_reset', {
						values: { trigger: triggerLabel(triggerType) }
					})
				}
			};
		} catch (err) {
			templateFeedback = {
				...templateFeedback,
				[triggerType]: {
					error: err instanceof ApiError ? err.message : $_('common.error'),
					success: ''
				}
			};
		}
	}

	async function saveProfile(e: Event) {
		e.preventDefault();
		profileFeedback = { error: '', success: '' };
		loading = true;
		try {
			const updated = await putUserSettings({
				display_name: displayName,
				language,
				currency,
				timezone,
				theme
			});
			localStorage.setItem('theme', updated.theme);
			applyTheme(updated.theme === 'dark' ? 'dark' : 'light');
			setLocale(updated.language);
			user.update((u) => (u ? { ...u, ...updated } : u));
			timezone = updated.timezone;
			profileFeedback = { error: '', success: $_('common.saved') };
			toast($_('common.saved'));
		} catch (err) {
			profileFeedback = { error: formatApiError(err), success: '' };
		} finally {
			loading = false;
		}
	}

	async function savePassword(e: Event) {
		e.preventDefault();
		passwordFeedback = { error: '', success: '' };
		if (newPassword !== confirmPassword) {
			passwordFeedback = { error: $_('errors.PASSWORDS_MISMATCH'), success: '' };
			return;
		}
		if (!validatePasswordPolicy(newPassword, $user?.login ?? '')) {
			passwordFeedback = { error: $_('auth.password.requirements'), success: '' };
			return;
		}
		if (oldPassword === newPassword) {
			passwordFeedback = { error: $_('errors.PASSWORD_UNCHANGED'), success: '' };
			return;
		}
		loading = true;
		try {
			await changePassword(oldPassword, newPassword, confirmPassword);
			oldPassword = '';
			newPassword = '';
			confirmPassword = '';
			passwordFeedback = { error: '', success: $_('settings.password.changed') };
			toast($_('settings.password.changed'));
		} catch (err) {
			passwordFeedback = { error: formatApiError(err), success: '' };
		} finally {
			loading = false;
		}
	}

	async function handleCreateToken(e: Event) {
		e.preventDefault();
		tokenFormError = '';
		loading = true;
		try {
			const opts = newTokenNeverExpires
				? { neverExpires: true }
				: {
						expiresAt: apiDateTimeToRFC3339(fromDateLocalEnd(newTokenExpiresAt, timezone))
					};
			createdToken = await createToken(newTokenName.trim(), opts);
			newTokenName = '';
			newTokenNeverExpires = false;
			newTokenExpiresAt = defaultTokenExpiryLocal(timezone);
			toast($_('common.saved'));
			await loadTokens();
		} catch (err) {
			tokenFormError = formatApiError(err);
		} finally {
			loading = false;
		}
	}

	async function revokeToken(id: string) {
		const ok = await confirm({
			message: $_('settings.tokens.confirm.revoke'),
			confirmLabel: $_('common.delete'),
			danger: true
		});
		if (!ok) return;
		await deleteToken(id);
		toast($_('common.deleted'));
		await loadTokens();
	}

	function copyToken(token: string) {
		navigator.clipboard.writeText(token);
	}

	function closeTokenModal() {
		createdToken = null;
	}

	const settingsTabs = $derived.by(() => {
		void $locale;
		const tabs = [
			{ id: 'profile', label: tr('settings.tab.profile') },
			{ id: 'password', label: tr('settings.tab.password') },
			{ id: 'tokens', label: tr('settings.tab.tokens') },
			{ id: 'notifications', label: tr('settings.tab.notifications') },
			{ id: 'categories', label: tr('settings.tab.categories') },
			{ id: 'import', label: tr('settings.tab.import') }
		];
		if ($user?.is_admin) {
			tabs.push({ id: 'admin', label: tr('admin.title') });
		}
		return tabs;
	});

	const adminTabs = $derived.by(() => {
		void $locale;
		return [
			{ id: 'system', label: tr('admin.tab.system') },
			{ id: 'users', label: tr('admin.tab.users') },
			{ id: 'backups', label: tr('admin.tab.backups') },
			{ id: 'diagnostics', label: tr('admin.tab.diagnostics') }
		];
	});

	function formatOptional(value: string | null) {
		return value && value.trim() !== '' ? value : '—';
	}

	function formatTokenExpiry(value: string | null) {
		if (!value || value.trim() === '') {
			return $_('settings.tokens.col.never');
		}
		return formatAPIOperationDateTimeForDisplay(value, timezone);
	}

	const breadcrumbItems = $derived.by((): BreadcrumbItem[] => {
		void $locale;
		const home: BreadcrumbItem = { href: '/', label: tr('nav.home') };
		const settings: BreadcrumbItem = { href: '/settings', label: tr('settings.title') };

		if (tab === 'profile') {
			return [home, settings];
		}

		if (tab === 'admin') {
			const admin: BreadcrumbItem = {
				href: '/settings',
				label: tr('admin.title'),
				search: 'tab=admin'
			};
			if (adminTab === 'system') {
				return [home, settings, admin];
			}
			const adminTabLabels: Record<Exclude<AdminTab, 'system'>, string> = {
				users: tr('admin.tab.users'),
				backups: tr('admin.tab.backups'),
				diagnostics: tr('admin.tab.diagnostics')
			};
			return [
				home,
				settings,
				admin,
				{
					href: '/settings',
					label: adminTabLabels[adminTab],
					search: `tab=admin&admin_tab=${adminTab}`
				}
			];
		}

		const tabLabels: Record<Exclude<Tab, 'profile' | 'admin'>, string> = {
			password: tr('settings.tab.password'),
			tokens: tr('settings.tab.tokens'),
			notifications: tr('settings.tab.notifications'),
			categories: tr('settings.tab.categories'),
			import: tr('settings.tab.import')
		};

		return [home, settings, { href: '/settings', label: tabLabels[tab], search: `tab=${tab}` }];
	});
</script>

<svelte:head>
	<title>{$_('settings.title')} — {$_('app.title')}</title>
</svelte:head>

<BackLink items={breadcrumbItems} />

<h1 class="mb-6 text-2xl font-semibold">{$_('settings.title')}</h1>

<div class="mb-6">
	<PageTabs active={tab} tabs={settingsTabs} onchange={(next) => selectTab(next as Tab)} />
</div>

{#if tab === 'profile'}
	<form class="card max-w-lg space-y-4" onsubmit={saveProfile}>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="login">{$_('settings.login')}</label>
			<input
				id="login"
				class="input cursor-not-allowed opacity-80"
				type="text"
				value={$user?.login ?? ''}
				readonly
				tabindex="-1"
			/>
			<p class="mt-1 text-xs" style:color="var(--text-muted)">{$_('settings.login.readonly')}</p>
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="display"
				>{$_('register.display_name')}</label
			>
			<input id="display" class="input" bind:value={displayName} />
		</div>
		<Select
			id="lang"
			label={$_('settings.language')}
			bind:value={language}
			options={[
				{ value: 'ru', label: 'Русский' },
				{ value: 'en', label: 'English' }
			]}
		/>
		<Select
			id="currency"
			label={$_('settings.currency')}
			bind:value={currency}
			options={[
				{ value: 'RUB', label: 'RUB' },
				{ value: 'USD', label: 'USD' },
				{ value: 'EUR', label: 'EUR' }
			]}
		/>
		<TimezonePicker
			id="tz"
			label={$_('settings.timezone')}
			hint={$_('settings.timezone.hint')}
			bind:value={timezone}
		/>
		<Select
			id="theme"
			label={$_('settings.theme')}
			bind:value={theme}
			options={[
				{ value: 'light', label: $_('settings.theme.light') },
				{ value: 'dark', label: $_('settings.theme.dark') }
			]}
		/>
		<FormFeedback error={profileFeedback.error} success={profileFeedback.success} />
		<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
	</form>
{:else if tab === 'password'}
	<form class="card max-w-lg space-y-4" onsubmit={savePassword}>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="old"
				>{$_('settings.password.current')}</label
			>
			<input id="old" type="password" class="input" bind:value={oldPassword} required />
		</div>
		<div>
			<label class="mb-1.5 block text-sm font-medium" for="new">{$_('settings.password.new')}</label
			>
			<input
				id="new"
				type="password"
				class="input"
				bind:value={newPassword}
				minlength="8"
				required
			/>
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
		<FormFeedback error={passwordFeedback.error} success={passwordFeedback.success} />
		<button type="submit" class="btn-primary" disabled={loading}>{$_('settings.save')}</button>
	</form>
{:else if tab === 'notifications'}
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
					{#if blockFeedback.telegram.error}
						<p class="text-sm" style:color="var(--danger)">{blockFeedback.telegram.error}</p>
					{/if}
					{#if blockFeedback.telegram.success}
						<p class="text-sm" style:color="var(--primary)">{blockFeedback.telegram.success}</p>
					{/if}
					{#if channelFeedback.telegram.error}
						<p class="text-sm" style:color="var(--danger)">{channelFeedback.telegram.error}</p>
					{/if}
					{#if channelFeedback.telegram.success}
						<p class="text-sm" style:color="var(--primary)">{channelFeedback.telegram.success}</p>
					{/if}
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
					{#if blockFeedback.max.error}
						<p class="text-sm" style:color="var(--danger)">{blockFeedback.max.error}</p>
					{/if}
					{#if blockFeedback.max.success}
						<p class="text-sm" style:color="var(--primary)">{blockFeedback.max.success}</p>
					{/if}
					{#if channelFeedback.max.error}
						<p class="text-sm" style:color="var(--danger)">{channelFeedback.max.error}</p>
					{/if}
					{#if channelFeedback.max.success}
						<p class="text-sm" style:color="var(--primary)">{channelFeedback.max.success}</p>
					{/if}
				</div>

				<div class="card space-y-4">
					<h3 class="text-lg font-semibold">{$_('settings.notifications.triggers.title')}</h3>
					<p class="text-sm" style:color="var(--text-muted)">
						{$_('settings.notifications.triggers.types_hint')}
					</p>
					<div class="hidden md:block md:overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead>
								<tr style:color="var(--text-muted)">
									<th class="pb-3 pr-4 font-medium">
										{$_('settings.notifications.triggers.col.name')}
									</th>
									<th class="pb-3 pr-4 font-medium">
										{$_('settings.notifications.triggers.col.description')}
									</th>
									<th class="pb-3 text-right font-medium">
										{$_('settings.notifications.triggers.col.state')}
									</th>
								</tr>
							</thead>
							<tbody>
								{#each notificationTriggerRows() as row (row.key)}
									<tr class="border-t align-top" style:border-color="var(--border)">
										<td class="py-3 pr-4 leading-snug">
											{$_(`settings.notifications.triggers.${row.key}`)}
										</td>
										<td class="py-3 pr-4 leading-relaxed" style:color="var(--text-muted)">
											{$_(`settings.notifications.triggers.${row.hintKey}`)}
										</td>
										<td class="py-3 text-right">
											<div class="flex justify-end">
												<ToggleSwitch
													checked={isNotificationTriggerEnabled(row.key)}
													label={$_(`settings.notifications.triggers.${row.key}`)}
													onchange={() => toggleNotificationTrigger(row.key)}
												/>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<div class="space-y-3 md:hidden">
						{#each notificationTriggerRows() as row (row.key)}
							<article class="rounded-xl border p-4" style:border-color="var(--border)">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0">
										<p class="font-medium leading-snug">
											{$_(`settings.notifications.triggers.${row.key}`)}
										</p>
										<p class="mt-1 text-sm leading-relaxed" style:color="var(--text-muted)">
											{$_(`settings.notifications.triggers.${row.hintKey}`)}
										</p>
									</div>
									<ToggleSwitch
										checked={isNotificationTriggerEnabled(row.key)}
										label={$_(`settings.notifications.triggers.${row.key}`)}
										onchange={() => toggleNotificationTrigger(row.key)}
									/>
								</div>
							</article>
						{/each}
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
					{#if blockFeedback.triggerTypes.error}
						<p class="text-sm" style:color="var(--danger)">{blockFeedback.triggerTypes.error}</p>
					{/if}
					{#if blockFeedback.triggerTypes.success}
						<p class="text-sm" style:color="var(--primary)">
							{blockFeedback.triggerTypes.success}
						</p>
					{/if}
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

					<div class="hidden md:block md:overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead>
								<tr style:color="var(--text-muted)">
									<th class="pb-3 pr-4 font-medium">
										{$_('settings.notifications.triggers.col.name')}
									</th>
									<th class="pb-3 pr-4 font-medium">
										{$_('settings.notifications.triggers.col.description')}
									</th>
									<th class="pb-3 text-right font-medium">
										{$_('settings.notifications.triggers.col.state')}
									</th>
								</tr>
							</thead>
							<tbody>
								{#each notificationPolicyRows() as row (row.key)}
									{@const editable = policyFieldEditable(row.triggerKey)}
									<tr
										class="border-t align-top"
										class:opacity-50={!editable}
										style:border-color="var(--border)"
									>
										<td class="py-3 pr-4 leading-snug">
											{$_(`settings.notifications.policy.${row.nameKey}`)}
										</td>
										<td class="py-3 pr-4 leading-relaxed" style:color="var(--text-muted)">
											{$_(`settings.notifications.policy.${row.hintKey}`)}
											{#if !editable}
												<p class="mt-1 text-xs">{policyDisabledHint(row.triggerKey)}</p>
											{/if}
										</td>
										<td class="py-3 text-right">
											<IntegerInput
												class="input w-20 text-right tabular-nums"
												min={row.min}
												max={row.max}
												disabled={!editable}
												value={policyFieldValue(row.key)}
												onchange={(v) => setPolicyFieldValue(row.key, v)}
											/>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<div class="space-y-3 md:hidden">
						{#each notificationPolicyRows() as row (row.key)}
							{@const editable = policyFieldEditable(row.triggerKey)}
							<article
								class="rounded-xl border p-4"
								class:opacity-50={!editable}
								style:border-color="var(--border)"
							>
								<p class="font-medium leading-snug">
									{$_(`settings.notifications.policy.${row.nameKey}`)}
								</p>
								<p class="mt-1 text-sm leading-relaxed" style:color="var(--text-muted)">
									{$_(`settings.notifications.policy.${row.hintKey}`)}
									{#if !editable}
										<span class="mt-1 block text-xs">{policyDisabledHint(row.triggerKey)}</span>
									{/if}
								</p>
								<div class="mt-3 flex justify-end">
									<IntegerInput
										class="input w-20 text-right tabular-nums"
										min={row.min}
										max={row.max}
										disabled={!editable}
										value={policyFieldValue(row.key)}
										onchange={(v) => setPolicyFieldValue(row.key, v)}
									/>
								</div>
							</article>
						{/each}
					</div>

					<section
						class="space-y-3 border-t pt-6"
						class:opacity-50={!policyScheduleEditable()}
						style:border-color="var(--border)"
					>
						<h4 class="text-sm font-medium">
							{$_('settings.notifications.triggers.schedule_title')}
						</h4>
						{#if !policyScheduleEditable()}
							<p class="text-sm" style:color="var(--text-muted)">
								{policyScheduleDisabledHint()}
							</p>
						{/if}
						<label class="block max-w-xs space-y-1.5 text-sm">
							<span class="block leading-snug"
								>{$_('settings.notifications.triggers.send_time')}</span
							>
							<input
								class="input w-full"
								type="time"
								disabled={!policyScheduleEditable()}
								bind:value={notificationTimeLocal}
							/>
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
					{#if blockFeedback.triggerPolicy.error}
						<p class="text-sm" style:color="var(--danger)">{blockFeedback.triggerPolicy.error}</p>
					{/if}
					{#if blockFeedback.triggerPolicy.success}
						<p class="text-sm" style:color="var(--primary)">
							{blockFeedback.triggerPolicy.success}
						</p>
					{/if}
				</div>

				{#each visibleTemplates(templates) as tpl (tpl.trigger_type)}
					{@const editable = templateEditable(tpl.trigger_type)}
					<div
						class="card space-y-4"
						class:opacity-50={!editable}
						class:pointer-events-none={!editable}
						class:select-none={!editable}
					>
						<h3 class="text-lg font-semibold">{triggerLabel(tpl.trigger_type)}</h3>
						{#if !editable}
							<p class="text-sm" style:color="var(--text-muted)">
								{templateDisabledHint(tpl.trigger_type)}
							</p>
						{/if}
						<textarea
							id={templateTextareaId(tpl.trigger_type)}
							class="input min-h-[88px]"
							disabled={!editable}
							readonly={!editable}
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
									disabled={!editable}
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
								disabled={!editable}
								onclick={() =>
									previewTemplate(tpl.trigger_type, templateValue(tpl.trigger_type, tpl.template))}
							>
								{$_('settings.notifications.templates.preview')}
							</button>
							<button
								type="button"
								class="btn-ghost"
								disabled={!editable}
								onclick={() => resetTemplate(tpl.trigger_type)}
							>
								{$_('settings.notifications.templates.reset')}
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
						{#if templateFeedback[tpl.trigger_type]?.error}
							<p class="text-sm" style:color="var(--danger)">
								{templateFeedback[tpl.trigger_type].error}
							</p>
						{/if}
						{#if templateFeedback[tpl.trigger_type]?.success}
							<p class="text-sm" style:color="var(--primary)">
								{templateFeedback[tpl.trigger_type].success}
							</p>
						{/if}
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
			</div>
		</div>
	{/if}
{:else if tab === 'categories'}
	<CategoriesTab />
{:else if tab === 'import'}
	<ImportTab />
{:else if tab === 'admin'}
	{#if !$user?.is_admin}
		<div class="card max-w-lg">
			<p style:color="var(--text-muted)">{$_('common.error')}</p>
		</div>
	{:else}
		<div class="mb-6">
			<PageTabs
				active={adminTab}
				tabs={adminTabs}
				onchange={(next) => selectAdminTab(next as AdminTab)}
			/>
		</div>
		<div class="space-y-4">
			{#if adminTab === 'system'}
				<AdminSystemTab />
			{:else if adminTab === 'users'}
				<AdminUsersTab />
			{:else if adminTab === 'backups'}
				<AdminBackupsTab />
			{:else}
				<AdminDiagnosticsTab />
			{/if}
			<AdminSupportLinks />
		</div>
	{/if}
{:else if tab === 'tokens'}
	<div class="space-y-6">
		<form class="card space-y-4" onsubmit={handleCreateToken}>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="mb-1.5 block text-sm font-medium" for="token-name"
						>{$_('settings.tokens.name')}</label
					>
					<input
						id="token-name"
						class="input w-full"
						bind:value={newTokenName}
						placeholder="Home Assistant"
						required
					/>
				</div>
				<DateTimePicker
					id="token-expires"
					label={$_('settings.tokens.expires')}
					bind:value={newTokenExpiresAt}
					disabled={newTokenNeverExpires}
					required={!newTokenNeverExpires}
					{...dateOnlyPicker}
				/>
			</div>
			<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<label class="flex cursor-pointer items-center gap-3">
					<ToggleSwitch
						checked={newTokenNeverExpires}
						label={$_('settings.tokens.never_expires')}
						onchange={() => (newTokenNeverExpires = !newTokenNeverExpires)}
					/>
					<span class="text-sm">{$_('settings.tokens.never_expires')}</span>
				</label>
				<button type="submit" class="btn-primary w-full sm:w-auto" disabled={loading}>
					{$_('common.create')}
				</button>
			</div>
			{#if newTokenNeverExpires}
				<p class="text-sm font-medium" style:color="var(--danger)">
					{$_('settings.tokens.perpetual_warning')}
				</p>
			{/if}
			<FormFeedback error={tokenFormError} />
		</form>

		{#if createdToken}
			{@const newToken = createdToken}
			<ModalShell open={true} title={$_('settings.tokens.created.title')} onclose={closeTokenModal}>
				<p class="mb-4 text-sm" style:color="var(--text-muted)">
					{$_('settings.tokens.created.hint')}
				</p>
				<code
					class="block overflow-x-auto rounded-lg px-3 py-2 text-sm"
					style:background-color="var(--bg)">{newToken.token}</code
				>
				{#snippet footer()}
					<button type="button" class="btn-primary" onclick={() => copyToken(newToken.token)}>
						{$_('settings.tokens.copy')}
					</button>
					<button type="button" class="btn-ghost" onclick={closeTokenModal}
						>{$_('common.close')}</button
					>
				{/snippet}
			</ModalShell>
		{/if}

		<div class="card md:overflow-x-auto">
			<div class="hidden md:block">
				<table class="w-full text-left text-sm">
					<thead>
						<tr style:color="var(--text-muted)">
							<th class="pb-3 pr-4">{$_('settings.tokens.col.name')}</th>
							<th class="pb-3 pr-4">Prefix</th>
							<th class="pb-3 pr-4">{$_('settings.tokens.col.expires')}</th>
							<th class="pb-3 pr-4">{$_('settings.tokens.col.last_used')}</th>
							<th class="pb-3"></th>
						</tr>
					</thead>
					<tbody>
						{#each tokens as t (t.id)}
							<tr class="border-t" style:border-color="var(--border)">
								<td class="py-3 pr-4">{t.name}</td>
								<td class="py-3 pr-4 font-mono">{t.token_prefix}</td>
								<td class="py-3 pr-4">{formatTokenExpiry(t.expires_at)}</td>
								<td class="py-3 pr-4">{formatOptional(t.last_used_at)}</td>
								<td class="py-3 text-right">
									<button type="button" class="btn-ghost" onclick={() => revokeToken(t.id)}>
										{$_('common.delete')}
									</button>
								</td>
							</tr>
						{:else}
							<tr>
								<td colspan="5" class="py-4" style:color="var(--text-muted)"
									>{$_('settings.tokens.empty')}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="space-y-3 md:hidden">
				{#each tokens as t (t.id)}
					<article class="rounded-xl border p-4" style:border-color="var(--border)">
						<p class="font-medium">{t.name}</p>
						<dl class="mt-2 grid gap-2 text-sm">
							<div class="flex justify-between gap-2">
								<dt style:color="var(--text-muted)">Prefix</dt>
								<dd class="font-mono">{t.token_prefix}</dd>
							</div>
							<div class="flex justify-between gap-2">
								<dt style:color="var(--text-muted)">{$_('settings.tokens.col.expires')}</dt>
								<dd>{formatTokenExpiry(t.expires_at)}</dd>
							</div>
							<div class="flex justify-between gap-2">
								<dt style:color="var(--text-muted)">{$_('settings.tokens.col.last_used')}</dt>
								<dd>{formatOptional(t.last_used_at)}</dd>
							</div>
						</dl>
						<button type="button" class="btn-ghost mt-3 w-full" onclick={() => revokeToken(t.id)}>
							{$_('common.delete')}
						</button>
					</article>
				{:else}
					<p class="py-4 text-sm" style:color="var(--text-muted)">{$_('settings.tokens.empty')}</p>
				{/each}
			</div>
		</div>
	</div>
{/if}
