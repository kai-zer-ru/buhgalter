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
    static final String EXTRA_QUIET_WAKE_TOKEN = "buhgalter.quiet_wake_token";

    private static final long WAKE_COOLDOWN_MS = 15_000L;
    private static volatile long lastWakeElapsedMs;
    private static volatile long pendingQuietWakeToken;
    private static volatile boolean quietWakeActive;
    private static volatile boolean userEngaged;

    private InterceptPendingWake() {}

    static void onQueued(
            Context context,
            String dedupeKey,
            String packageName,
            String title,
            String text,
            String bigText) {
        Context app = context.getApplicationContext();
        DraftNotifyProvisional.show(app, dedupeKey, packageName, title, text, bigText);
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
     * Activity left the foreground after a normal user session. Next quiet wake may hide again.
     * Skipped while {@code quietWakeActive} so a wake mid-parse does not clear engagement races.
     */
    static void onActivityStopped() {
        if (!quietWakeActive) {
            userEngaged = false;
        }
    }

    /**
     * Launcher / recents / PIN / fingerprint: the user is looking at the app.
     * Clears quiet hide so finishQuietWake must not {@code moveTaskToBack}.
     */
    static void markUserEngaged() {
        userEngaged = true;
        quietWakeActive = false;
        pendingQuietWakeToken = 0;
    }

    /**
     * Consumes a one-shot quiet-wake delivery. Sticky Intent extras alone must not
     * re-trigger hide after process death or HyperOS bring-to-front without a new start.
     *
     * @return true only for a fresh quiet wake that has not been consumed yet
     */
    static boolean consumeQuietWakeIntent(Intent intent) {
        if (intent == null) {
            return false;
        }
        boolean quietExtra = intent.getBooleanExtra(EXTRA_QUIET_WAKE, false);
        long token = intent.getLongExtra(EXTRA_QUIET_WAKE_TOKEN, 0L);
        // Strip sticky extras immediately so recreate / next launcher open cannot replay.
        if (quietExtra) {
            intent.removeExtra(EXTRA_QUIET_WAKE);
            intent.removeExtra(EXTRA_QUIET_WAKE_TOKEN);
        }
        return consumeQuietWake(quietExtra, token);
    }

    /** Pure one-shot gate (unit-tested without Android Intent). */
    static boolean consumeQuietWake(boolean quietExtra, long token) {
        if (!quietExtra) {
            return false;
        }
        long expected = pendingQuietWakeToken;
        if (token == 0L || expected == 0L || token != expected) {
            return false;
        }
        pendingQuietWakeToken = 0;
        quietWakeActive = true;
        // Do not clear userEngaged: if the user already opened/unlocked, never hide.
        return true;
    }

    /** Hide after processing only if this was a true background wake, not a user open / unlock. */
    static boolean shouldMoveTaskToBack(boolean quietActive, boolean engaged, boolean pinEnabled) {
        return quietActive && !engaged && !pinEnabled;
    }

    static void resetForTests() {
        quietWakeActive = false;
        userEngaged = false;
        lastWakeElapsedMs = 0;
        pendingQuietWakeToken = 0;
    }

    /** Test helper: arm the same one-shot token {@link #maybeQuietWake} would put on the Intent. */
    static long armQuietWakeTokenForTests() {
        long token = SystemClock.elapsedRealtime();
        if (token == 0L) {
            token = 1L;
        }
        pendingQuietWakeToken = token;
        return token;
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
        long token = now == 0L ? 1L : now;
        pendingQuietWakeToken = token;
        try {
            Intent i = new Intent(app, MainActivity.class);
            i.addFlags(
                    Intent.FLAG_ACTIVITY_NEW_TASK
                            | Intent.FLAG_ACTIVITY_SINGLE_TOP
                            | Intent.FLAG_ACTIVITY_REORDER_TO_FRONT);
            i.putExtra(EXTRA_QUIET_WAKE, true);
            i.putExtra(EXTRA_QUIET_WAKE_TOKEN, token);
            app.startActivity(i);
        } catch (RuntimeException ignored) {
            // OEM may block background activity starts — provisional notice still helps.
            pendingQuietWakeToken = 0;
        }
    }
}
