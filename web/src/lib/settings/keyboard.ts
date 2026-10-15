import type { NotificationInlineKeyboard, NotificationKeyboardButton } from '$lib/api/client';
import { moveId } from '$lib/drag-reorder';

export type KeyboardProvider = 'telegram' | 'max';

export type EditableKeyboardButton = NotificationKeyboardButton & { id: string };

export type EditableKeyboard = {
	rows: EditableKeyboardButton[][];
};

const maxTGCallbackDataBytes = 64;
const maxTGButtonTextLen = 64;
const maxTGURLLen = 2048;
const maxTGButtons = 100;
const maxTGButtonsPerRow = 8;

const maxMAXButtons = 210;
const maxMAXRows = 30;
const maxMAXButtonsPerRow = 7;
const maxMAXSpecialPerRow = 3;
const maxMAXURLLen = 2048;
const maxMAXPayloadLen = 1024;

export const TELEGRAM_BUTTON_TYPES = [
	'url',
	'callback_data',
	'web_app',
	'login_url',
	'switch_inline_query',
	'switch_inline_query_current_chat',
	'switch_inline_query_chosen_chat',
	'callback_game',
	'pay',
	'copy_text'
] as const;

export const MAX_BUTTON_TYPES = [
	'callback',
	'link',
	'message',
	'open_app',
	'request_contact',
	'request_geo_location',
	'clipboard'
] as const;

export const INBOUND_ONLY_TYPES = new Set([
	'pay',
	'callback_game',
	'request_contact',
	'request_geo_location'
]);

let idSeq = 0;

export function nextKeyboardId(prefix = 'kb'): string {
	idSeq += 1;
	return `${prefix}-${idSeq}`;
}

export function emptyInlineKeyboard(): NotificationInlineKeyboard {
	return { rows: [] };
}

export function normalizeInlineKeyboard(
	kb: NotificationInlineKeyboard | null | undefined
): NotificationInlineKeyboard {
	if (!kb || !Array.isArray(kb.rows)) return emptyInlineKeyboard();
	return {
		rows: kb.rows.map((row) => (Array.isArray(row) ? row.map((b) => ({ ...b })) : []))
	};
}

export function toEditableKeyboard(
	kb: NotificationInlineKeyboard | null | undefined
): EditableKeyboard {
	const normalized = normalizeInlineKeyboard(kb);
	return {
		rows: normalized.rows.map((row) =>
			row.map((btn) => ({
				...btn,
				id: nextKeyboardId('btn')
			}))
		)
	};
}

export function toApiKeyboard(kb: EditableKeyboard): NotificationInlineKeyboard {
	return {
		rows: kb.rows
			.map((row) => row.map((btn) => stripEmptyButtonFields(btn)))
			.filter((row) => row.length > 0)
	};
}

function stripEmptyButtonFields(btn: EditableKeyboardButton): NotificationKeyboardButton {
	const out: NotificationKeyboardButton = { type: btn.type, text: btn.text };
	if (btn.url) out.url = btn.url;
	if (btn.callback_data) out.callback_data = btn.callback_data;
	if (btn.web_app?.url) out.web_app = { url: btn.web_app.url };
	if (btn.login_url?.url) {
		out.login_url = { url: btn.login_url.url };
		if (btn.login_url.forward_text) out.login_url.forward_text = btn.login_url.forward_text;
		if (btn.login_url.bot_username) out.login_url.bot_username = btn.login_url.bot_username;
		if (btn.login_url.request_write_access != null) {
			out.login_url.request_write_access = btn.login_url.request_write_access;
		}
	}
	if (btn.switch_inline_query != null) out.switch_inline_query = btn.switch_inline_query;
	if (btn.switch_inline_query_current_chat != null) {
		out.switch_inline_query_current_chat = btn.switch_inline_query_current_chat;
	}
	if (btn.switch_inline_query_chosen_chat) {
		out.switch_inline_query_chosen_chat = btn.switch_inline_query_chosen_chat;
	}
	if (btn.callback_game) out.callback_game = btn.callback_game;
	if (btn.pay) out.pay = true;
	if (btn.copy_text?.text) out.copy_text = { text: btn.copy_text.text };
	if (btn.payload) out.payload = btn.payload;
	return out;
}

export function buttonTypesFor(provider: KeyboardProvider): readonly string[] {
	return provider === 'telegram' ? TELEGRAM_BUTTON_TYPES : MAX_BUTTON_TYPES;
}

export function defaultButtonType(provider: KeyboardProvider): string {
	return provider === 'telegram' ? 'url' : 'link';
}

