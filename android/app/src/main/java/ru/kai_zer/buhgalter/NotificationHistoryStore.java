package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

/**
 * Ring buffer of recent notifications for intercept debugging (on-device only).
 *
 * <p>Reads must never block the WebView JS thread. Do not hold a monitor across
 * {@link SharedPreferences.Editor#commit()}.
 */
final class NotificationHistoryStore {
    private static final String PREFS = "buhgalter.notification_intercept.history.v1";
    private static final String KEY_ITEMS = "items";
    private static final int MAX_ITEMS = 80;
    private static final Object WRITE_LOCK = new Object();

    private NotificationHistoryStore() {}

    private static SharedPreferences prefs(Context context) {
        return context.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static void append(Context context, JSONObject item) {
        final String payload;
        synchronized (WRITE_LOCK) {
            try {
                JSONArray arr = readArray(context);
                JSONArray next = new JSONArray();
                next.put(item);
                for (int i = 0; i < arr.length() && next.length() < MAX_ITEMS; i++) {
                    next.put(arr.get(i));
                }
                payload = next.toString();
            } catch (JSONException e) {
                return;
            }
        }
        // Persist outside the monitor so WebView peekJson cannot deadlock.
        prefs(context).edit().putString(KEY_ITEMS, payload).commit();
    }

    static JSONArray peekAll(Context context) {
        return readArray(context);
    }

    /** Lock-free read for WebView / Capacitor. */
    static String peekAllJson(Context context) {
        try {
            String raw = prefs(context).getString(KEY_ITEMS, "[]");
            return raw != null ? raw : "[]";
        } catch (RuntimeException e) {
            return "[]";
        }
    }

    static void clear(Context context) {
        prefs(context).edit().putString(KEY_ITEMS, "[]").commit();
    }

    private static JSONArray readArray(Context context) {
        String raw = prefs(context).getString(KEY_ITEMS, "[]");
        try {
            return new JSONArray(raw != null ? raw : "[]");
        } catch (JSONException e) {
            return new JSONArray();
        }
    }
}
