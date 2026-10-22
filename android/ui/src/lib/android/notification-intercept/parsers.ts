import {
	bankIdForPackage,
	inferBankIdFromText,
	isAllowlistedPackage,
	isMarketplacePackage,
	resolveRawBankNotification,
	walletIdForPackage
} from './banks';
import type { ParsedPurchase, RawBankNotification } from './types';

// Avoid \\b with Cyrillic — JS word boundaries are ASCII-oriented.
// Do not IGNORE «перевод по сбп» — solo outgoing → expense; pair with «от» → transfer.
const IGNORE_RE =
	/(код|code|otp|пароль|вход|войдите|баланс|остаток|отказ|отклонен|отклонён|заблокир)/i;

/**
 * Incoming / credit → income. Direction marker «от», not channel «по СБП».
 * «перевод по …» alone is outgoing (see OUTGOING_TRANSFER_RE → purchase).
 */
const INCOME_RE =
	/(выплат|процент|начислен|кэшб[еэ]к|кешб[еэ]к|cashback|поступлен|поступил[аи]?|зачисл|пополнен|входящ|перевод[\s\S]{0,120}?от|вам\s+перевел|зарплат|стипенд|доход\b|заработн)/i;

/** Outgoing transfer («перевод по СБП/номеру…») → expense draft unless «от» makes it income. */
const OUTGOING_TRANSFER_RE = /перевод\s+по/i;

/** Marketplace shell: wallet/bank chrome in title. */
const MARKETPLACE_WALLET_TITLE_RE = /(кошел[её]к|банк|ozon\s*card|wb\s*банк)/i;

/**
 * Marketplace shell: explicit bank-op key (not «на покупку одежды» / promo copy).
 * Use «покупка» (nom.), not «покупк» — otherwise «Потратьте на покупку…» passes the gate.
 */
const MARKETPLACE_OP_RE =
	/(перевод\s+по|перевод\s+от|списал|оплат|покупка|поступлен|поступил|пополнен|выплат|зачисл|начислен|кэшб[еэ]к|кешб[еэ]к|cashback)/i;

/** Marketplace marketing / delivery — never a draft even if text has ₽. */
const MARKETPLACE_MARKETING_RE =
	/(у\s+вас\s+на\s+сч[её]те|потратьте|подобрать|купить\s+сейчас|вау-цен|пункт[еа]?\s+выдач|доставил|можно\s+забирать|личное\s+приглашение|\d+\s*%|выгодой\s+до)/i;

/** True when an amount is followed by a currency marker (bare `р` ok; avoids `р` in names). */
const HAS_CURRENCY_AMOUNT_RE = /\d(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.?|RUB|rub)/i;

const CANCEL_RE =
	/(отмена\s+покупки|отмен[аы]\s+операц|отменена?\s+покупк|возврат\s+средств|возврат\s+покупк|purchase\s+cancel|canceled?\s+purchase|refund)/i;

const PURCHASE_HINT_RE =
	/(покупк|оплат|списал|списан|трата|платёж|платеж|purchase|payment|spent|оплата|\bpaid\b)/i;

/** Titles that are bank/card chrome, not a merchant (Yandex cancel uses «Карта Пэй»). */
const GENERIC_TITLE_RE =
	/^(карта(\s+пэй)?|card(\s+pay)?|пэй|pay|яндекс(\s+пэй)?|yandex(\s+pay)?|тинькофф|т-?банк|сбер(банк)?|втб(\s+онлайн)?|vtb(\s+online)?|wb\s*(банк|кошел[её]к)|кошел[её]к|mir(\s*pay)?|мир(\s*пэй|\s*pay)?|samsung(\s*(pay|wallet))?|google(\s*(pay|wallet))?|сбпэй|сбп|huawei(\s*(pay|wallet))?|mi(\s*pay)?|xiaomi(\s*(pay|wallet))?|юmoney|юмани|yoomoney)$/i;

const AMOUNT_RE =
	/(?:^|[^\d])(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.?|RUB|rub)?(?!\d)/i;

/** *XXXX, карта *XXXX, СЧЁТ5159 / Счет*1736 (optional * / bullet between счёт and digits). */
const LAST4_RE = /(?:сч[её]т\s*[*⁎•∙●○∗]?|(?:[*⁎•∙●○∗]|карты?\s*|карта\s*|card\s*))(\d{4})\b/i;

/** Cyrillic-safe token bounds (JS \b is ASCII-only). */
const TOKEN_BOUND_L = '(?:^|[\\s.,:;!\\-–—])';
const TOKEN_BOUND_R = '(?=[\\s.,:;!\\-–—]|$)';

const PURCHASE_WORD_RE = new RegExp(
	`${TOKEN_BOUND_L}(покупка|оплата|оплат|списание|списан|трата|платёж|платеж|purchase|payment|spent|paid)${TOKEN_BOUND_R}`,
	'gi'
);

