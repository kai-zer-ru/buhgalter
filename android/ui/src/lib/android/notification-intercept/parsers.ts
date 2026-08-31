import { bankIdForPackage } from './banks';
import type { ParsedPurchase, RawBankNotification } from './types';

// Avoid \\b with Cyrillic — JS word boundaries are ASCII-oriented.
const IGNORE_RE =
	/(код|code|otp|пароль|вход|войдите|баланс|остаток|отказ|отклонен|отклонён|заблокир|перевод\s+на\s+карт)/i;

/** Incoming / credit pushes → income drafts (not expense). */
const INCOME_RE =
	/(выплат|процент|начислен|кэшб[еэ]к|кешб[еэ]к|cashback|поступило|поступил[аи]?|зачисл|пополнен|входящ|перевод\s+от|вам\s+перевел|зарплат|стипенд|доход\b|заработн)/i;

const CANCEL_RE =
	/(отмена\s+покупки|отмен[аы]\s+операц|отменена?\s+покупк|возврат\s+средств|возврат\s+покупк|purchase\s+cancel|canceled?\s+purchase|refund)/i;

const PURCHASE_HINT_RE = /(покупк|оплат|списан|трата|платёж|платеж|purchase|payment|spent|оплата)/i;

/** Titles that are bank/card chrome, not a merchant (Yandex cancel uses «Карта Пэй»). */
const GENERIC_TITLE_RE =
	/^(карта(\s+пэй)?|card(\s+pay)?|пэй|pay|яндекс(\s+пэй)?|yandex(\s+pay)?|тинькофф|т-?банк|сбер(банк)?|wb\s*банк)$/i;

const AMOUNT_RE =
	/(?:^|[^\d])(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.|RUB|rub)?(?!\d)/i;

const LAST4_RE = /(?:\*|⁎|•|∙|●|○|∗|карты?\s*|карта\s*|card\s*)(\d{4})\b/i;

const PURCHASE_WORD_RE =
	/\b(покупка|оплата|оплат|списание|списан|трата|платёж|платеж|purchase|payment|spent)\b/gi;

/** T-Bank SMS: «Покупка, карта *2552. 56 RUB. STOLOVAYA. Доступно …» */
const TBANK_SMS_MERCHANT_RE =
	/(?:^|[^\d])(?:\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.|RUB)\.?\s+([A-Za-zА-Яа-яЁё0-9 ._-]{2,60}?)\s*\.?\s*(?:доступно|available)/i;

function combinedText(raw: RawBankNotification): string {
	// SMS title is the sender address — exclude from amount/kind heuristics.
	if (raw.channel === 'sms') {
		return [raw.text, raw.bigText].filter(Boolean).join('\n');
	}
	return [raw.title, raw.text, raw.bigText].filter(Boolean).join('\n');
}

function normalizeAmount(raw: string): string | null {
	const cleaned = raw.replace(/[\s\u00a0]/g, '').replace(',', '.');
	const n = Number(cleaned);
	if (!Number.isFinite(n) || n <= 0) return null;
	return n.toFixed(2);
}

function extractLast4(text: string): string | undefined {
	const m = text.match(LAST4_RE);
	return m?.[1];
}

function extractAmount(text: string): string | null {
	const currencyFirst =
		text.match(
			/(?:₽|руб\.?|RUB)\s*(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)/i
		) ??
		text.match(
			/(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.|RUB)/i
		);
	if (currencyFirst?.[1]) {
		const a = normalizeAmount(currencyFirst[1]);
		if (a) return a;
	}
	const m = text.match(AMOUNT_RE);
	if (!m?.[1]) return null;
	return normalizeAmount(m[1]);
}

/**
 * Merchant / counterparty label: prefer notification title when it looks like a store name
 * (Yandex Pay puts the shop in EXTRA_TITLE and purchase details in EXTRA_TEXT).
 * Cancel pushes often use a generic title («Карта Пэй») and put the shop in the body.
 */
