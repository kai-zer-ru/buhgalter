import { get } from 'svelte/store';
import { locale } from 'svelte-i18n';
import { cachedGet, invalidateApiCache } from '$lib/api/cache';

const API_BASE = '';

function acceptLanguage(): string {
	const code = get(locale);
	return code === 'en' ? 'en' : 'ru';
}

export class ApiError extends Error {
	constructor(
		public code: string,
		message: string,
		public status: number
	) {
		super(message);
		this.name = 'ApiError';
	}
}

/** HTTP statuses that usually mean the server or proxy is temporarily down. */
export function isTransientHttpError(status: number): boolean {
	return status === 408 || status === 502 || status === 503 || status === 504;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		credentials: 'include',
		headers: {
			Accept: 'application/json',
			'Accept-Language': acceptLanguage(),
			...(init?.body && !(init.body instanceof FormData)
				? { 'Content-Type': 'application/json' }
				: {}),
			...init?.headers
		},
		...init
	});

	if (!res.ok) {
		let code = 'UNKNOWN';
		let message = res.statusText;
		try {
			const body = await res.json();
			code = body?.error?.code ?? code;
			message = body?.error?.message ?? message;
		} catch {
			// ignore
		}
		throw new ApiError(code, message, res.status);
	}

	if (res.status === 204) {
		return undefined as T;
	}
	return res.json() as Promise<T>;
}

export type SetupStatus = {
	configured: boolean;
	database: string;
	registration_enabled: boolean;
	external_url: string;
};

export type SetupPayload = {
	admin_login: string;
	admin_display_name: string;
	admin_password: string;
	admin_password_confirm: string;
	registration_enabled: boolean;
	external_url: string;
};

export type SetupRestoreResponse = {
	message: string;
	configured: boolean;
};

export type User = {
	id: string;
	login: string;
	display_name: string;
	is_admin: boolean;
	language: string;
	currency: string;
	timezone: string;
	theme: string;
};

export type UserSettings = {
	display_name: string;
	language: string;
	currency: string;
	timezone: string;
	theme: string;
};

export type NotificationTemplate = {
	trigger_type: string;
	template: string;
	placeholders: string[];
	is_custom: boolean;
};

export type NotificationSettings = {
	secret_key_configured: boolean;
	telegram_enabled: boolean;
	telegram_configured: boolean;
	telegram_chat_id?: string | null;
	max_enabled: boolean;
	max_configured: boolean;
	max_provider?: 'a161' | 'official' | null;
	max_user_id?: number | null;
	max_recipient_id?: number | null;
	trigger_debt: boolean;
	trigger_credit: boolean;
	trigger_planned: boolean;
	trigger_password_reset?: boolean;
	debt_days_before: number;
	my_debt_overdue_days_limit: number;
	owed_debt_overdue_start_after_days: number;
	owed_debt_overdue_days_limit: number;
	credit_days_before: number;
	notification_time_local: string;
	templates: NotificationTemplate[];
};

export type NotificationSettingsUpdate = {
	telegram_enabled?: boolean;
	telegram_bot_token?: string;
	telegram_chat_id?: string;
	max_enabled?: boolean;
	max_provider?: 'a161' | 'official';
	max_token?: string;
	max_user_id?: number | null;
	max_recipient_id?: number | null;
	trigger_debt?: boolean;
	trigger_credit?: boolean;
	trigger_planned?: boolean;
	trigger_password_reset?: boolean;
	debt_days_before?: number;
	my_debt_overdue_days_limit?: number;
	owed_debt_overdue_start_after_days?: number;
	owed_debt_overdue_days_limit?: number;
	credit_days_before?: number;
	notification_time_local?: string;
	templates?: Array<{ trigger_type: string; template: string }>;
};

export type AdminSettings = {
	registration_enabled: boolean;
	external_url: string;
	secret_key_set: boolean;
};

export type AdminSettingsUpdate = {
	registration_enabled: boolean;
	external_url: string;
};

export type AdminDiagnostics = {
	app_version: string;
	build_commit: string;
	build_time: string;
	db_migration_version: number;
	install_method: string;
	previous_app_version: string | null;
	go_version: string;
	os: string;
	arch: string;
	uptime_seconds: number;
	db_size_bytes: number;
	users_count: number;
	data_dir: string;
	log_dir: string;
	addr: string;
	static_embed: boolean;
	external_url: string;
	env: Record<string, string>;
};