/** Amount + currency as one unit; no trailing \b after Cyrillic `р`. */
const AMOUNT_WITH_CURRENCY_RE =
	/(?:^|[^\d])(\d{1,3}(?:[ \u00a0]\d{3})+(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.?|RUB)(?=[\s.,:;!]|\b|$)/gi;

/** Orphan currency left after stripping bare digits of the purchase amount. */
const ORPHAN_CURRENCY_RE = /(?:^|[\s.,:;!\-–—])(?:₽|руб\.?|р\.?|RUB)(?=[\s.,:;!\-–—]|$)/gi;

/** Card / account chrome words — Cyrillic-safe (Latin \b ok for MIR/Visa). */
const CARD_CHROME_RE = new RegExp(
	`${TOKEN_BOUND_L}(карта|card|счёт|счет|на)${TOKEN_BOUND_R}|\\b(MIR|Visa|MasterCard)\\b`,
	'gi'
);

/** Tail from Баланс / Доступно (no \\b after Cyrillic — JS word bounds are ASCII-only). */
const BALANCE_TAIL_RE = /[\s.,:;!\-–—]+(?:баланс|доступно|available)[\s\S]*$/i;

/** T-Bank SMS: «Покупка, карта *2552. 56 RUB. STOLOVAYA. Доступно …» */
const TBANK_SMS_MERCHANT_RE =
	/(?:^|[^\d])(?:\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.?|RUB)\.?\s+([A-Za-zА-Яа-яЁё0-9 ._-]{2,60}?)\s*\.?\s*(?:доступно|available)/i;

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
	// Operation amount ≠ trailing «Доступно/Баланс …»; never treat «80%» as currency.
	const scoped = text.replace(BALANCE_TAIL_RE, ' ');
	const currencyFirst =
		scoped.match(
			/(?:₽|руб\.?|RUB)\s*(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)/i
		) ??
		scoped.match(
			/(\d{1,3}(?:[ \u00a0]\d{3})*(?:[.,]\d{1,2})?|\d+(?:[.,]\d{1,2})?)\s*(?:₽|руб\.?|р\.?|RUB)(?!\s*%)/i
		);
	if (!currencyFirst?.[1]) return null;
	return normalizeAmount(currencyFirst[1]);
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
	t = t.replace(
		/^(сбер|сбербанк|тинькофф|т-?банк|tinkoff|яндекс|mir\s*pay|мир\s*пэй|samsung\s*(pay|wallet)|google\s*(pay|wallet)|сбпэй|huawei\s*(pay|wallet)|mi\s*pay|юmoney|yoomoney)\s*[:.]?\s*/i,
		''
	);
	t = t.replace(CANCEL_RE, ' ');
	t = t.replace(INCOME_RE, ' ');
	t = t.replace(OUTGOING_TRANSFER_RE, ' ');
	t = t.replace(PURCHASE_WORD_RE, ' ').replace(/\s+/g, ' ').trim();
	// Prefer amount+currency as one unit; bare digits leave orphan `р`/`₽`.
	t = t.replace(AMOUNT_WITH_CURRENCY_RE, ' ');
	const amountAlt = amount.replace('.', '[,.]');
	t = t.replace(new RegExp(`(?:^|[^\\d])${amountAlt}(?!\\d)`, 'i'), ' ');
	t = t.replace(ORPHAN_CURRENCY_RE, ' ');
	// Strip card mask before leftover amounts — otherwise «2552» becomes «255» + stray digit.
	t = t.replace(LAST4_RE, ' ');
	t = t.replace(AMOUNT_WITH_CURRENCY_RE, ' ');
	t = t.replace(CARD_CHROME_RE, ' ');
	t = t.replace(BALANCE_TAIL_RE, ' ');
	t = t.replace(/[•*⁎∙●○∗]+/g, ' ');
	t = t.replace(/\s+/g, ' ').trim();
	const inMatch = t.match(/(?:^|[\s.,:;])в\s+([A-Za-zА-Яа-яЁё0-9 ._-]{2,60})/i);
	if (inMatch?.[1]) {
		t = inMatch[1].trim();
	}
	t = t.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '').trim();
	if (t.length > 80) t = t.slice(0, 80).trim();
	return t;
}

/**
 * Counterparty: «от …» for income; next line after «Перевод по СБП …» for outgoing (WB).
 * No \\b — JS word boundaries are ASCII-only (same as other Cyrillic regexes here).
 */