function extractMerchant(raw: RawBankNotification, amount: string): string {
	if (raw.channel === 'sms') {
		const body = [raw.text, raw.bigText].filter(Boolean).join(' ').replace(/\s+/g, ' ').trim();
		const tbank = body.match(TBANK_SMS_MERCHANT_RE);
		if (tbank?.[1]) {
			return tbank[1].replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '').trim();
		}
	}

	const title = raw.title.trim();
	// SMS title is the originator (900 / T-Bank), never a shop name.
	if (
		raw.channel !== 'sms' &&
		title &&
		!GENERIC_TITLE_RE.test(title) &&
		!PURCHASE_HINT_RE.test(title) &&
		!CANCEL_RE.test(title) &&
		!INCOME_RE.test(title) &&
		!IGNORE_RE.test(title) &&
		title.length <= 80
	) {
		if (!/^\d/.test(title) && !/(₽|RUB|руб)/i.test(title)) {
			return title;
		}
	}

	let t = [raw.text, raw.bigText].filter(Boolean).join(' ').replace(/\s+/g, ' ').trim();
	t = t.replace(/^(сбер|сбербанк|тинькофф|т-?банк|tinkoff|яндекс)\s*[:.]?\s*/i, '');
	t = t.replace(CANCEL_RE, ' ');
	t = t.replace(INCOME_RE, ' ');
	t = t.replace(PURCHASE_WORD_RE, ' ').replace(/\s+/g, ' ').trim();
	const amountAlt = amount.replace('.', '[,.]');
	t = t.replace(new RegExp(`\\b${amountAlt}\\b`, 'i'), ' ');
	// Strip card mask before amounts — otherwise «2552» becomes «255» + stray digit.
	t = t.replace(LAST4_RE, ' ');
	t = t.replace(/\b\d{1,3}(?:[ \u00a0]\d{3})+(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.|RUB)\b/gi, ' ');
	t = t.replace(/\b\d+(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.|RUB)\b/gi, ' ');
	t = t.replace(/\b(карта|card|счёт|счет|доступно|на|MIR|Visa|MasterCard)\b/gi, ' ');
	t = t.replace(/[•*⁎∙●○∗]+/g, ' ');
	t = t.replace(/\s+/g, ' ').trim();
	const inMatch = t.match(/\bв\s+([A-Za-zА-Яа-яЁё0-9 ._-]{2,60})/i);
	if (inMatch?.[1]) {
		t = inMatch[1].trim();
	}
	t = t.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '').trim();
	if (t.length > 80) t = t.slice(0, 80).trim();
	return t;
}

/** Short label for income when there is no counterparty (e.g. «Выплата процентов»). */
function extractIncomeLabel(raw: RawBankNotification): string {
	const line = [raw.text, raw.bigText, raw.title]
		.filter(Boolean)
		.join(' ')
		.replace(/\s+/g, ' ')
		.trim();
	const beforeAmount = line.split(/(?:\d[\d \u00a0]*(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.|RUB))/i)[0];
	let t = (beforeAmount || line).trim();
	t = t.replace(GENERIC_TITLE_RE, ' ').replace(/\s+/g, ' ').trim();
	t = t.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '').trim();
	if (t.length > 80) t = t.slice(0, 80).trim();
	return t;
}

function looksLikeIncome(text: string): boolean {
	if (CANCEL_RE.test(text)) return false;
	if (!INCOME_RE.test(text)) return false;
	if (PURCHASE_HINT_RE.test(text)) return false;
	return /(?:₽|руб\.?|RUB)/i.test(text) && AMOUNT_RE.test(text);
}

function looksLikePurchase(text: string): boolean {
	if (CANCEL_RE.test(text)) return false;
	if (INCOME_RE.test(text)) return false;
	if (PURCHASE_HINT_RE.test(text)) return true;
	if (IGNORE_RE.test(text)) return false;
	return /(?:₽|руб\.?|RUB)/i.test(text) && AMOUNT_RE.test(text);
}

function hashRaw(raw: RawBankNotification): string {
	return (
		raw.dedupeKey || `${raw.packageName}|${raw.postedAt}|${raw.title}|${raw.text}|${raw.bigText}`
	);
}

export function parseBankNotification(raw: RawBankNotification): ParsedPurchase | null {
	const bankId = bankIdForPackage(raw.packageName);
	if (!bankId) return null;

	const text = combinedText(raw);
	if (!text.trim()) return null;

	const isCancel = CANCEL_RE.test(text);
	const isIncome = !isCancel && looksLikeIncome(text);
	if (!isCancel && !isIncome && !looksLikePurchase(text)) return null;

	const amount = extractAmount(text);
	if (!amount) return null;

	const last4 = extractLast4(text);
	let merchantText = extractMerchant(raw, amount);
	if (isIncome && !merchantText) {
		merchantText = extractIncomeLabel(raw);
	}
	const occurredAt = new Date(raw.postedAt > 0 ? raw.postedAt : Date.now()).toISOString();

	return {
		bankId,
		packageName: raw.packageName,
		amount,
		occurredAt,
		merchantText,
		last4,
		rawHash: hashRaw(raw),
		kind: isCancel ? 'cancel' : isIncome ? 'income' : 'purchase'
	};
}
