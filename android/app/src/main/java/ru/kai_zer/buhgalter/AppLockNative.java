package ru.kai_zer.buhgalter;

import android.content.Context;

/**
 * Plain prefs for PIN-lock so intercept quiet-wake can skip without EncryptedSharedPreferences
 * or widget "hide amounts" (those are not the same as PIN enabled).
 */
final class AppLockNative {
    private static final String PREFS = "buhgalter_app_lock";
    private static final String KEY_PIN_ENABLED = "pin_enabled";

    private AppLockNative() {}

    static void setPinEnabled(Context context, boolean enabled) {
        context.getApplicationContext()
                .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .edit()
                .putBoolean(KEY_PIN_ENABLED, enabled)
                .commit();
    }

    static boolean isPinEnabled(Context context) {
        return context.getApplicationContext()
                .getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getBoolean(KEY_PIN_ENABLED, false);
    }
}
