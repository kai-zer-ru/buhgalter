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
		getUserSettings,
		listTokens,
		previewNotificationTemplate,
		putUserSettings,
		putNotificationSettings,
		resetNotificationTemplates,
		sendNotificationTest,
		type APIToken,
		type APITokenCreated,
		type NotificationTemplate
	} from '$lib/api/client';
	import { user } from '$lib/stores/auth';
	import { applyTheme } from '$lib/theme';
	import { setLocale } from '$lib/i18n';
	import TimezonePicker from '$lib/components/TimezonePicker.svelte';
	import PageTabs from '$lib/components/PageTabs.svelte';
	import Select from '$lib/components/Select.svelte';
	import FormFeedback from '$lib/components/FormFeedback.svelte';
	import ModalShell from '$lib/components/ModalShell.svelte';
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { confirm } from '$lib/confirm';
	import { validatePasswordPolicy } from '$lib/password-policy';
	import { formatApiError } from '$lib/api/errors';
	import { toast } from '$lib/toast';
	import CategoriesTab from '$lib/settings/CategoriesTab.svelte';
	import ImportTab from '$lib/settings/ImportTab.svelte';
	import AdminSystemTab from '../admin/+page.svelte';
	import AdminUsersTab from '../admin/users/+page.svelte';
	import AdminBackupsTab from '../admin/backups/+page.svelte';
	import AdminDiagnosticsTab from '../admin/diagnostics/+page.svelte';

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
	let notificationsFormFeedback = $state({ error: '', success: '' });
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
		'password_reset',
		'test'
	];

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
		if (tab !== 'notifications') return;
		void loadNotifications().catch((err) => {
			notificationsFormFeedback = {
				error: err instanceof ApiError ? err.message : $_('common.error'),
				success: ''
			};
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
		notificationsFormFeedback = { error: '', success: '' };
		if (
			!validateDaysRange(debtDaysBefore) ||
			!validateDaysRange(creditDaysBefore) ||
			!validateOverdueDaysLimit(myDebtOverdueDaysLimit) ||
			!validateOverdueDaysLimit(owedDebtOverdueStartAfterDays) ||
			!validateOverdueDaysLimit(owedDebtOverdueDaysLimit)
		) {
			notificationsFormFeedback = {
				error: $_('settings.notifications.error.policy_range'),
				success: ''
			};
			return;
		}
		if (!validateLocalTime(notificationTimeLocal)) {
			notificationsFormFeedback = {
				error: $_('settings.notifications.error.time_format'),
				success: ''
			};
			return;
		}
		for (const [triggerType, template] of Object.entries(templatesDirty)) {
			const templateError = validateTemplateText(template);
			if (templateError) {
				notificationsFormFeedback = {
					error: `${triggerLabel(triggerType)}: ${templateError}`,
					success: ''
				};
				return;
			}
		}
		loading = true;
		try {
			const templateUpdates = Object.entries(templatesDirty).map(([trigger_type, template]) => ({
				trigger_type,
				template
			}));
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
				notification_time_local: notificationTimeLocal.trim(),
				templates: templateUpdates.length > 0 ? templateUpdates : undefined
			});
			telegramBotToken = '';
			maxToken = '';
			await loadNotifications();
			notificationsFormFeedback = {
				error: '',
				success: $_('settings.notifications.success.saved')
			};
			toast($_('settings.notifications.success.saved'));
		} catch (err) {
			notificationsFormFeedback = {
				error: err instanceof ApiError ? err.message : $_('common.error'),
				success: ''
			};
		} finally {
			loading = false;
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
			await putNotificationSettings({
				telegram_enabled: telegramEnabled,
				telegram_bot_token: telegramBotToken.trim() || undefined,
				telegram_chat_id: telegramChatId.trim() || undefined
			});
			telegramBotToken = '';
			await loadNotifications();
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
			await putNotificationSettings({
				max_enabled: maxEnabled,
				max_provider: maxProvider,
				max_token: maxToken.trim() || undefined,
				max_user_id: maxUserId.trim() ? Number(maxUserId) : null,
				max_recipient_id: maxRecipientId.trim() ? Number(maxRecipientId) : null
			});
			maxToken = '';
			await loadNotifications();
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
			await putNotificationSettings({
				trigger_debt: triggerDebt,
				trigger_credit: triggerCredit,
				trigger_planned: triggerPlanned,
				trigger_password_reset: $user?.is_admin ? triggerPasswordReset : undefined
			});
			await loadNotifications();
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
			await putNotificationSettings({
				debt_days_before: debtDaysBefore,
				my_debt_overdue_days_limit: myDebtOverdueDaysLimit,
				owed_debt_overdue_start_after_days: owedDebtOverdueStartAfterDays,
				owed_debt_overdue_days_limit: owedDebtOverdueDaysLimit,
				credit_days_before: creditDaysBefore,
				notification_time_local: notificationTimeLocal.trim()
			});
			await loadNotifications();
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
			await putNotificationSettings({
				templates: [{ trigger_type: triggerType, template }]
			});
			await loadNotifications();
			templateFeedback = {
				...templateFeedback,
				[triggerType]: {
					error: '',
					success: $_('settings.notifications.success.template_saved', {
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
		} finally {
			loading = false;
		}
	}

	async function previewTemplate(triggerType: string, template: string) {
		if (!notificationSecretConfigured) return;
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
		templateFeedback = {
			...templateFeedback,
			[triggerType]: { error: '', success: '' }
		};
		try {
			await resetNotificationTemplates(triggerType);
			await loadNotifications();
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
			createdToken = await createToken(newTokenName.trim(), null);
			newTokenName = '';
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
</script>

<svelte:head>
	<title>{$_('settings.title')} — {$_('app.title')}</title>
</svelte:head>

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
						<div class="grid w-full max-w-md gap-2">
							<div class="grid grid-cols-[1fr_auto] items-center gap-3">
								<span class="text-sm leading-tight"
									>{$_('settings.notifications.triggers.debt')}</span
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
						{#if blockFeedback.triggerPolicy.error}
							<p class="text-sm" style:color="var(--danger)">{blockFeedback.triggerPolicy.error}</p>
						{/if}
						{#if blockFeedback.triggerPolicy.success}
							<p class="text-sm" style:color="var(--primary)">
								{blockFeedback.triggerPolicy.success}
							</p>
						{/if}
					</div>

					<div class="card space-y-4">
						<h3 class="text-lg font-semibold">{$_('settings.notifications.templates.title')}</h3>
						{#each orderedTemplates(templates) as tpl (tpl.trigger_type)}
							<details class="rounded-lg border p-3" style:border-color="var(--border)" open>
								<summary class="flex cursor-pointer items-center justify-between gap-2">
									<span class="font-medium">{triggerLabel(tpl.trigger_type)}</span>
									<span class="text-xs" style:color="var(--text-muted)">
										{tpl.is_custom
											? $_('settings.notifications.templates.state.custom')
											: $_('settings.notifications.templates.state.default')}
									</span>
								</summary>
								<div class="mt-3 space-y-2">
									<textarea
										id={templateTextareaId(tpl.trigger_type)}
										class="input min-h-[88px]"
										value={templateValue(tpl.trigger_type, tpl.template)}
										oninput={(e) =>
											updateTemplate(
												tpl.trigger_type,
												(e.currentTarget as HTMLTextAreaElement).value
											)}></textarea>
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
									<div class="flex flex-wrap gap-2">
										<button
											type="button"
											class="btn-ghost"
											onclick={() =>
												saveTemplate(
													tpl.trigger_type,
													templateValue(tpl.trigger_type, tpl.template)
												)}
											disabled={loading}
										>
											{$_('settings.notifications.templates.save')}
										</button>
										<button
											type="button"
											class="btn-ghost"
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
											class="btn-ghost"
											onclick={() => resetTemplate(tpl.trigger_type)}
										>
											{$_('settings.notifications.templates.reset')}
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
							</details>
						{/each}
					</div>
					<FormFeedback
						error={notificationsFormFeedback.error}
						success={notificationsFormFeedback.success}
					/>
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
		{#if adminTab === 'system'}
			<AdminSystemTab />
		{:else if adminTab === 'users'}
			<AdminUsersTab />
		{:else if adminTab === 'backups'}
			<AdminBackupsTab />
		{:else}
			<AdminDiagnosticsTab />
		{/if}
	{/if}
{:else if tab === 'tokens'}
	<div class="space-y-6">
		<form class="card flex flex-wrap items-end gap-3" onsubmit={handleCreateToken}>
			<div class="min-w-[200px] flex-1">
				<label class="mb-1.5 block text-sm font-medium" for="token-name"
					>{$_('settings.tokens.name')}</label
				>
				<input
					id="token-name"
					class="input"
					bind:value={newTokenName}
					placeholder="Home Assistant"
					required
				/>
			</div>
			<button type="submit" class="btn-primary" disabled={loading}>{$_('common.create')}</button>
			<div class="w-full basis-full">
				<FormFeedback error={tokenFormError} />
			</div>
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
								<td class="py-3 pr-4">{formatOptional(t.expires_at)}</td>
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
								<dd>{formatOptional(t.expires_at)}</dd>
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
