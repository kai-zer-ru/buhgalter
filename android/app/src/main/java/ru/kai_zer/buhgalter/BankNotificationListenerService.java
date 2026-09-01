package ru.kai_zer.buhgalter;

import android.app.Notification;
import android.content.ComponentName;
import android.content.Context;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.service.notification.NotificationListenerService;
import android.service.notification.StatusBarNotification;

import org.json.JSONException;
import org.json.JSONObject;

import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.Set;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Captures bank app push notifications into {@link NotificationInterceptStore}.
 * Also writes a debug history of all posts while capture is enabled.
 * Parsing and draft UI live in the Capacitor WebView (JS).
 *
 * <p>On MIUI/HyperOS the system often grants notification access but does not bind the
 * listener until {@link #requestRebind} / component toggle — see {@link #ensureConnected}.
 */
public class BankNotificationListenerService extends NotificationListenerService {

    private static volatile BankNotificationListenerService instance;
    private static final AtomicReference<CountDownLatch> connectLatch = new AtomicReference<>();

    /** SMS apps that mirror bank SMS in the notification shade (Google Messages, etc.). */
    private static final Set<String> MESSAGING_PACKAGES =
            Collections.unmodifiableSet(
                    new HashSet<>(
                            Arrays.asList(
                                    "com.google.android.apps.messaging",
                                    "com.samsung.android.messaging",
                                    "com.android.mms")));

    @Override
    public void onListenerConnected() {
        super.onListenerConnected();
        instance = this;
        CountDownLatch latch = connectLatch.getAndSet(null);
        if (latch != null) {
            latch.countDown();
        }
        // Catch up on shade items the system delivered while we were unbound (MIUI).
        if (NotificationInterceptStore.isCaptureEnabled(this)) {
            try {
                StatusBarNotification[] active = getActiveNotifications();
                if (active != null) {
                    boolean anyQueued = false;
                    for (StatusBarNotification sbn : active) {
                        if (process(this, sbn, true) == 2) {
                            anyQueued = true;
                        }
                    }
                    if (anyQueued) {
                        NotificationInterceptPlugin.emitPendingAvailable();
                    }
                }
            } catch (SecurityException ignored) {
                // not fully bound yet
            }
        }
    }

    @Override
    public void onListenerDisconnected() {
        if (instance == this) {
            instance = null;
        }
        super.onListenerDisconnected();
    }

    @Override
    public void onNotificationPosted(StatusBarNotification sbn) {
        if (!NotificationInterceptStore.isCaptureEnabled(this)) {
            return;
        }
        process(this, sbn, false);
    }

    static boolean isConnected() {
        return instance != null;
    }

    static ComponentName componentName(Context context) {
        return new ComponentName(context, BankNotificationListenerService.class);
    }

    /**
     * Ask the system to bind the listener (permission may already be granted).
     * Safe to call from the UI / Capacitor bridge thread.
     *
     * @return true if connected within timeout
     */
    static boolean ensureConnected(Context context, long timeoutMs) {
        if (instance != null) {
            return true;
        }
        Context app = context.getApplicationContext();
        CountDownLatch latch = new CountDownLatch(1);
        connectLatch.set(latch);
        requestSystemRebind(app);

        try {
            if (latch.await(timeoutMs, TimeUnit.MILLISECONDS)) {
                return instance != null;
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }

        // Still null — force component toggle (helps MIUI after granting access).
        if (instance == null) {
            toggleComponent(app);
            CountDownLatch latch2 = new CountDownLatch(1);
            connectLatch.set(latch2);
            requestSystemRebind(app);
            try {
                latch2.await(timeoutMs, TimeUnit.MILLISECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
        connectLatch.compareAndSet(latch, null);
        return instance != null;
    }

    private static void requestSystemRebind(Context context) {
        ComponentName cn = componentName(context);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            try {
                requestRebind(cn);
            } catch (RuntimeException ignored) {
                // Some OEMs throw if already binding
            }
        }
    }

    private static void toggleComponent(Context context) {
        try {
            PackageManager pm = context.getPackageManager();
            ComponentName cn = componentName(context);
            pm.setComponentEnabledSetting(
                    cn,
                    PackageManager.COMPONENT_ENABLED_STATE_DISABLED,
                    PackageManager.DONT_KILL_APP);
            pm.setComponentEnabledSetting(
                    cn,
                    PackageManager.COMPONENT_ENABLED_STATE_ENABLED,
                    PackageManager.DONT_KILL_APP);
        } catch (RuntimeException ignored) {
            // ignore OEM quirks
        }
    }

    /**
     * Scan notifications currently in the shade (not dismissed).
     *
     * @return number processed, or -1 if the listener is not connected
     */
    static int scanActiveNotifications(Context context) {
        if (!ensureConnected(context, 3500)) {
            return -1;
        }
        BankNotificationListenerService svc = instance;
        if (svc == null) {
            return -1;
        }
        StatusBarNotification[] active;
        try {
            active = svc.getActiveNotifications();
        } catch (SecurityException e) {
            return -1;
        }
        if (active == null) {
            return 0;
        }
        int n = 0;
        boolean anyQueued = false;
        for (StatusBarNotification sbn : active) {
            int r = process(svc, sbn, true);
            if (r > 0) {
                n += 1;
            }
            if (r == 2) {
                anyQueued = true;
            }
        }
        if (anyQueued) {
            NotificationInterceptPlugin.emitPendingAvailable();
        }
        return n;
    }

    /**
     * @param forceHistory write history even when capture is off (manual scan)
     * @return 0 skipped, 1 history only, 2 queued for JS
     */
    static int process(Context context, StatusBarNotification sbn, boolean forceHistory) {
        if (sbn == null || sbn.isOngoing()) {
            return 0;
        }

        String packageName = sbn.getPackageName();
        if (packageName == null || packageName.isEmpty()) {
            return 0;
        }

        Notification notification = sbn.getNotification();
        if (notification == null) {
            return 0;
        }
        Bundle extras = notification.extras;
        CharSequence titleCs = extras != null ? extras.getCharSequence(Notification.EXTRA_TITLE) : null;
        CharSequence textCs = extras != null ? extras.getCharSequence(Notification.EXTRA_TEXT) : null;
        CharSequence bigCs = extras != null ? extras.getCharSequence(Notification.EXTRA_BIG_TEXT) : null;

        String title = titleCs != null ? titleCs.toString().trim() : "";
        String text = textCs != null ? textCs.toString().trim() : "";
        String bigText = bigCs != null ? bigCs.toString().trim() : "";
        if (title.isEmpty() && text.isEmpty() && bigText.isEmpty()) {
            return 0;
        }

        long postTime = sbn.getPostTime();
        String dedupeKey = packageName + "|" + postTime + "|" + title + "|" + text + "|" + bigText;
        String channel = "push";
        String effectivePackage = packageName;
        if (MESSAGING_PACKAGES.contains(packageName)) {
            String bankPkg = NotificationInterceptStore.packageForSmsSender(context, title);
            if (bankPkg == null || bankPkg.isEmpty()) {
                String firstLine = text;
                int dot = text.indexOf('.');
                if (dot > 0) {
                    firstLine = text.substring(0, dot);
                }
                bankPkg = NotificationInterceptStore.packageForSmsSender(context, firstLine);
            }
            if (bankPkg != null && !bankPkg.isEmpty()) {
                effectivePackage = bankPkg;
                channel = "sms";
            }
        }
        boolean allowed =
                NotificationInterceptStore.allowedPackages(context).contains(effectivePackage);
        boolean captureOn = NotificationInterceptStore.isCaptureEnabled(context);

        if (forceHistory || captureOn) {
            try {
                JSONObject historyItem = new JSONObject();
                historyItem.put("packageName", packageName);
                historyItem.put("title", title);
                historyItem.put("text", text);
                historyItem.put("bigText", bigText);
                historyItem.put("postedAt", postTime);
                historyItem.put("dedupeKey", dedupeKey);
                historyItem.put("channel", channel);
                historyItem.put("inAllowlist", allowed);
                historyItem.put("queued", allowed && captureOn);
                if (!effectivePackage.equals(packageName)) {
                    historyItem.put("resolvedPackageName", effectivePackage);
                }
                // Primary: same prefs as pending queue (reliable on MIUI).
                NotificationInterceptStore.appendHistory(context, historyItem);
                // Legacy file — best effort.
                NotificationHistoryStore.append(context, historyItem);
            } catch (JSONException ignored) {
                // ignore history failures
            }
        }

        if (!allowed || !captureOn) {
            return forceHistory || captureOn ? 1 : 0;
        }

        try {
            JSONObject item = new JSONObject();
            item.put("packageName", effectivePackage);
            item.put("title", title);
            item.put("text", text);
            item.put("bigText", bigText);
            item.put("postedAt", postTime);
            item.put("dedupeKey", dedupeKey);
            item.put("channel", channel);
            NotificationInterceptStore.append(context, item);
            if (!forceHistory) {
                NotificationInterceptPlugin.emitPendingAvailable();
            }
        } catch (JSONException ignored) {
            // ignore
        }
        return 2;
    }
}