export function createDefaultButton(provider: KeyboardProvider): EditableKeyboardButton {
	const type = defaultButtonType(provider);
	return applyButtonTypeChange(
		{
			id: nextKeyboardId('btn'),
			type,
			text: ''
		},
		type
	);
}

export function applyButtonTypeChange(
	btn: EditableKeyboardButton,
	type: string
): EditableKeyboardButton {
	const next: EditableKeyboardButton = {
		id: btn.id,
		type,
		text: btn.text
	};
	switch (type) {
		case 'url':
		case 'link':
		case 'open_app':
			next.url = btn.url ?? '';
			break;
		case 'callback_data':
			next.callback_data = btn.callback_data ?? '';
			break;
		case 'web_app':
			next.web_app = { url: btn.web_app?.url ?? btn.url ?? '' };
			break;
		case 'login_url':
			next.login_url = { url: btn.login_url?.url ?? btn.url ?? '' };
			break;
		case 'copy_text':
			next.copy_text = { text: btn.copy_text?.text ?? '' };
			break;
		case 'callback':
		case 'clipboard':
		case 'message':
			next.payload = btn.payload ?? '';
			break;
		case 'switch_inline_query':
			next.switch_inline_query = btn.switch_inline_query ?? '';
			break;
		case 'switch_inline_query_current_chat':
			next.switch_inline_query_current_chat = btn.switch_inline_query_current_chat ?? '';
			break;
		case 'switch_inline_query_chosen_chat':
			next.switch_inline_query_chosen_chat = btn.switch_inline_query_chosen_chat ?? {};
			break;
		case 'callback_game':
			next.callback_game = btn.callback_game ?? {};
			break;
		case 'pay':
			next.pay = true;
			break;
		default:
			break;
	}
	return next;
}

/** One-click URL/link button preset for the keyboard builder. */
export type KeyboardLinkPreset = {
	/** Full i18n key for the chip («Пресет: …»). */
	chipKey: string;
	/** Full i18n key for the button text. */
	labelKey: string;
	url: string;
};

const PRESET_DEBT: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_debts',
	labelKey: 'settings.notifications.keyboard.preset.open_debts_label',
	url: '{debt_url}'
};
const PRESET_CREDIT: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_credit',
	labelKey: 'settings.notifications.keyboard.preset.open_credit_label',
	url: '{credit_url}'
};
const PRESET_SUBSCRIPTION: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_subscription',
	labelKey: 'settings.notifications.keyboard.preset.open_subscription_label',
	url: '{subscription_url}'
};
const PRESET_TRANSACTIONS: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_transactions',
	labelKey: 'settings.notifications.keyboard.preset.open_transactions_label',
	url: '{transaction_url}'
};
const PRESET_BUDGET: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_budget',
	labelKey: 'settings.notifications.keyboard.preset.open_budget_label',
	url: '{budget_url}'
};
const PRESET_ACCOUNT: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_account',
	labelKey: 'settings.notifications.keyboard.preset.open_account_label',
	url: '{account_url}'
};
const PRESET_RESET: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.reset_password',
	labelKey: 'settings.notifications.keyboard.preset.reset_password_label',
	url: '{reset_url}'
};
const PRESET_MODERATION: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.moderation',
	labelKey: 'settings.notifications.keyboard.preset.moderation_label',
	url: '{moderation_url}'
};
const PRESET_SETTINGS: KeyboardLinkPreset = {
	chipKey: 'settings.notifications.keyboard.preset.open_settings',
	labelKey: 'settings.notifications.keyboard.preset.open_settings_label',
	url: '{settings_url}'
};

/** Contextual link presets for a notification trigger; `null`/empty → global chips. */
export function linkPresetsForTrigger(
	triggerType: string | null | undefined
): KeyboardLinkPreset[] {
	switch (triggerType) {
		case 'debt_overdue':
		case 'debt_due_soon':
			return [PRESET_DEBT];
		case 'credit_payment':
			return [PRESET_CREDIT];
		case 'subscription_charge':
			return [PRESET_SUBSCRIPTION];
		case 'planned_operation':
			return [PRESET_TRANSACTIONS];
		case 'budget_threshold':
			return [PRESET_BUDGET];
		case 'auto_topup_disabled':
		case 'balance_shortfall':
			return [PRESET_ACCOUNT];
		case 'password_reset':
			return [PRESET_RESET];
		case 'user_registration':
			return [PRESET_MODERATION];
		case 'test':
			return [PRESET_SETTINGS];
		case null:
		case undefined:
		case '':
			return [PRESET_DEBT, PRESET_CREDIT, PRESET_TRANSACTIONS, PRESET_BUDGET, PRESET_SETTINGS];
		default:
			return [];
	}
}

