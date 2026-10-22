package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.pm.ApplicationInfo;

import java.util.Locale;

/**
 * OEM dual-app / «клонировать приложение» (Xiaomi user 999 and similar).
 * A clone shares applicationId with the main install and can lock or corrupt
 * WebView LevelDB and Android Keystore — do not start Capacitor there.
 */
public final class CloneGuard {
    /** Xiaomi / OPPO / Vivo Dual Apps user. */
    public static final int DUAL_APP_USER_ID = 999;

    private CloneGuard() {}

    public static int userIdFromUid(int uid) {
        return uid / 100_000;
    }

    public static boolean isDualAppUserId(int userId) {
        return userId == DUAL_APP_USER_ID;
    }

    public static boolean dataDirLooksLikeClone(String dataDir) {
        if (dataDir == null || dataDir.isEmpty()) {
            return false;
        }
        String normalized = dataDir.replace('\\', '/').toLowerCase(Locale.US);
        return normalized.contains("/user/999/")
                || normalized.contains("/user_de/999/")
                || normalized.contains("/parallel")
                || normalized.contains("virtualxposed")
                || normalized.contains("dualaid");
    }

    public static boolean isUnsupportedClone(int uid, String dataDir) {
        return isDualAppUserId(userIdFromUid(uid)) || dataDirLooksLikeClone(dataDir);
    }

    public static boolean isUnsupportedClone(Context context) {
        ApplicationInfo info = context.getApplicationInfo();
        return isUnsupportedClone(info.uid, info.dataDir);
    }
}
