package ru.kai_zer.buhgalter;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.text.TextUtils;

import androidx.core.app.NotificationCompat;
import androidx.core.app.NotificationManagerCompat;
import androidx.core.content.ContextCompat;

import org.json.JSONArray;
import org.json.JSONObject;

/**
 * Shows / cancels Buhgalter draft notifications with Accept / Reject actions.
 */
final class DraftNotifyHelper {
    static final String CHANNEL_ID = "buhgalter_intercept_drafts";
    static final String ACTION_ACCEPT = "ru.kai_zer.buhgalter.DRAFT_ACCEPT";
    static final String ACTION_REJECT = "ru.kai_zer.buhgalter.DRAFT_REJECT";
    static final String ACTION_OPEN = "ru.kai_zer.buhgalter.DRAFT_OPEN";
    static final String EXTRA_DRAFT_ID = "draft_id";

    private DraftNotifyHelper() {}

    static void ensureChannel(Context context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return;
        NotificationManager nm =
                (NotificationManager) context.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        NotificationChannel existing = nm.getNotificationChannel(CHANNEL_ID);
        if (existing != null) return;
        NotificationChannel ch =
                new NotificationChannel(
                        CHANNEL_ID,
                        context.getString(R.string.draft_notify_channel_name),
                        NotificationManager.IMPORTANCE_DEFAULT);
        ch.setDescription(context.getString(R.string.draft_notify_channel_desc));
        nm.createNotificationChannel(ch);
    }

    static boolean canPostNotifications(Context context) {
        if (Build.VERSION.SDK_INT < 33) {
            return NotificationManagerCompat.from(context).areNotificationsEnabled();
        }
        return ContextCompat.checkSelfPermission(
                        context, android.Manifest.permission.POST_NOTIFICATIONS)
                == PackageManager.PERMISSION_GRANTED;
    }

    static int notificationIdForDraft(String draftId) {
        if (draftId == null) return 0;
        return 0x44000000 | (draftId.hashCode() & 0x00FFFFFF);
    }

    static void cancelAll(Context context) {
        NotificationManagerCompat nm = NotificationManagerCompat.from(context);
        JSONArray drafts = DraftNotifyStore.listDrafts(context);
        for (int i = 0; i < drafts.length(); i++) {
            JSONObject o = drafts.optJSONObject(i);
            if (o == null) continue;
            String id = o.optString("id", "");
            if (!id.isEmpty()) {
                nm.cancel(notificationIdForDraft(id));
            }
        }
    }

    static void cancelDraft(Context context, String draftId) {
        if (TextUtils.isEmpty(draftId)) return;
        NotificationManagerCompat.from(context).cancel(notificationIdForDraft(draftId));
    }

    /**
     * Refresh shade notifications from the current mirror. No-ops when shade/capture/permission off.
     */
    static void refreshAll(Context context) {
        ensureChannel(context);
        boolean shadeOn = DraftNotifyStore.isShadeEnabled(context);
        boolean captureOn = NotificationInterceptStore.isCaptureEnabled(context);
        if (!shadeOn || !captureOn || !canPostNotifications(context)) {
            cancelAll(context);
            return;
        }
        JSONArray drafts = DraftNotifyStore.listDrafts(context);
        NotificationManagerCompat nm = NotificationManagerCompat.from(context);
        for (int i = 0; i < drafts.length(); i++) {
            JSONObject draft = drafts.optJSONObject(i);
            if (draft == null) continue;
            String id = draft.optString("id", "");
            if (DraftNotifyStore.isSnoozed(context, id)) {
                cancelDraft(context, id);
                continue;
            }
            showOne(context, nm, draft);
        }
    }

    static void upsertAndNotify(Context context, JSONObject draft) {
        if (draft == null) return;
        DraftNotifyStore.upsertDraft(context, draft);
        ensureChannel(context);
        String draftId = draft.optString("id", "");
        if (!DraftNotifyStore.isShadeEnabled(context)
                || !NotificationInterceptStore.isCaptureEnabled(context)
                || !canPostNotifications(context)
                || DraftNotifyStore.isSnoozed(context, draftId)) {
            cancelDraft(context, draftId);
            return;
        }
        showOne(context, NotificationManagerCompat.from(context), draft);
    }