export function createLinkPreset(
	provider: KeyboardProvider,
	preset: KeyboardLinkPreset
): EditableKeyboardButton {
	const type = provider === 'telegram' ? 'url' : 'link';
	return {
		id: nextKeyboardId('btn'),
		type,
		text: '',
		url: preset.url
	};
}

export function moveRowByIds(
	rows: EditableKeyboardButton[][],
	rowIds: string[],
	fromId: string,
	toId: string
): EditableKeyboardButton[][] | null {
	const reordered = moveId(rowIds, fromId, toId);
	if (!reordered) return null;
	return reordered.map((id) => {
		const idx = rowIds.indexOf(id);
		return rows[idx];
	});
}

/** Move a button before another (same or different row). */
export function moveButton(
	rows: EditableKeyboardButton[][],
	fromId: string,
	toId: string
): EditableKeyboardButton[][] | null {
	if (fromId === toId) return null;
	let fromRow = -1;
	let fromCol = -1;
	let toRow = -1;
	let toCol = -1;
	for (let r = 0; r < rows.length; r++) {
		for (let c = 0; c < rows[r].length; c++) {
			if (rows[r][c].id === fromId) {
				fromRow = r;
				fromCol = c;
			}
			if (rows[r][c].id === toId) {
				toRow = r;
				toCol = c;
			}
		}
	}
	if (fromRow < 0 || toRow < 0) return null;
	const next = rows.map((row) => [...row]);
	const [btn] = next[fromRow].splice(fromCol, 1);
	let insertCol = toCol;
	if (fromRow === toRow && fromCol < toCol) insertCol -= 1;
	next[toRow].splice(insertCol, 0, btn);
	return next;
}

export function addRow(
	rows: EditableKeyboardButton[][],
	provider: KeyboardProvider
): EditableKeyboardButton[][] {
	return [...rows, [createDefaultButton(provider)]];
}

export function addButtonToRow(
	rows: EditableKeyboardButton[][],
	rowIndex: number,
	provider: KeyboardProvider
): EditableKeyboardButton[][] | null {
	if (rowIndex < 0 || rowIndex >= rows.length) return null;
	const next = rows.map((row) => [...row]);
	next[rowIndex] = [...next[rowIndex], createDefaultButton(provider)];
	return next;
}

export function removeButton(
	rows: EditableKeyboardButton[][],
	buttonId: string
): EditableKeyboardButton[][] {
	return rows.map((row) => row.filter((b) => b.id !== buttonId)).filter((row) => row.length > 0);
}

export function removeRow(
	rows: EditableKeyboardButton[][],
	rowIndex: number
): EditableKeyboardButton[][] {
	if (rowIndex < 0 || rowIndex >= rows.length) return rows;
	return rows.filter((_, i) => i !== rowIndex);
}

export function updateButton(
	rows: EditableKeyboardButton[][],
	buttonId: string,
	patch: Partial<EditableKeyboardButton>
): EditableKeyboardButton[][] {
	return rows.map((row) =>
		row.map((b) => {
			if (b.id !== buttonId) return b;
			const merged = { ...b, ...patch, id: b.id };
			if (patch.type && patch.type !== b.type) {
				return applyButtonTypeChange(merged, patch.type);
			}
			return merged;
		})
	);
}

export type KeyboardValidationError = {
	code: string;
	values?: Record<string, string | number>;
};

function isMaxSpecialRowType(t: string): boolean {
	return (
		t === 'link' || t === 'open_app' || t === 'request_contact' || t === 'request_geo_location'
	);
}

export function validateEditableKeyboard(
	provider: KeyboardProvider,
	kb: EditableKeyboard
): KeyboardValidationError | null {
	const api = toApiKeyboard(kb);
	return validateInlineKeyboard(provider, api);
}