export type AdminUser = {
	id: string;
	login: string;
	display_name: string;
	is_admin: boolean;
	created_at: string;
};

export type APIToken = {
	id: string;
	name: string;
	token_prefix: string;
	expires_at: string | null;
	last_used_at: string | null;
	created_at: string;
};

export type APITokenCreated = APIToken & { token: string };

export type BackupFile = {
	filename: string;
	size: number;
	created_at: string;
};

export type BackupSettings = {
	backup_enabled: boolean;
	backup_time: string;
	backup_retention: number;
};

export function getSetupStatus() {
	return request<SetupStatus>('/api/v1/setup/status');
}

export function postSetup(payload: SetupPayload) {
	return request<{ message: string }>('/api/v1/setup', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function postSetupRestore(file: File, confirm: string = 'RESTORE') {
	const form = new FormData();
	form.append('file', file);
	form.append('confirm', confirm);
	return request<SetupRestoreResponse>('/api/v1/setup/restore', {
		method: 'POST',
		body: form
	});
}

export function getHealth() {
	return request<{ status: string; version: string; db: string }>('/api/v1/health');
}

export function login(login: string, password: string) {
	return request<{ token: string; user: User }>('/api/v1/auth/login', {
		method: 'POST',
		body: JSON.stringify({ login, password })
	});
}

export function register(
	login: string,
	password: string,
	passwordConfirm: string,
	displayName: string
) {
	return request<{ token: string; user: User }>('/api/v1/auth/register', {
		method: 'POST',
		body: JSON.stringify({
			login,
			password,
			password_confirm: passwordConfirm,
			display_name: displayName
		})
	});
}

export function logout() {
	invalidateApiCache();
	return request<void>('/api/v1/auth/logout', { method: 'POST' });
}

export type PasswordResetRequest = {
	id: string;
	user_id: string;
	login: string;
	display_name: string;
	created_at: string;
};

export function requestPasswordReset(login: string) {
	return request<void>('/api/v1/auth/request-password-reset', {
		method: 'POST',
		body: JSON.stringify({ login })
	});
}

export function listPasswordResetRequests() {
	return request<PasswordResetRequest[]>('/api/v1/admin/password-reset-requests');
}

export function ackPasswordResetRequest(id: string) {
	return request<void>(`/api/v1/admin/password-reset-requests/${id}/ack`, { method: 'POST' });
}

export function resetAdminUserPassword(
	id: string,
	payload: { new_password: string; new_password_confirm: string }
) {
	return request<void>(`/api/v1/admin/users/${id}/password`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function getMe() {
	return request<User>('/api/v1/auth/me');
}

export function getUserSettings() {
	return request<UserSettings>('/api/v1/user/settings');
}

export function putUserSettings(settings: UserSettings) {
	return request<UserSettings>('/api/v1/user/settings', {
		method: 'PUT',
		body: JSON.stringify(settings)
	});
}

export function getNotificationSettings() {
	return request<NotificationSettings>('/api/v1/user/notifications');
}

export function putNotificationSettings(payload: NotificationSettingsUpdate) {
	return request<NotificationSettings>('/api/v1/user/notifications', {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function sendNotificationTest(channel: 'telegram' | 'max') {
	return request<{ status: string }>('/api/v1/user/notifications/test', {
		method: 'POST',
		body: JSON.stringify({ channel })
	});
}

export function previewNotificationTemplate(payload: { trigger_type: string; template: string }) {
	return request<{ text: string }>('/api/v1/user/notifications/templates/preview', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function resetNotificationTemplates(triggerType?: string) {
	return request<NotificationSettings>('/api/v1/user/notifications/templates/reset', {
		method: 'POST',
		body: JSON.stringify(triggerType ? { trigger_type: triggerType } : {})
	});
}

export function changePassword(
	currentPassword: string,
	newPassword: string,
	newPasswordConfirm: string
) {
	return request<void>('/api/v1/user/password', {
		method: 'PUT',
		body: JSON.stringify({
			current_password: currentPassword,
			new_password: newPassword,
			new_password_confirm: newPasswordConfirm
		})
	});
}

export function listTokens() {
	return request<APIToken[]>('/api/v1/user/tokens');
}

export function createToken(name: string, expiresAt: string | null) {
	return request<APITokenCreated>('/api/v1/user/tokens', {
		method: 'POST',
		body: JSON.stringify({ name, expires_at: expiresAt })
	});
}

export function deleteToken(id: string) {
	return request<void>(`/api/v1/user/tokens/${id}`, { method: 'DELETE' });
}

export function getAdminSettings() {
	return request<AdminSettings>('/api/v1/admin/settings');
}

export function putAdminSettings(settings: AdminSettingsUpdate) {
	return request<AdminSettings>('/api/v1/admin/settings', {
		method: 'PUT',
		body: JSON.stringify(settings)
	});
}

export function putAdminNotificationSecretKey(notificationSecretKey: string) {
	return request<AdminSettings>('/api/v1/admin/settings/notification-secret', {
		method: 'PUT',
		body: JSON.stringify({ notification_secret_key: notificationSecretKey })
	});
}

export function listAdminUsers() {
	return request<AdminUser[]>('/api/v1/admin/users');
}

export function createAdminUser(payload: {
	login: string;
	password: string;
	password_confirm: string;
	display_name: string;
	is_admin: boolean;
}) {
	return request<AdminUser>('/api/v1/admin/users', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function deleteAdminUser(id: string) {
	return request<void>(`/api/v1/admin/users/${id}`, { method: 'DELETE' });
}

export function getAdminDiagnostics() {
	return request<AdminDiagnostics>('/api/v1/admin/diagnostics');
}

export function listBackups() {
	return request<BackupFile[]>('/api/v1/admin/backups');
}

export function getBackupSettings() {
	return request<BackupSettings>('/api/v1/admin/backups/settings');
}

export function putBackupSettings(settings: BackupSettings) {
	return request<BackupSettings>('/api/v1/admin/backups/settings', {
		method: 'PUT',
		body: JSON.stringify(settings)
	});
}

export function runBackup() {
	return request<{ filename: string }>('/api/v1/admin/backups/run', { method: 'POST' });
}

export function restoreBackup(file: File, confirm: string) {
	const form = new FormData();
	form.append('file', file);
	form.append('confirm', confirm);
	return request<{ message: string }>('/api/v1/admin/backups/restore', {
		method: 'POST',
		body: form
	});
}

export function backupDownloadUrl(filename?: string) {
	if (filename) {
		return `/api/v1/admin/backups/${encodeURIComponent(filename)}/download`;
	}
	return '/api/v1/admin/backups/download';
}

export async function getRegistrationEnabled(): Promise<boolean> {
	try {
		const s = await getSetupStatus();
		return s.registration_enabled;
	} catch {
		return false;
	}
}

export type Bank = {
	id: string;
	name: string;
	bic: string | null;
	icon_path: string;
	sort_order: number;
};

export type Account = {
	id: string;
	name: string;
	type: 'cash' | 'bank';
	bank_id: string | null;
	bank_name?: string | null;
	bank_icon?: string | null;
	initial_balance: number;
	balance: number;
	balance_display: string;
	status: 'active' | 'archived';
	is_primary: boolean;
	created_at: string;
	updated_at: string;
};

export type Category = {
	id: string;
	name: string;
	type: 'income' | 'expense';
	icon: string;
	sort_order: number;
	is_primary: boolean;
	is_system: boolean;
	subcategory_count: number;
	created_at: string;
};

export type Subcategory = {
	id: string;
	category_id: string;
	name: string;
	icon: string;
	sort_order: number;
	created_at: string;
};

export function listBanks() {
	return cachedGet('/api/v1/banks', () => request<Bank[]>('/api/v1/banks'));
}

export function listAccounts(status?: 'active' | 'archived') {
	const q = status ? `?status=${status}` : '';
	const path = `/api/v1/accounts${q}`;
	return request<Account[]>(path);
}

export function getAccount(id: string) {
	return request<Account>(`/api/v1/accounts/${id}`);
}

export function createAccount(payload: {
	name: string;
	type: 'cash' | 'bank';
	bank_id?: string;
	initial_balance: string;
}) {
	return request<Account>('/api/v1/accounts', {
		method: 'POST',
		body: JSON.stringify(payload)
	}).then((account) => {
		invalidateApiCache('/api/v1/accounts');
		return account;
	});
}

export function updateAccount(
	id: string,
	payload: { name: string; bank_id?: string; initial_balance?: string }
) {
	return request<Account>(`/api/v1/accounts/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	}).then((account) => {
		invalidateApiCache('/api/v1/accounts');
		return account;
	});
}

export function archiveAccount(id: string) {
	return request<Account>(`/api/v1/accounts/${id}/archive`, { method: 'POST' }).then((account) => {
		invalidateApiCache('/api/v1/accounts');
		return account;
	});
}

export function unarchiveAccount(id: string) {
	return request<Account>(`/api/v1/accounts/${id}/unarchive`, { method: 'POST' }).then(
		(account) => {
			invalidateApiCache('/api/v1/accounts');
			return account;
		}
	);
}

export function setPrimaryAccount(id: string) {
	return request<Account>(`/api/v1/accounts/${id}/primary`, { method: 'POST' }).then((account) => {
		invalidateApiCache('/api/v1/accounts');
		return account;
	});
}

export function deleteAccount(id: string) {
	return request<void>(`/api/v1/accounts/${id}`, { method: 'DELETE' }).then((result) => {
		invalidateApiCache('/api/v1/accounts');
		return result;
	});
}

export function listCategories(type?: 'income' | 'expense') {
	const q = type ? `?type=${type}` : '';
	const path = `/api/v1/categories${q}`;
	return cachedGet(path, () => request<Category[]>(path));
}

function invalidateCategoriesCache() {
	invalidateApiCache('/api/v1/categories');
}

export function createCategory(payload: {
	name: string;
	type: 'income' | 'expense';
	icon: string;
	sort_order?: number;
}) {
	return request<Category>('/api/v1/categories', {
		method: 'POST',
		body: JSON.stringify(payload)
	}).then((category) => {
		invalidateCategoriesCache();
		return category;
	});
}

export function updateCategory(
	id: string,
	payload: { name: string; icon: string; sort_order?: number }
) {
	return request<Category>(`/api/v1/categories/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	}).then((category) => {
		invalidateCategoriesCache();
		return category;
	});
}

export function deleteCategory(id: string) {
	return request<void>(`/api/v1/categories/${id}`, { method: 'DELETE' }).then((result) => {
		invalidateCategoriesCache();
		return result;
	});
}

export function reorderCategories(type: 'income' | 'expense', ids: string[]) {
	return request<Category[]>('/api/v1/categories/order', {
		method: 'PUT',
		body: JSON.stringify({ type, ids })
	}).then((categories) => {
		invalidateCategoriesCache();
		return categories;
	});
}

export function setPrimaryCategory(id: string) {
	return request<Category>(`/api/v1/categories/${id}/primary`, { method: 'POST' }).then(
		(category) => {
			invalidateCategoriesCache();
			return category;
		}
	);
}

export function listSubcategories(categoryId: string) {
	return request<Subcategory[]>(`/api/v1/categories/${categoryId}/subcategories`);
}

export function reorderSubcategories(categoryId: string, ids: string[]) {
	return request<Subcategory[]>(`/api/v1/categories/${categoryId}/subcategories/order`, {
		method: 'PUT',
		body: JSON.stringify({ ids })
	}).then((subcategories) => {
		invalidateCategoriesCache();
		return subcategories;
	});
}

export function createSubcategory(categoryId: string, payload: { name: string; icon?: string }) {
	return request<Subcategory>(`/api/v1/categories/${categoryId}/subcategories`, {
		method: 'POST',
		body: JSON.stringify(payload)
	}).then((subcategory) => {
		invalidateCategoriesCache();
		return subcategory;
	});
}

export function updateSubcategory(id: string, payload: { name: string; icon?: string }) {
	return request<Subcategory>(`/api/v1/subcategories/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	}).then((subcategory) => {
		invalidateCategoriesCache();
		return subcategory;
	});
}

export function deleteSubcategory(id: string) {
	return request<void>(`/api/v1/subcategories/${id}`, { method: 'DELETE' }).then((result) => {
		invalidateCategoriesCache();
		return result;
	});
}

export type Transaction = {
	id: string;
	account_id: string;
	account_name?: string;
	transfer_account_name?: string;
	type: 'income' | 'expense' | 'transfer';
	kind: 'manual' | 'future';
	amount: number;
	amount_display: string;
	description: string | null;
	category_id: string | null;
	category_name?: string | null;
	category_icon?: string | null;
	category_is_system?: boolean;
	subcategory_id: string | null;
	subcategory_name?: string | null;
	transfer_group_id?: string | null;
	transfer_account_id?: string | null;
	transfer_is_out?: boolean;
	credit_payment_linked?: boolean;
	transaction_date: string;
	created_at: string;
	updated_at: string;
};

export type RecurringOperation = {
	id: string;
	type: 'income' | 'expense';
	amount: number;
	amount_display: string;
	description: string | null;
	account_id: string;
	account_name: string;
	category_id: string;
	category_name: string;
	subcategory_id: string | null;
	subcategory_name: string | null;
	period: 'week' | 'two_weeks' | 'month' | 'year';
	weekday: number | null;
	day_of_month: number | null;
	start_date: string;
	time_local: string;
	next_run_at: string;
	last_run_at: string | null;
	active: boolean;
	created_at: string;
	updated_at: string;
};

export type TransactionList = {
	data: Transaction[];
	meta: { page: number; limit: number; total: number };
};

export type StatsSummary = {
	income_total: number;
	expense_total: number;
	balance_delta: number;
	transaction_count: number;
};

export type StatsCategoryItem = {
	category_id: string;
	category_name: string;
	icon: string;
	type: 'income' | 'expense';
	total: number;
	percentage: number;
	count: number;
};

export type StatsSubcategoryItem = {
	category_id: string;
	category_name: string;
	category_icon: string;
	subcategory_id: string;
	subcategory_name: string;
	total: number;
	percentage: number;
	count: number;
};

export type StatsPeriodItem = {
	period: string;
	income: number;
	expense: number;
};

export type StatsContext = StatsSummary & {
	scope: 'all' | 'account' | 'debtor' | 'credit' | 'debts';
	scope_id?: string;
	lent_total?: number;
	borrowed_total?: number;
	paid_total?: number;
	payment_count?: number;
	remaining_amount?: number;
};

export type AccountBalanceSummary = {
	id: string;
	name: string;
	type: 'cash' | 'bank';
	bank_icon?: string | null;
	balance: number;
	balance_display: string;
	forecast_balance: number;
	forecast_display: string;
	has_future_this_month: boolean;
};

export type AccountsSummary = {
	accounts: AccountBalanceSummary[];
	total_balance: number;
	total_forecast: number;
};

export type Dashboard = {
	total_balance: number;
	total_forecast: number;
	accounts: AccountBalanceSummary[];
	recent_transactions: Transaction[];
	debts_summary: DebtsSummary;
};

export type DebtsSummary = {
	i_owe: number;
	owed_to_me: number;
	overdue_i_owe: number;
	overdue_owed_to_me: number;
	active_count: number;
};

export type DebtTransaction = {
	id: string;
	account_id: string;
	account_name?: string;
	type: string;
	kind: string;
	amount: number;
	amount_display: string;
	description: string | null;
	category_name?: string;
	transaction_date: string;
};

export type Debtor = {
	id: string;
	name: string;
	created_at: string;
};

export type DebtorDetail = Debtor & {
	i_owe: number;
	owed_to_me: number;
	debts: Debt[];
	transactions: DebtTransaction[];
};

export type Debt = {
	id: string;
	debtor_id: string;
	debtor_name: string;
	direction: 'lent' | 'borrowed';
	amount: number;
	amount_display: string;
	affects_balance: boolean;
	debt_date: string;
	due_date: string;
	description: string | null;
	transaction_id: string | null;
	is_settled: boolean;
	settled_at: string | null;
	is_overdue: boolean;
	created_at: string;
	account_id?: string | null;
	account_name?: string | null;
};

export type Transfer = {
	group_id: string;
	from_account_id: string;
	to_account_id: string;
	amount: number;
	amount_display: string;
	commission: number;
	commission_display: string;
	description: string | null;
	transaction_date: string;
	kind: string;
	legs: Transaction[];
};

export function listTransactions(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<TransactionList>(`/api/v1/transactions${q}`);
}

export function getTransaction(id: string) {
	return request<Transaction>(`/api/v1/transactions/${id}`);
}

export function createTransaction(payload: {
	account_id: string;
	type: 'income' | 'expense';
	amount: string;
	description?: string;
	category_id?: string;
	subcategory_id?: string;
	subcategory_name?: string;
	transaction_date: string;
}) {
	return request<Transaction>('/api/v1/transactions', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function updateTransaction(
	id: string,
	payload: {
		account_id: string;
		type: 'income' | 'expense';
		amount: string;
		description?: string;
		category_id?: string;
		subcategory_id?: string;
		subcategory_name?: string;
		transaction_date: string;
	}
) {
	return request<Transaction>(`/api/v1/transactions/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function deleteTransaction(id: string) {
	return request<void>(`/api/v1/transactions/${id}`, { method: 'DELETE' });
}

export function listRecurringOperations() {
	return request<RecurringOperation[]>('/api/v1/recurring-operations');
}

export function createRecurringOperation(payload: {
	type: 'income' | 'expense';
	amount: string;
	description?: string;
	account_id: string;
	category_id: string;
	subcategory_id?: string;
	period: 'week' | 'two_weeks' | 'month' | 'year';
	weekday?: number;
	day_of_month?: number;
	start_date: string;
	time_local?: string;
	active?: boolean;
}) {
	return request<RecurringOperation>('/api/v1/recurring-operations', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function updateRecurringOperation(
	id: string,
	payload: {
		type: 'income' | 'expense';
		amount: string;
		description?: string;
		account_id: string;
		category_id: string;
		subcategory_id?: string;
		period: 'week' | 'two_weeks' | 'month' | 'year';
		weekday?: number;
		day_of_month?: number;
		start_date: string;
		time_local?: string;
		active?: boolean;
	}
) {
	return request<RecurringOperation>(`/api/v1/recurring-operations/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function deleteRecurringOperation(id: string) {
	return request<void>(`/api/v1/recurring-operations/${id}`, { method: 'DELETE' });
}

export function createTransfer(payload: {
	from_account_id: string;
	to_account_id: string;
	amount: string;
	commission?: string;
	description?: string;
	transaction_date: string;
}) {
	return request<Transfer>('/api/v1/transfers', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function updateTransfer(
	groupId: string,
	payload: {
		from_account_id: string;
		to_account_id: string;
		amount: string;
		commission?: string;
		description?: string;
		transaction_date: string;
	}
) {
	return request<Transfer>(`/api/v1/transfers/${groupId}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function deleteTransfer(groupId: string) {
	return request<void>(`/api/v1/transfers/${groupId}`, { method: 'DELETE' });
}

export function getDashboard() {
	return request<Dashboard>('/api/v1/dashboard');
}

export function getStatsSummary(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<StatsSummary>(`/api/v1/stats/summary${q}`);
}

export function getStatsByCategory(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<{ items: StatsCategoryItem[] }>(`/api/v1/stats/by-category${q}`);
}

export function getStatsBySubcategory(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<{ items: StatsSubcategoryItem[] }>(`/api/v1/stats/by-subcategory${q}`);
}

export function getStatsByPeriod(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<{ items: StatsPeriodItem[] }>(`/api/v1/stats/by-period${q}`);
}

export function searchStats(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<TransactionList>(`/api/v1/stats/search${q}`);
}

export function getStatsContext(params?: Record<string, string>) {
	const q = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<StatsContext>(`/api/v1/stats/context${q}`);
}

export function getAccountsSummary() {
	return request<AccountsSummary>('/api/v1/accounts/summary');
}

export function getAccountBalance(id: string) {
	return request<AccountBalanceSummary>(`/api/v1/accounts/${id}/balance`);
}

export function listDebtors() {
	return cachedGet('/api/v1/debtors', () => request<Debtor[]>('/api/v1/debtors'));
}

export function getDebtor(id: string) {
	return request<DebtorDetail>(`/api/v1/debtors/${id}`);
}

export function createDebtor(name: string) {
	return request<Debtor>('/api/v1/debtors', {
		method: 'POST',
		body: JSON.stringify({ name })
	}).then((debtor) => {
		invalidateApiCache('/api/v1/debtors');
		return debtor;
	});
}

export function listDebts(params?: { settled?: string }) {
	const q = params?.settled !== undefined ? `?settled=${params.settled}` : '';
	return request<Debt[]>(`/api/v1/debts${q}`);
}

export function getDebt(id: string) {
	return request<Debt>(`/api/v1/debts/${id}`);
}

export function createDebt(payload: {
	debtor_id?: string;
	debtor_name?: string;
	direction: 'lent' | 'borrowed';
	amount: string;
	debt_date: string;
	due_date: string;
	affects_balance: boolean;
	description?: string;
	account_id?: string;
}) {
	return request<Debt>('/api/v1/debts', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function settleDebt(
	id: string,
	payload: {
		amount?: string;
		settled_at: string;
		affects_balance: boolean;
		account_id?: string;
	}
) {
	return request<Debt>(`/api/v1/debts/${id}/settle`, {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function deleteDebt(id: string) {
	return request<void>(`/api/v1/debts/${id}`, { method: 'DELETE' });
}

export function getDebtsSummary() {
	return request<DebtsSummary>('/api/v1/debts/summary');
}

export type CreditPayment = {
	id: string;
	credit_id: string;
	transaction_id: string | null;
	transaction_kind?: string | null;
	amount: number;
	amount_display: string;
	payment_date: string;
	kind: 'scheduled' | 'early' | 'auto' | 'retroactive';
	is_applied: boolean;
	exclude_from_stats: boolean;
	created_at: string;
};

export type SchedulePreviewEntry = {
	payment_date: string;
	amount: number;
	amount_display?: string;
};

export type Credit = {
	id: string;
	name: string | null;
	credit_kind?: 'consumer' | 'mortgage';
	principal_amount: number;
	principal_amount_display: string;
	property_price?: number | null;
	property_price_display?: string | null;
	down_payment?: number;
	down_payment_display?: string;
	down_payment_affects_balance?: boolean;
	down_payment_transaction_id?: string | null;
	issue_date: string;
	term_months: number;
	interest_rate: number;
	payment_interval: 'month' | 'week' | 'two_weeks' | 'manual';
	paid_amount: number;
	paid_amount_display: string;
	monthly_payment: number;
	monthly_payment_display: string;
	remaining_amount: number;
	remaining_amount_display: string;
	debit_account_id: string;
	debit_account_name: string;
	debit_time_local?: string | null;
	bank_id?: string | null;
	bank_name?: string | null;
	bank_id_locked?: boolean;
	added_retroactively: boolean;
	recorded_at: string;
	status: 'active' | 'closed';
	closed_at: string | null;
	is_installment: boolean;
	next_payment_date?: string | null;
	next_payment_amount?: number | null;
	schedule?: CreditPayment[];
	created_at: string;
	updated_at: string;
};

export function listCredits(params?: { status?: string }) {
	const q = params?.status ? `?status=${params.status}` : '';
	return request<Credit[]>(`/api/v1/credits${q}`);
}

export function getCredit(id: string) {
	return request<Credit>(`/api/v1/credits/${id}`);
}

export function createCredit(payload: Record<string, unknown>) {
	return request<Credit>('/api/v1/credits', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function updateCredit(id: string, payload: Record<string, unknown>) {
	return request<Credit>(`/api/v1/credits/${id}`, {
		method: 'PUT',
		body: JSON.stringify(payload)
	});
}

export function addCreditPayment(
	id: string,
	payload: { amount: string; payment_date: string; account_id?: string }
) {
	return request<Credit>(`/api/v1/credits/${id}/payments`, {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function deleteCreditPayment(creditId: string, paymentId: string) {
	return request<Credit>(`/api/v1/credits/${creditId}/payments/${paymentId}`, { method: 'DELETE' });
}

export function updateCreditSchedule(
	creditId: string,
	payload: { payments: { id: string; amount: string }[] }
) {
	return request<Credit>(`/api/v1/credits/${creditId}/schedule`, {
		method: 'PATCH',
		body: JSON.stringify(payload)
	});
}

export function completeCredit(
	id: string,
	payload: { affects_balance: boolean; payment_date: string }
) {
	return request<Credit>(`/api/v1/credits/${id}/close`, {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export function deleteCredit(id: string, mode: 'cascade' | 'keep_transactions') {
	return request<void>(`/api/v1/credits/${id}?mode=${mode}`, { method: 'DELETE' });
}

export function previewCreditSchedule(payload: Record<string, unknown>) {
	return request<{
		schedule_preview: SchedulePreviewEntry[];
		calculated_monthly_payment: number;
		calculated_monthly_payment_display: string;
	}>('/api/v1/credits/schedule/preview', {
		method: 'POST',
		body: JSON.stringify(payload)
	});
}

export type ImportPreviewItem = {
	row: number;
	action: 'create_expense' | 'create_income' | 'create_transfer';
	account?: string;
	to_account?: string;
	amount: number;
	category?: string;
	subcategory?: string;
	date: string;
	description?: string;
};

export type ImportRowError = {
	row: number;
	message: string;
};

export type AccountMappingSuggestion = {
	file_name: string;
	mode: 'create' | 'existing';
	account_id?: string;
	account_name?: string;
	account_type?: 'cash' | 'bank';
	bank_id?: string;
};

export type CategoryMappingSuggestion = {
	file_name: string;
	type: 'expense' | 'income';
	mode: 'create' | 'existing';
	category_id?: string;
	category_name?: string;
};

export type CategoryMapEntry = {
	mode: 'create' | 'existing';
	category_id?: string;
};

export type SubcategoryMappingSuggestion = {
	file_category: string;
	file_subcategory: string;
	type: 'expense' | 'income';
	mode: 'create' | 'existing';
	subcategory_id?: string;
	subcategory_name?: string;
};

export type SubcategoryMapEntry = {
	mode: 'create' | 'existing';
	subcategory_id?: string;
};

export type ImportReport = {
	total_rows: number;
	processed_rows?: number;
	valid_rows: number;
	skipped_duplicates: number;
	created_transactions?: number;
	errors: ImportRowError[];
	logs?: string[];
	preview: ImportPreviewItem[];
	accounts_to_create: string[];
	account_mappings: AccountMappingSuggestion[];
	category_mappings: CategoryMappingSuggestion[];
	subcategory_mappings: SubcategoryMappingSuggestion[];
	categories_to_create: string[];
};

export type ImportJob = {
	id: string;
	filename: string;
	status: 'queued' | 'running' | 'done' | 'failed';
	error_message?: string;
	report?: ImportReport;
	created_at: string;
	started_at?: string;
	finished_at?: string;
};

export type AccountMapEntry = {
	mode: 'create' | 'existing';
	account_id?: string;
	account_type?: 'cash' | 'bank';
	bank_id?: string;
};

export type ImportOptions = {
	file: File;
	preset?: 'cubux' | 'custom';
	deduplicate?: boolean;
	confirm?: boolean;
	column_map?: Record<string, string>;
	account_map?: Record<string, AccountMapEntry>;
	category_map?: Record<string, CategoryMapEntry>;
	subcategory_map?: Record<string, SubcategoryMapEntry>;
	auto_subcategory?: boolean;
	idempotencyKey?: string;
};

function importFormData(opts: ImportOptions): FormData {
	const form = new FormData();
	form.append('file', opts.file);
	form.append('preset', opts.preset ?? 'cubux');
	form.append('deduplicate', String(opts.deduplicate ?? true));
	form.append('auto_subcategory', String(opts.auto_subcategory ?? true));
	if (opts.confirm) form.append('confirm', 'true');
	if (opts.column_map) form.append('column_map', JSON.stringify(opts.column_map));
	if (opts.account_map) form.append('account_map', JSON.stringify(opts.account_map));
	if (opts.category_map) form.append('category_map', JSON.stringify(opts.category_map));
	if (opts.subcategory_map) form.append('subcategory_map', JSON.stringify(opts.subcategory_map));
	return form;
}

export function previewImport(opts: ImportOptions) {
	return request<ImportReport>('/api/v1/import/preview', {
		method: 'POST',
		body: importFormData(opts)
	});
}

export function peekImportHeaders(file: File) {
	const form = new FormData();
	form.append('file', file);
	return request<{ headers: string[] }>('/api/v1/import/headers', {
		method: 'POST',
		body: form
	});
}

export function commitImport(opts: ImportOptions) {
	const headers: Record<string, string> = {};
	if (opts.idempotencyKey) headers['Idempotency-Key'] = opts.idempotencyKey;
	return request<ImportReport>('/api/v1/import', {
		method: 'POST',
		headers,
		body: importFormData({ ...opts, confirm: true })
	});
}

export function createImportJob(opts: ImportOptions) {
	const headers: Record<string, string> = {};
	if (opts.idempotencyKey) headers['Idempotency-Key'] = opts.idempotencyKey;
	return request<ImportJob>('/api/v1/import/jobs', {
		method: 'POST',
		headers,
		body: importFormData({ ...opts, confirm: true })
	});
}

export function getImportJob(id: string) {
	return request<ImportJob>(`/api/v1/import/jobs/${encodeURIComponent(id)}`);
}

export function exportCSVUrl(params: {
	from?: string;
	to?: string;
	account_id?: string;
	category_id?: string;
}) {
	const q = new URLSearchParams();
	if (params.from) q.set('from', params.from);
	if (params.to) q.set('to', params.to);
	if (params.account_id) q.set('account_id', params.account_id);
	if (params.category_id) q.set('category_id', params.category_id);
	const qs = q.toString();
	return `/api/v1/export${qs ? `?${qs}` : ''}`;
}