    /** Deep-link path: open create form for this shade item (not the drafts list). */
    static String contentPath(JSONObject draft) {
        if (draft == null) return draftsListPath();
        if ("transfer".equals(draft.optString("type", ""))) {
            String fromId = draft.optString("fromDraftId", "");
            String toId = draft.optString("toDraftId", "");
            if (!TextUtils.isEmpty(fromId) && !TextUtils.isEmpty(toId)) {
                return "/transfers/new?from=/settings/bank-notifications/drafts"
                        + "&intercept_drafts="
                        + UriEncode.encode(fromId)
                        + ","
                        + UriEncode.encode(toId);
            }
            return draftsListPath();
        }
        String draftId = draft.optString("id", "");
        if (TextUtils.isEmpty(draftId) || draftId.startsWith("xfer:")) {
            return draftsListPath();
        }
        String type = "income".equals(draft.optString("type", "")) ? "income" : "expense";
        return "/transactions/new?type="
                + type
                + "&from=/settings/bank-notifications/drafts"
                + "&intercept_draft="
                + UriEncode.encode(draftId);
    }

    static String draftsListPath() {
        return "/settings/bank-notifications/drafts";
    }

    private static void showOne(Context context, NotificationManagerCompat nm, JSONObject draft) {
        String draftId = draft.optString("id", "");
        if (draftId.isEmpty()) return;

        boolean transfer = "transfer".equals(draft.optString("type", ""));
        boolean income = "income".equals(draft.optString("type", ""));
        String amount = draft.optString("amount", "");
        String merchant =
                firstNonEmpty(
                        draft.optString("merchantName", ""),
                        draft.optString("merchantText", ""),
                        draft.optString("description", ""));

        String title;
        if (transfer) {
            title = context.getString(R.string.draft_notify_title_transfer);
        } else if (!TextUtils.isEmpty(merchant)) {
            title = merchant;
        } else {
            title =
                    context.getString(
                            income
                                    ? R.string.draft_notify_title_income
                                    : R.string.draft_notify_title_expense);
        }
        String body =
                TextUtils.isEmpty(amount)
                        ? context.getString(R.string.draft_notify_body_fallback)
                        : context.getString(R.string.draft_notify_body_amount, amount);

        PendingIntent content =
                actionPending(context, ACTION_OPEN, draftId, notificationIdForDraft(draftId));
        PendingIntent accept =
                actionPending(context, ACTION_ACCEPT, draftId, notificationIdForDraft(draftId) + 1);
        PendingIntent reject =
                actionPending(context, ACTION_REJECT, draftId, notificationIdForDraft(draftId) + 2);

        NotificationCompat.Builder builder =
                new NotificationCompat.Builder(context, CHANNEL_ID)
                        .setSmallIcon(R.drawable.ic_launcher_monochrome)
                        .setContentTitle(title)
                        .setContentText(body)
                        .setStyle(new NotificationCompat.BigTextStyle().bigText(body))
                        .setContentIntent(content)
                        .setAutoCancel(true)
                        .setOnlyAlertOnce(true)
                        .setCategory(NotificationCompat.CATEGORY_MESSAGE)
                        .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                        .addAction(
                                0,
                                context.getString(R.string.draft_notify_action_accept),
                                accept)
                        .addAction(
                                0,
                                context.getString(R.string.draft_notify_action_reject),
                                reject);

        try {
            nm.notify(notificationIdForDraft(draftId), builder.build());
        } catch (SecurityException ignored) {
            // POST_NOTIFICATIONS denied at runtime
        }
    }

    private static PendingIntent actionPending(
            Context context, String action, String draftId, int requestCode) {
        Intent intent = new Intent(context, DraftNotifyReceiver.class);
        intent.setAction(action);
        intent.putExtra(EXTRA_DRAFT_ID, draftId);
        return PendingIntent.getBroadcast(
                context,
                requestCode,
                intent,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }

    private static String firstNonEmpty(String... values) {
        for (String v : values) {
            if (v != null && !v.trim().isEmpty()) return v.trim();
        }
        return "";
    }

    /** Minimal URL-encode for draft UUID (no spaces expected). */
    static final class UriEncode {
        private UriEncode() {}

        static String encode(String raw) {
            try {
                return java.net.URLEncoder.encode(raw, "UTF-8");
            } catch (Exception e) {
                return raw;
            }
        }
    }
}
