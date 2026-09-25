package ru.kai_zer.buhgalter;

import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.text.TextUtils;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;

import java.util.HashSet;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import ru.kai_zer.buhgalter.widgets.WidgetDeepLinks;

/**
 * Immediate shade notice when a bank push/SMS is queued, before JS parses a draft.
 * Replaced by real Accept/Reject notifications after {@code syncDraftNotifications}.
 */
final class DraftNotifyProvisional {
    private static final String PREFS = "buhgalter.draft_notify.provisional.v1";
    private static final String KEY_IDS = "ids";
    private static final Pattern AMOUNT =
            Pattern.compile(
                    "(?<!\\d)(\\d{1,3}(?:[\\s\\u00a0]\\d{3})*(?:[.,]\\d{1,2})?|\\d+[.,]\\d{1,2}|\\d+)\\s*(?:₽|RUB|руб\\.?)?",
                    Pattern.CASE_INSENSITIVE | Pattern.UNICODE_CASE);

    private DraftNotifyProvisional() {}

    static int notificationId(String dedupeKey) {
        if (dedupeKey == null) return 0x45000000;
        return 0x45000000 | (dedupeKey.hashCode() & 0x00FFFFFF);
    }

    static void show(Context context, String dedupeKey, String title, String text, String bigText) {
        if (!DraftNotifyStore.isShadeEnabled(context)
                || !NotificationInterceptStore.isCaptureEnabled(context)
                || !DraftNotifyHelper.canPostNotifications(context)) {
            return;
        }
        if (TextUtils.isEmpty(dedupeKey)) return;
        DraftNotifyHelper.ensureChannel(context);

        String bodySource = firstNonEmpty(bigText, text, title);
        String amount = extractAmount(bodySource);
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

    static String extractAmount(String raw) {
        if (raw == null || raw.isEmpty()) return "";
        Matcher m = AMOUNT.matcher(raw.replace('\u00a0', ' '));
        String best = "";
        while (m.find()) {
            String cand = m.group(1);
            if (cand == null) continue;
            cand = cand.replace(" ", "").replace('\u00a0', ' ').trim();
            // Prefer values with decimal part (purchase amounts).
            if (cand.contains(".") || cand.contains(",")) {
                return cand.replace(',', '.');
            }
            if (best.isEmpty()) best = cand;
        }
        return best;
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
}