function extractTransferCounterparty(
	text: string,
	kind: 'income' | 'purchase'
): string | undefined {
	if (kind === 'income') {
		const m = text.match(
			/(?:^|[\s.,:;])от\s+([A-Za-zА-Яа-яЁё][^₽\n]*?)(?=\s*(?:\+?\d[\d \u00a0]*(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.?|RUB)|баланс|доступно|available|\d{1,2}:\d{2}|$))/i
		);
		if (m?.[1]) {
			const name = m[1]
				.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '')
				.replace(/\s+/g, ' ')
				.trim();
			if (name) return name;
		}
		return undefined;
	}
	const wb = text.match(
		/перевод\s+по\s+сбп[^\n]*\n+\s*([A-Za-zА-Яа-яЁё][^\n₽]{1,60}?)(?=\n+\s*(?:доступно|баланс|available)|$)/i
	);
	if (!wb?.[1]) return undefined;
	const name = wb[1]
		.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '')
		.replace(/\s+/g, ' ')
		.trim();
	return name || undefined;
}

/** Marketplace packages: only wallet/bank title or an explicit operation key. */
function passesMarketplaceGate(raw: RawBankNotification, text: string): boolean {
	if (!isMarketplacePackage(raw.packageName)) return true;
	if (MARKETPLACE_MARKETING_RE.test(text)) return false;
	if (MARKETPLACE_WALLET_TITLE_RE.test(raw.title)) return true;
	return MARKETPLACE_OP_RE.test(text);
}

/** Short label for income when there is no counterparty (e.g. «Выплата процентов»). */
function extractIncomeLabel(raw: RawBankNotification): string {
	const line = [raw.text, raw.bigText, raw.title]
		.filter(Boolean)
		.join(' ')
		.replace(/\s+/g, ' ')
		.trim();
	const beforeAmount = line.split(/(?:\d[\d \u00a0]*(?:[.,]\d{1,2})?\s*(?:₽|руб\.?|р\.?|RUB))/i)[0];
	let t = (beforeAmount || line).trim();
	t = t.replace(GENERIC_TITLE_RE, ' ').replace(/\s+/g, ' ').trim();
	t = t.replace(/^[\s.,:;!\-–—]+|[\s.,:;!\-–—]+$/g, '').trim();
	if (t.length > 80) t = t.slice(0, 80).trim();
	return t;
}

function looksLikeIncome(text: string): boolean {
	if (CANCEL_RE.test(text)) return false;
	if (!INCOME_RE.test(text)) return false;
	// «перевод от» wins over purchase words / «перевод по СБП» in the same SMS.
	if (PURCHASE_HINT_RE.test(text) && !/перевод\s+от|(?:^|[\s.,:;])от\s+/i.test(text)) {
		return false;
	}
	return HAS_CURRENCY_AMOUNT_RE.test(text) && AMOUNT_RE.test(text);
}

function looksLikePurchase(text: string): boolean {
	if (CANCEL_RE.test(text)) return false;
	// Incoming «от» → income, even when the channel is «по СБП».
	if (INCOME_RE.test(text)) return false;
	if (PURCHASE_HINT_RE.test(text) || OUTGOING_TRANSFER_RE.test(text)) return true;
	if (IGNORE_RE.test(text)) return false;
	return HAS_CURRENCY_AMOUNT_RE.test(text) && AMOUNT_RE.test(text);
}

function hashRaw(raw: RawBankNotification): string {
	return (
		raw.dedupeKey || `${raw.packageName}|${raw.postedAt}|${raw.title}|${raw.text}|${raw.bigText}`
	);
}

export function parseBankNotification(raw: RawBankNotification): ParsedPurchase | null {
	const resolved = resolveRawBankNotification(raw);
	if (!isAllowlistedPackage(resolved.packageName)) return null;
	const packageBankId = bankIdForPackage(resolved.packageName);
	const walletId = walletIdForPackage(resolved.packageName);

	const text = combinedText(resolved);
	if (!text.trim()) return null;
	if (!passesMarketplaceGate(resolved, text)) return null;

	const isCancel = CANCEL_RE.test(text);
	const isIncome = !isCancel && looksLikeIncome(text);
	if (!isCancel && !isIncome && !looksLikePurchase(text)) return null;

	const amount = extractAmount(text);
	if (!amount) return null;

	const last4 = extractLast4(text);
	const bankId = packageBankId ?? inferBankIdFromText(text) ?? walletId;
	if (!bankId) return null;
	let merchantText = extractTransferCounterparty(text, isIncome ? 'income' : 'purchase') ?? '';
	if (!merchantText) {
		merchantText = extractMerchant(resolved, amount);
	}
	if (isIncome && !merchantText) {
		merchantText = extractIncomeLabel(resolved);
	}
	const occurredAt = new Date(resolved.postedAt > 0 ? resolved.postedAt : Date.now()).toISOString();

	return {
		bankId,
		packageName: resolved.packageName,
		amount,
		occurredAt,
		merchantText,
		last4,
		rawHash: hashRaw(resolved),
		kind: isCancel ? 'cancel' : isIncome ? 'income' : 'purchase'
	};
}
