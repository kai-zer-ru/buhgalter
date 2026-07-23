package ru.kai_zer.buhgalter.widgets;

import android.content.Context;
import android.content.SharedPreferences;

import androidx.security.crypto.EncryptedSharedPreferences;
import androidx.security.crypto.MasterKeys;

import org.json.JSONObject;

public final class WidgetSnapshotStore {
    private static final String PREFS = "buhgalter_widget_bridge";
    private static final String KEY_BASE_URL = "base_url";
    private static final String KEY_TOKEN = "token";
    private static final String KEY_LOCK = "lock_enabled";
    private static final String KEY_SNAPSHOT = "snapshot_json";

    private WidgetSnapshotStore() {}

    private static SharedPreferences prefs(Context context) {
        try {
            String masterKeyAlias = MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC);
            return EncryptedSharedPreferences.create(
                    PREFS,
                    masterKeyAlias,
                    context,
                    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM);
        } catch (Exception e) {
            return context.getSharedPreferences(PREFS + "_fallback", Context.MODE_PRIVATE);
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
