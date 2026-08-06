package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.Set;

/**
 * Persists raw bank notification payloads until the WebView consumes them.
 */
final class NotificationInterceptStore {
    private static final String PREFS = "buhgalter.notification_intercept.v1";
    private static final String KEY_PENDING = "pending";
    /** Debug history in the same prefs file as the working pending queue. */
    private static final String KEY_HISTORY = "history";
    private static final String KEY_CAPTURE = "capture_enabled";
    /** JSON string array — avoid SharedPreferences StringSet update quirks. */
    private static final String KEY_PACKAGES_JSON = "allowed_packages_json";
    /** Legacy StringSet key; migrated on read. */
    private static final String KEY_PACKAGES_LEGACY = "allowed_packages";
    private static final int MAX_PENDING = 100;
    private static final int MAX_HISTORY = 80;
    private static final Object HISTORY_LOCK = new Object();

    /**
     * Fallback when WebView has not synced the allowlist yet (cold start / MIUI kill).
     * Keep in sync with {@code banks.ts} KNOWN_BANK_APPS packages.
     */
    private static final Set<String> DEFAULT_BANK_PACKAGES =
            Collections.unmodifiableSet(
                    new HashSet<>(
                            Arrays.asList(
                                    "ru.sberbankmobile",
                                    "com.idamob.tinkoff.android",
                                    "ru.vtb24.mobilebanking.android",
                                    "ru.alfabank.mobile.android",
                                    "ru.gazprombank.android.mobilebank.app",
                                    "ru.raiffeisennews",
                                    "ru.rosbank.android",
                                    "ru.mkb.mobile",
                                    "ru.rshb.dbo",
                                    "com.openbank",
                                    "ru.sovcomcard.halva.v1",
                                    "ru.sovcombank.mobile",
                                    "ru.ftc.faktura.psb",
                                    "logo.com.mbanking",
                                    "ru.uralsib.mb",
                                    "ru.homecredit.mycredit",
                                    "ru.ozon.fintech.finance",
                                    "ru.ozon.app.android",
                                    "com.yandex.bank",
                                    "ru.wildberries.fintech",
                                    "com.wildberries.ru",
                                    "ru.otpbank.mobile",
                                    "ru.atb.mobilbank")));

    private NotificationInterceptStore() {}

