package ru.kai_zer.buhgalter.widgets;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Looper;

import androidx.security.crypto.EncryptedSharedPreferences;
import androidx.security.crypto.MasterKeys;

import org.json.JSONObject;

/**
 * Widget snapshot + token. {@link MasterKeys#getOrCreate} on Xiaomi Keystore can block
 * ~30s after Dual Apps — never call it on the main thread (widget {@code onUpdate}
 * and cold start share that thread with the activity).
 */
public final class WidgetSnapshotStore {
    private static final String PREFS = "buhgalter_widget_bridge";
    private static final String KEY_BASE_URL = "base_url";
    private static final String KEY_TOKEN = "token";
    private static final String KEY_LOCK = "lock_enabled";
    private static final String KEY_SNAPSHOT = "snapshot_json";
    private static final Object LOCK = new Object();

    private static volatile SharedPreferences cached;

    private WidgetSnapshotStore() {}

    /** Open encrypted prefs off the UI thread so later widget binds are instant. */
    public static void warmup(Context context) {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            throw new IllegalStateException("warmup must not run on the main thread");
        }
        prefs(context);
    }

    static SharedPreferences resolvePrefs(
            SharedPreferences cachedPrefs,
            SharedPreferences fallback,
            SharedPreferences encrypted,
            boolean mainThread) {
        if (cachedPrefs != null) {
            return cachedPrefs;
        }
        if (mainThread) {
            return fallback;
        }
        return encrypted != null ? encrypted : fallback;
    }

    private static SharedPreferences fallbackPrefs(Context context) {
        return context.getApplicationContext()
                .getSharedPreferences(PREFS + "_fallback", Context.MODE_PRIVATE);
    }

    private static SharedPreferences openEncrypted(Context context) {
        try {
            String masterKeyAlias = MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC);
            return EncryptedSharedPreferences.create(
                    PREFS,
                    masterKeyAlias,
                    context.getApplicationContext(),
                    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM);
        } catch (Exception e) {
            return null;
        }
    }

    private static SharedPreferences prefs(Context context) {
        SharedPreferences hit = cached;
        if (hit != null) {
            return hit;
        }
        SharedPreferences fallback = fallbackPrefs(context);
        boolean mainThread = Looper.myLooper() == Looper.getMainLooper();
        if (mainThread) {
            return resolvePrefs(null, fallback, null, true);
        }
        synchronized (LOCK) {
            if (cached != null) {
                return cached;
            }
            SharedPreferences encrypted = openEncrypted(context);
            cached = resolvePrefs(null, fallback, encrypted, false);
            return cached;
        }
    }

    public static void publish(
            Context context, String baseUrl, String token, boolean lockEnabled, String snapshotJson) {
        prefs(context)
                .edit()
                .putString(KEY_BASE_URL, baseUrl != null ? baseUrl : "")
                .putString(KEY_TOKEN, token != null ? token : "")
                .putBoolean(KEY_LOCK, lockEnabled)
                .putString(KEY_SNAPSHOT, snapshotJson != null ? snapshotJson : "")
                .apply();
    }

    public static void setLockEnabled(Context context, boolean lockEnabled) {
        prefs(context).edit().putBoolean(KEY_LOCK, lockEnabled).apply();
    }

    public static void clear(Context context) {
        prefs(context).edit().clear().apply();
    }

    public static String getBaseUrl(Context context) {
        return prefs(context).getString(KEY_BASE_URL, "");
    }

    public static String getToken(Context context) {
        return prefs(context).getString(KEY_TOKEN, "");
    }

    public static boolean isLockEnabled(Context context) {
        return prefs(context).getBoolean(KEY_LOCK, false);
    }

    public static String getSnapshotJson(Context context) {
        return prefs(context).getString(KEY_SNAPSHOT, "");
    }

    public static JSONObject getSnapshot(Context context) {
        String raw = getSnapshotJson(context);
        if (raw == null || raw.isEmpty()) return null;
        try {
            return new JSONObject(raw);
        } catch (Exception e) {
            return null;
        }
    }

    public static boolean hasAuth(Context context) {
        String base = getBaseUrl(context);
        String token = getToken(context);
        return base != null && !base.isEmpty() && token != null && !token.isEmpty();
    }
}
