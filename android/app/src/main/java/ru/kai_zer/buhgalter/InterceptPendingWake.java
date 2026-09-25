package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.Intent;
import android.os.SystemClock;

/**
 * After a bank push/SMS is queued: provisional shade notice + wake WebView when the UI is dead
 * so JS can parse drafts and post Accept/Reject notifications without a manual open.
 */
final class InterceptPendingWake {
    static final String EXTRA_QUIET_WAKE = "buhgalter.quiet_wake_pending";

    private static final long WAKE_COOLDOWN_MS = 15_000L;
    private static volatile long lastWakeElapsedMs;
    private static volatile boolean quietWakeActive;

    private InterceptPendingWake() {}

    static void onQueued(
            Context context, String dedupeKey, String title, String text, String bigText) {
        Context app = context.getApplicationContext();
        DraftNotifyProvisional.show(app, dedupeKey, title, text, bigText);
        NotificationInterceptPlugin.emitPendingAvailable();
        maybeQuietWake(app);
    }

    static boolean isQuietWakeActive() {
        return quietWakeActive;
    }

    static void clearQuietWake() {
        quietWakeActive = false;
    }

    static void markQuietWakeFromIntent(Intent intent) {
        if (intent != null && intent.getBooleanExtra(EXTRA_QUIET_WAKE, false)) {
            quietWakeActive = true;
            intent.removeExtra(EXTRA_QUIET_WAKE);
        }
    }

    private static void maybeQuietWake(Context app) {
        if (NotificationInterceptPlugin.hasLiveBridge()) {
            return;
        }
        // PIN/app-lock blocks intercept init until unlock — don't flash the lock screen.
        try {
            if (ru.kai_zer.buhgalter.widgets.WidgetSnapshotStore.isLockEnabled(app)) {
                return;
            }
        } catch (RuntimeException ignored) {
            // prefs may be cold; skip wake
            return;
        }
        long now = SystemClock.elapsedRealtime();
        long prev = lastWakeElapsedMs;
        if (prev > 0 && now - prev < WAKE_COOLDOWN_MS) {
            return;
        }
        lastWakeElapsedMs = now;
        try {
            Intent i = new Intent(app, MainActivity.class);
            i.addFlags(
                    Intent.FLAG_ACTIVITY_NEW_TASK
                            | Intent.FLAG_ACTIVITY_SINGLE_TOP
                            | Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
            i.putExtra(EXTRA_QUIET_WAKE, true);
            app.startActivity(i);
        } catch (RuntimeException ignored) {
            // OEM may block background activity starts — provisional notice still helps.
        }
    }
}
