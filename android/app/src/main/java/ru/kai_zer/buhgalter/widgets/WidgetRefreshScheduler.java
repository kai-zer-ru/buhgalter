package ru.kai_zer.buhgalter.widgets;

import android.content.Context;

import androidx.work.Constraints;
import androidx.work.ExistingPeriodicWorkPolicy;
import androidx.work.ExistingWorkPolicy;
import androidx.work.NetworkType;
import androidx.work.OneTimeWorkRequest;
import androidx.work.PeriodicWorkRequest;
import androidx.work.WorkManager;

import java.util.concurrent.TimeUnit;

/**
 * Schedules background widget refreshes. Never call {@link #runOnce} from {@code onUpdate}:
 * Worker → {@link WidgetUpdater#updateAll} → {@code ACTION_APPWIDGET_UPDATE} → {@code onUpdate}
 * would enqueue forever and pin the CPU (EncryptedSharedPreferences + 6 HTTP calls).
 */
public final class WidgetRefreshScheduler {
    private static final String UNIQUE_PERIODIC = "buhgalter_widget_refresh";
    private static final String UNIQUE_ONCE = "buhgalter_widget_refresh_once";
    private static final String TAG_ONCE = "buhgalter_widget_refresh_once";

    private WidgetRefreshScheduler() {}

    public static void ensurePeriodic(Context context) {
        WorkManager wm = WorkManager.getInstance(context);
        // App uses WorkManager only for widgets. cancelAllWork clears the unbounded
        // OneTime backlog from the old onUpdate→runOnce loop (those had no unique name/tag).
        wm.cancelAllWork();

        Constraints constraints =
                new Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build();
        PeriodicWorkRequest request =
                new PeriodicWorkRequest.Builder(WidgetRefreshWorker.class, 60, TimeUnit.MINUTES)
                        .setConstraints(constraints)
                        .build();
        wm.enqueueUniquePeriodicWork(UNIQUE_PERIODIC, ExistingPeriodicWorkPolicy.KEEP, request);
    }

    /**
     * Network refresh once. Coalesced via unique work — concurrent callers do not stack.
     * Safe from {@code onEnabled} / configure / explicit refresh; not from {@code onUpdate}.
     */
    public static void runOnce(Context context) {
        ensurePeriodic(context);
        OneTimeWorkRequest request =
                new OneTimeWorkRequest.Builder(WidgetRefreshWorker.class).addTag(TAG_ONCE).build();
        WorkManager.getInstance(context)
                .enqueueUniqueWork(UNIQUE_ONCE, ExistingWorkPolicy.KEEP, request);
    }
}
