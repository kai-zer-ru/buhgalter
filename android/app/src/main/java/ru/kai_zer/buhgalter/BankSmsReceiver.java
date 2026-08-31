package ru.kai_zer.buhgalter;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.telephony.SmsMessage;

import androidx.core.content.ContextCompat;

import org.json.JSONException;
import org.json.JSONObject;

/**
 * Captures bank SMS into {@link NotificationInterceptStore} (same pending queue as push).
 * Requires runtime {@code RECEIVE_SMS} and capture enabled + sender allowlist.
 */
public class BankSmsReceiver extends BroadcastReceiver {

    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null || intent.getAction() == null) {
            return;
        }
        if (!"android.provider.Telephony.SMS_RECEIVED".equals(intent.getAction())) {
            return;
        }
        Context app = context.getApplicationContext();
        if (!NotificationInterceptStore.isCaptureEnabled(app)) {
            return;
        }
        if (ContextCompat.checkSelfPermission(app, android.Manifest.permission.RECEIVE_SMS)
                != PackageManager.PERMISSION_GRANTED) {
            return;
        }

        Bundle bundle = intent.getExtras();
        if (bundle == null) {
            return;
        }
        Object[] pdus = (Object[]) bundle.get("pdus");
        if (pdus == null || pdus.length == 0) {
            return;
        }
        String format = bundle.getString("format");
        boolean anyQueued = false;
        for (Object pdu : pdus) {
            if (!(pdu instanceof byte[])) {
                continue;
            }
            SmsMessage msg = createSmsMessage((byte[]) pdu, format);
            if (msg == null) {
                continue;
            }
            String sender = msg.getDisplayOriginatingAddress();
            String body = msg.getMessageBody();
            if (sender == null || sender.trim().isEmpty()) {
                continue;
            }
            if (body == null) {
                body = "";
            }
            body = body.trim();
            if (body.isEmpty()) {
                continue;
            }
            String packageName = NotificationInterceptStore.packageForSmsSender(app, sender);
            if (packageName == null || packageName.isEmpty()) {
                continue;
            }
            long postedAt = msg.getTimestampMillis();
            if (postedAt <= 0) {
                postedAt = System.currentTimeMillis();
            }
            String senderNorm = NotificationInterceptStore.normalizeSmsSender(sender);
            String dedupeKey =
                    "sms|" + senderNorm + "|" + postedAt + "|" + Integer.toHexString(body.hashCode());
            try {
                JSONObject historyItem = new JSONObject();
                historyItem.put("packageName", packageName);
                historyItem.put("title", sender.trim());
                historyItem.put("text", body);
                historyItem.put("bigText", "");
                historyItem.put("postedAt", postedAt);
                historyItem.put("dedupeKey", dedupeKey);
                historyItem.put("channel", "sms");
                historyItem.put("inAllowlist", true);
                historyItem.put("queued", true);
                NotificationInterceptStore.appendHistory(app, historyItem);
                NotificationHistoryStore.append(app, historyItem);

                JSONObject item = new JSONObject();
                item.put("packageName", packageName);
                item.put("title", sender.trim());
                item.put("text", body);
                item.put("bigText", "");
                item.put("postedAt", postedAt);
                item.put("dedupeKey", dedupeKey);
                item.put("channel", "sms");
                NotificationInterceptStore.append(app, item);
                anyQueued = true;
            } catch (JSONException ignored) {
                // ignore malformed row
            }
        }
        if (anyQueued) {
            NotificationInterceptPlugin.emitPendingAvailable();
        }
    }

    private static SmsMessage createSmsMessage(byte[] pdu, String format) {
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M && format != null) {
                return SmsMessage.createFromPdu(pdu, format);
            }
            return SmsMessage.createFromPdu(pdu);
        } catch (RuntimeException e) {
            return null;
        }
    }
}
