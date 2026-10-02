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
    private static volatile boolean userEngaged;

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

    static boolean isUserEngaged() {
        return userEngaged;
    }

    static void clearQuietWake() {
        quietWakeActive = false;
    }

    /**
     * Launcher / recents / PIN / fingerprint: the user is looking at the app.
     * Clears quiet hide so finishQuietWake must not {@code moveTaskToBack}.
     */
    static void markUserEngaged() {
        userEngaged = true;
        quietWakeActive = false;
    }

    static void markQuietWakeFromIntent(Intent intent) {
        if (intent != null && intent.getBooleanExtra(EXTRA_QUIET_WAKE, false)) {
            quietWakeActive = true;
            userEngaged = false;
        }
    }

    /** Hide after processing only if this was a true background wake, not a user open / unlock. */
    static boolean shouldMoveTaskToBack(boolean quietActive, boolean engaged, boolean pinEnabled) {
        return quietActive && !engaged && !pinEnabled;
    }

    static void resetForTests() {
        quietWakeActive = false;
        userEngaged = false;
        lastWakeElapsedMs = 0;
    }

    private static void maybeQuietWake(Context app) {
        if (NotificationInterceptPlugin.hasLiveBridge()) {
            return;
        }
        // PIN lock: JS intercept waits until unlock — do not flash the lock screen.
        try {
            if (AppLockNative.isPinEnabled(app)
                    || ru.kai_zer.buhgalter.widgets.WidgetSnapshotStore.isLockEnabled(app)) {
                return;
            }
        } catch (RuntimeException ignored) {
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
