package ru.kai_zer.buhgalter;

import android.content.Context;
import android.webkit.JavascriptInterface;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

/**
 * Synchronous WebView bridge for notification history.
 * Capacitor plugin promises can hang on some OEM WebViews; this path does not use them.
 */
public final class NotificationHistoryJsBridge {
    private final Context app;

    public NotificationHistoryJsBridge(Context context) {
        this.app = context.getApplicationContext();
    }

    @JavascriptInterface
    public String peekJson() {
        try {
            String primary = NotificationInterceptStore.peekHistoryJson(app);
            String legacy = NotificationHistoryStore.peekAllJson(app);
            return mergeHistoryJson(primary, legacy);
        } catch (RuntimeException e) {
            return "[]";
        }
    }

    @JavascriptInterface
    public void clear() {
        try {
            NotificationInterceptStore.clearHistory(app);
            NotificationHistoryStore.clear(app);
        } catch (RuntimeException ignored) {
            // ignore
        }
    }

    @JavascriptInterface
    public String listenerStateJson() {
        try {
            JSONObject o = new JSONObject();
            o.put("listenerConnected", BankNotificationListenerService.isConnected());
            o.put("captureEnabled", NotificationInterceptStore.isCaptureEnabled(app));
            o.put("notificationAccess", hasNotificationAccess());
            o.put("allowedPackageCount", NotificationInterceptStore.allowedPackages(app).size());
            String hist = NotificationInterceptStore.peekHistoryJson(app);
            int n = 0;
            try {
                n = new JSONArray(hist).length();
            } catch (JSONException ignored) {
                n = 0;
            }
            o.put("historyCount", n);
            return o.toString();
        } catch (Exception e) {
            return "{}";
        }
    }

    private static String mergeHistoryJson(String primary, String legacy) {
        try {
            JSONArray a = new JSONArray(primary != null ? primary : "[]");
            JSONArray b = new JSONArray(legacy != null ? legacy : "[]");
            if (b.length() == 0) {
                return a.toString();
            }
            if (a.length() == 0) {
                return b.toString();
            }
            java.util.HashSet<String> seen = new java.util.HashSet<>();
            JSONArray out = new JSONArray();
            for (int i = 0; i < a.length(); i++) {
                JSONObject o = a.getJSONObject(i);
                String key = o.optString("dedupeKey", "");
                if (!key.isEmpty()) {
                    seen.add(key);
                }
                out.put(o);
            }
            for (int i = 0; i < b.length() && out.length() < 80; i++) {
                JSONObject o = b.getJSONObject(i);
                String key = o.optString("dedupeKey", "");
                if (!key.isEmpty() && seen.contains(key)) {
                    continue;
                }
                if (!key.isEmpty()) {
                    seen.add(key);
                }
                out.put(o);
            }
            return out.toString();
        } catch (JSONException e) {
            return primary != null ? primary : "[]";
        }
    }

    private boolean hasNotificationAccess() {
        String flat =
                android.provider.Settings.Secure.getString(
                        app.getContentResolver(), "enabled_notification_listeners");
        if (android.text.TextUtils.isEmpty(flat)) {
            return false;
        }
        android.content.ComponentName expected =
                new android.content.ComponentName(app, BankNotificationListenerService.class);
        for (String entry : flat.split(":")) {
            android.content.ComponentName cn =
                    android.content.ComponentName.unflattenFromString(entry);
            if (cn != null && cn.equals(expected)) {
                return true;
            }
        }
        return false;
    }
}