export function validateInlineKeyboard(
	provider: KeyboardProvider,
	kb: NotificationInlineKeyboard
): KeyboardValidationError | null {
	const rows = kb.rows ?? [];
	const buttonCount = rows.reduce((n, row) => n + row.length, 0);
	if (buttonCount === 0) return null;

	if (provider === 'telegram') {
		if (buttonCount > maxTGButtons) {
			return { code: 'max_buttons', values: { max: maxTGButtons } };
		}
		for (let i = 0; i < rows.length; i++) {
			if (rows[i].length > maxTGButtonsPerRow) {
				return { code: 'max_buttons_per_row', values: { max: maxTGButtonsPerRow, row: i + 1 } };
			}
			for (let j = 0; j < rows[i].length; j++) {
				const err = validateTelegramButton(rows[i][j]);
				if (err) {
					err.values = { ...err.values, row: i + 1, button: j + 1 };
					return err;
				}
			}
		}
		return null;
	}

	if (rows.length > maxMAXRows) {
		return { code: 'max_rows', values: { max: maxMAXRows } };
	}
	if (buttonCount > maxMAXButtons) {
		return { code: 'max_buttons', values: { max: maxMAXButtons } };
	}
	for (let i = 0; i < rows.length; i++) {
		if (rows[i].length > maxMAXButtonsPerRow) {
			return { code: 'max_buttons_per_row', values: { max: maxMAXButtonsPerRow, row: i + 1 } };
		}
		let special = 0;
		for (let j = 0; j < rows[i].length; j++) {
			const err = validateMaxButton(rows[i][j]);
			if (err) {
				err.values = { ...err.values, row: i + 1, button: j + 1 };
				return err;
			}
			if (isMaxSpecialRowType(rows[i][j].type)) special++;
		}
		if (special > 0 && rows[i].length > maxMAXSpecialPerRow) {
			return { code: 'max_special_per_row', values: { max: maxMAXSpecialPerRow, row: i + 1 } };
		}
	}
	return null;
}

function validateTelegramButton(btn: NotificationKeyboardButton): KeyboardValidationError | null {
	const text = btn.text?.trim() ?? '';
	if (!text) return { code: 'text_required' };
	if ([...text].length > maxTGButtonTextLen) {
		return { code: 'text_too_long', values: { max: maxTGButtonTextLen } };
	}
	if (!(TELEGRAM_BUTTON_TYPES as readonly string[]).includes(btn.type)) {
		return { code: 'unknown_type', values: { type: btn.type } };
	}
	switch (btn.type) {
		case 'url':
			if (!(btn.url ?? '').trim()) return { code: 'url_required' };
			if ((btn.url ?? '').length > maxTGURLLen) {
				return { code: 'url_too_long', values: { max: maxTGURLLen } };
			}
			break;
		case 'callback_data':
			if (!btn.callback_data) return { code: 'callback_data_required' };
			if (new TextEncoder().encode(btn.callback_data).length > maxTGCallbackDataBytes) {
				return { code: 'callback_data_too_long', values: { max: maxTGCallbackDataBytes } };
			}
			break;
		case 'web_app':
			if (!(btn.web_app?.url ?? '').trim()) return { code: 'web_app_url_required' };
			if ((btn.web_app?.url ?? '').length > maxTGURLLen) {
				return { code: 'url_too_long', values: { max: maxTGURLLen } };
			}
			break;
		case 'login_url':
			if (!(btn.login_url?.url ?? '').trim()) return { code: 'login_url_required' };
			if ((btn.login_url?.url ?? '').length > maxTGURLLen) {
				return { code: 'url_too_long', values: { max: maxTGURLLen } };
			}
			break;
		case 'copy_text':
			if (!(btn.copy_text?.text ?? '').trim()) return { code: 'copy_text_required' };
			break;
		default:
			break;
	}
	return null;
}

function validateMaxButton(btn: NotificationKeyboardButton): KeyboardValidationError | null {
	const text = btn.text?.trim() ?? '';
	if (!text) return { code: 'text_required' };
	if (!(MAX_BUTTON_TYPES as readonly string[]).includes(btn.type)) {
		return { code: 'unknown_type', values: { type: btn.type } };
	}
	switch (btn.type) {
		case 'link':
		case 'open_app':
			if (!(btn.url ?? '').trim()) return { code: 'url_required' };
			if ((btn.url ?? '').length > maxMAXURLLen) {
				return { code: 'url_too_long', values: { max: maxMAXURLLen } };
			}
			break;
		case 'callback':
		case 'clipboard':
			if (!(btn.payload ?? '').trim()) return { code: 'payload_required' };
			if ((btn.payload ?? '').length > maxMAXPayloadLen) {
				return { code: 'payload_too_long', values: { max: maxMAXPayloadLen } };
			}
			break;
		case 'message':
			if ((btn.payload ?? '').length > maxMAXPayloadLen) {
				return { code: 'payload_too_long', values: { max: maxMAXPayloadLen } };
			}
			break;
		default:
			break;
	}
	return null;
}

export function buttonCount(kb: NotificationInlineKeyboard | EditableKeyboard): number {
	return (kb.rows ?? []).reduce((n, row) => n + row.length, 0);
}
