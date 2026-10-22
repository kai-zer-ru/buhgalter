package ru.kai_zer.buhgalter;

import android.app.PendingIntent;
import android.content.Context;
import android.text.TextUtils;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;

import java.util.HashSet;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import ru.kai_zer.buhgalter.widgets.WidgetDeepLinks;

/**
 * Immediate shade notice when a bank push/SMS is queued, before JS parses a draft.
 * Replaced by real Accept/Reject notifications after {@code syncDraftNotifications}.
 *
 * <p>Gate mirrors JS {@code parseBankNotification}: no provisional for texts that would not
 * become a draft (marketing, bare %, balance-only). Amount requires a currency marker.
 */
final class DraftNotifyProvisional {
    private static final String PREFS = "buhgalter.draft_notify.provisional.v1";
    private static final String KEY_IDS = "ids";

    /** Keep in sync with JS {@code MARKETPLACE_PACKAGES}. */
    static final Set<String> MARKETPLACE_PACKAGES =
            Set.of("com.wildberries.ru", "ru.ozon.app.android");

    /** Amount + currency only (optional currency caused 80% → 80 ₽). */
    private static final Pattern AMOUNT_WITH_CURRENCY =
            Pattern.compile(
                    "(?<!\\d)(\\d{1,3}(?:[\\s\\u00a0]\\d{3})*(?:[.,]\\d{1,2})?|\\d+(?:[.,]\\d{1,2})?)"
                            + "\\s*(?:₽|RUB|руб\\.?|р\\.?)(?!\\s*%)",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private static final Pattern BALANCE_TAIL =
            Pattern.compile(
                    "[\\s.,:;!\\-–—]+(?:баланс|доступно|available)[\\s\\S]*$",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private static final Pattern IGNORE_NOISE =
            Pattern.compile(
                    "(код|code|otp|пароль|вход|войдите|отказ|отклонен|отклонён|заблокир)",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private static final Pattern BALANCE_ONLY =
            Pattern.compile(
                    "(?:^|[\\s.,:;])(?:баланс|остаток)(?:$|[\\s.,:;:])",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    /** Marketplace: bank-op keys only. «покупка» not «покупк» (avoids «на покупку…»). */
    private static final Pattern OP_HINT =
            Pattern.compile(
                    "(покупка|оплат|списан|списал|трата|платёж|платеж|purchase|payment|spent|"
                            + "поступлен|поступил|пополнен|выплат|зачисл|начислен|"
                            + "кэшб[еэ]к|кешб[еэ]к|cashback|перевод\\s+по|перевод\\s+от|"
                            + "вам\\s+перевел)",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private static final Pattern MARKETPLACE_MARKETING =
            Pattern.compile(
                    "(у\\s+вас\\s+на\\s+сч[её]те|потратьте|подобрать|купить\\s+сейчас|вау-цен|"
                            + "пункт[еа]?\\s+выдач|доставил|можно\\s+забирать|личное\\s+приглашение|"
                            + "\\d+\\s*%|выгодой\\s+до)",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private static final Pattern WALLET_TITLE =
            Pattern.compile(
                    "(кошел[её]к|банк|ozon\\s*card|wb\\s*банк)",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private DraftNotifyProvisional() {}

    static int notificationId(String dedupeKey) {
        if (dedupeKey == null) return 0x45000000;
        return 0x45000000 | (dedupeKey.hashCode() & 0x00FFFFFF);
    }

    static void show(
            Context context,
            String dedupeKey,
            String packageName,
            String title,
            String text,
            String bigText) {
        if (!DraftNotifyStore.isShadeEnabled(context)
                || !NotificationInterceptStore.isCaptureEnabled(context)
                || !DraftNotifyHelper.canPostNotifications(context)) {
            return;
        }
        if (TextUtils.isEmpty(dedupeKey)) return;
        if (!shouldShow(packageName, title, text, bigText)) return;
        DraftNotifyHelper.ensureChannel(context);

        String bodySource = firstNonEmpty(bigText, text, title);
        String amount = extractAmount(combine(title, text, bigText));
        String notifTitle =
                !TextUtils.isEmpty(title)
                        ? title.trim()
                        : context.getString(R.string.draft_notify_provisional_title);
        String body =
                !TextUtils.isEmpty(amount)
                        ? context.getString(R.string.draft_notify_body_amount, amount)
                        : truncate(bodySource, 120);
        if (TextUtils.isEmpty(body)) {
            body = context.getString(R.string.draft_notify_provisional_body);
        }

        PendingIntent open =
                WidgetDeepLinks.open(
                        context,
                        notificationId(dedupeKey),
                        "/settings/bank-notifications/drafts");

        NotificationCompat.Builder builder =
                new NotificationCompat.Builder(context, DraftNotifyHelper.CHANNEL_ID)
                        .setSmallIcon(R.drawable.ic_launcher_monochrome)
                        .setContentTitle(notifTitle)
                        .setContentText(body)
                        .setStyle(new NotificationCompat.BigTextStyle().bigText(body))
                        .setContentIntent(open)
                        .setAutoCancel(true)
                        .setOnlyAlertOnce(true)
                        .setCategory(NotificationCompat.CATEGORY_STATUS)
                        .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                        .addAction(
                                0,
                                context.getString(R.string.draft_notify_provisional_open),
                                open);

        try {
            NotificationManagerCompat.from(context)
                    .notify(notificationId(dedupeKey), builder.build());
            rememberId(context, dedupeKey);
        } catch (SecurityException ignored) {
            // POST_NOTIFICATIONS denied
        }
    }

    /** Cancel one provisional notice (e.g. JS parse returned null). */
    static void cancel(Context context, String dedupeKey) {
        if (TextUtils.isEmpty(dedupeKey)) return;
        NotificationManagerCompat.from(context).cancel(notificationId(dedupeKey));
        Set<String> ids = loadIds(context);
        if (ids.remove(dedupeKey)) {
            context.getApplicationContext()
                    .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                    .edit()
                    .putStringSet(KEY_IDS, ids)
                    .apply();
        }
    }

    /** Cancel all provisional notices (called when real draft notifications are synced). */
    static void cancelAll(Context context) {
        Set<String> ids = loadIds(context);
        NotificationManagerCompat nm = NotificationManagerCompat.from(context);
        for (String key : ids) {
            nm.cancel(notificationId(key));
        }
        context.getApplicationContext()
                .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit()
                .remove(KEY_IDS)
                .apply();
    }

    /**
     * Same idea as JS parse gate: currency amount + op/wallet signal; marketplace stricter.
     * Package-visible for unit tests.
     */
    static boolean shouldShow(String packageName, String title, String text, String bigText) {
        String combined = combine(title, text, bigText);
        if (combined.isEmpty()) return false;
        if (IGNORE_NOISE.matcher(combined).find()) return false;
        boolean hasOp = OP_HINT.matcher(combined).find();
        boolean walletTitle = WALLET_TITLE.matcher(nullToEmpty(title)).find();
        boolean marketplace =
                packageName != null
                        && MARKETPLACE_PACKAGES.contains(packageName.toLowerCase(Locale.ROOT));
        if (marketplace) {
            if (MARKETPLACE_MARKETING.matcher(combined).find()) return false;
            if (!hasOp && !walletTitle) return false;
        } else if (BALANCE_ONLY.matcher(combined).find() && !hasOp) {
            return false;
        } else if (!hasOp && extractAmount(combined).isEmpty()) {
            // Bank/wallet: need op hint or at least a currency amount (merchant + sum).
            return false;
        }
        return !extractAmount(combined).isEmpty();
    }

    /**
     * Marketplace shell without wallet/op — history only, do not queue for JS.
     * Package-visible for unit tests / NLS.
     */
    static boolean shouldQueuePackage(String packageName, String title, String text, String bigText) {
        if (packageName == null || packageName.isEmpty()) return false;
        if (!MARKETPLACE_PACKAGES.contains(packageName.toLowerCase(Locale.ROOT))) {
            return true;
        }
        String combined = combine(title, text, bigText);
        if (MARKETPLACE_MARKETING.matcher(combined).find()) return false;
        if (WALLET_TITLE.matcher(nullToEmpty(title)).find()) return true;
        return OP_HINT.matcher(combined).find();
    }

    static String extractAmount(String raw) {
        if (raw == null || raw.isEmpty()) return "";
        String scoped =
                BALANCE_TAIL.matcher(raw.replace('\u00a0', ' ')).replaceFirst(" ");
        Matcher m = AMOUNT_WITH_CURRENCY.matcher(scoped);
        if (!m.find()) return "";
        String cand = m.group(1);
        if (cand == null) return "";
        return cand.replace(" ", "").replace('\u00a0', ' ').trim().replace(',', '.');
    }

    private static void rememberId(Context context, String dedupeKey) {
        Set<String> ids = loadIds(context);
        ids.add(dedupeKey);
        context.getApplicationContext()
                .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit()
                .putStringSet(KEY_IDS, ids)
                .apply();
    }

    private static Set<String> loadIds(Context context) {
        Set<String> raw =
                context.getApplicationContext()
                        .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                        .getStringSet(KEY_IDS, null);
        return raw == null ? new HashSet<>() : new HashSet<>(raw);
    }

    private static String combine(String title, String text, String bigText) {
        StringBuilder sb = new StringBuilder();
        appendPart(sb, title);
        appendPart(sb, text);
        appendPart(sb, bigText);
        return sb.toString().trim();
    }

    private static void appendPart(StringBuilder sb, String part) {
        if (part == null) return;
        String t = part.trim();
        if (t.isEmpty()) return;
        if (sb.length() > 0) sb.append('\n');
        sb.append(t);
    }

    private static String truncate(String s, int max) {
        if (s == null) return "";
        String t = s.trim().replaceAll("\\s+", " ");
        if (t.length() <= max) return t;
        return t.substring(0, max - 1) + "…";
    }

    private static String firstNonEmpty(String... values) {
        for (String v : values) {
            if (v != null && !v.trim().isEmpty()) return v.trim();
        }
        return "";
    }

    private static String nullToEmpty(String s) {
        return s == null ? "" : s;
    }
}