    private static SharedPreferences prefs(Context context) {
        return context.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static boolean isCaptureEnabled(Context context) {
        return prefs(context).getBoolean(KEY_CAPTURE, false);
    }

    static void setCaptureEnabled(Context context, boolean enabled) {
        prefs(context).edit().putBoolean(KEY_CAPTURE, enabled).commit();
    }

    static Set<String> allowedPackages(Context context) {
        SharedPreferences p = prefs(context);
        String json = p.getString(KEY_PACKAGES_JSON, null);
        if (json != null) {
            Set<String> fromJson = packagesFromJson(json);
            // Empty JSON means "not synced" or wiped — use built-in bank list so capture works.
            return fromJson.isEmpty() ? new HashSet<>(DEFAULT_BANK_PACKAGES) : fromJson;
        }
        // One-time migrate from legacy StringSet storage.
        Set<String> legacy = p.getStringSet(KEY_PACKAGES_LEGACY, null);
        if (legacy != null && !legacy.isEmpty()) {
            Set<String> copy = new HashSet<>(legacy);
            setAllowedPackages(context, copy);
            return copy;
        }
        return new HashSet<>(DEFAULT_BANK_PACKAGES);
    }

    static void setAllowedPackages(Context context, Set<String> packages) {
        JSONArray arr = new JSONArray();
        if (packages != null) {
            for (String pkg : packages) {
                if (pkg != null && !pkg.trim().isEmpty()) {
                    arr.put(pkg.trim());
                }
            }
        }
        prefs(context)
                .edit()
                .putString(KEY_PACKAGES_JSON, arr.toString())
                .remove(KEY_PACKAGES_LEGACY)
                .commit();
    }

    private static Set<String> packagesFromJson(String json) {
        Set<String> out = new HashSet<>();
        try {
            JSONArray arr = new JSONArray(json);
            for (int i = 0; i < arr.length(); i++) {
                String pkg = arr.optString(i, "").trim();
                if (!pkg.isEmpty()) {
                    out.add(pkg);
                }
            }
        } catch (JSONException ignored) {
            // empty
        }
        return out;
    }

    static synchronized void append(Context context, JSONObject item) {
        if (!isCaptureEnabled(context)) {
            return;
        }
        String pkg = item.optString("packageName", "");
        if (pkg.isEmpty() || !allowedPackages(context).contains(pkg)) {
            return;
        }
        try {
            JSONArray arr = readArray(context);
            String dedupeKey = item.optString("dedupeKey", "");
            if (!dedupeKey.isEmpty()) {
                for (int i = 0; i < arr.length(); i++) {
                    if (dedupeKey.equals(arr.getJSONObject(i).optString("dedupeKey", ""))) {
                        return;
                    }
                }
            }
            arr.put(item);
            while (arr.length() > MAX_PENDING) {
                JSONArray trimmed = new JSONArray();
                for (int i = arr.length() - MAX_PENDING; i < arr.length(); i++) {
                    trimmed.put(arr.get(i));
                }
                arr = trimmed;
            }
            prefs(context).edit().putString(KEY_PENDING, arr.toString()).apply();
        } catch (JSONException ignored) {
            // ignore malformed queue
        }
    }

    static synchronized JSONArray consumeAll(Context context) {
        JSONArray arr = readArray(context);
        prefs(context).edit().putString(KEY_PENDING, "[]").apply();
        return arr;
    }

    static synchronized JSONArray peekAll(Context context) {
        return readArray(context);
    }

    /** Remove pending rows by dedupeKey (leave the rest for a later retry). */
    static synchronized void acknowledge(Context context, Set<String> dedupeKeys) {
        if (dedupeKeys == null || dedupeKeys.isEmpty()) {
            return;
        }
        try {
            JSONArray arr = readArray(context);
            JSONArray kept = new JSONArray();
            for (int i = 0; i < arr.length(); i++) {
                JSONObject obj = arr.getJSONObject(i);
                String key = obj.optString("dedupeKey", "");
                if (key.isEmpty() || !dedupeKeys.contains(key)) {
                    kept.put(obj);
                }
            }
            prefs(context).edit().putString(KEY_PENDING, kept.toString()).apply();
        } catch (JSONException ignored) {
            // ignore
        }
    }

    private static JSONArray readArray(Context context) {
        String raw = prefs(context).getString(KEY_PENDING, "[]");
        try {
            return new JSONArray(raw);
        } catch (JSONException e) {
            return new JSONArray();
        }
    }

    /**
     * Append a debug history row. Uses the same SharedPreferences as the pending queue
     * (known to persist on MIUI when the separate history prefs file did not show up in UI).
     */
    static void appendHistory(Context context, JSONObject item) {
        final String payload;
        synchronized (HISTORY_LOCK) {
            try {
                JSONArray arr = readHistoryArray(context);
                String dedupeKey = item.optString("dedupeKey", "");
                if (!dedupeKey.isEmpty()) {
                    for (int i = 0; i < arr.length(); i++) {
                        if (dedupeKey.equals(arr.getJSONObject(i).optString("dedupeKey", ""))) {
                            return;
                        }
                    }
                }
                JSONArray next = new JSONArray();
                next.put(item);
                for (int i = 0; i < arr.length() && next.length() < MAX_HISTORY; i++) {
                    next.put(arr.get(i));
                }
                payload = next.toString();
            } catch (JSONException e) {
                return;
            }
        }
        prefs(context).edit().putString(KEY_HISTORY, payload).commit();
    }

    /** Lock-free read for WebView. */
    static String peekHistoryJson(Context context) {
        try {
            String raw = prefs(context).getString(KEY_HISTORY, "[]");
            return raw != null ? raw : "[]";
        } catch (RuntimeException e) {
            return "[]";
        }
    }

    static void clearHistory(Context context) {
        prefs(context).edit().putString(KEY_HISTORY, "[]").commit();
    }

    private static JSONArray readHistoryArray(Context context) {
        String raw = prefs(context).getString(KEY_HISTORY, "[]");
        try {
            return new JSONArray(raw != null ? raw : "[]");
        } catch (JSONException e) {
            return new JSONArray();
        }
    }
}
