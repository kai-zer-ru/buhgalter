package ru.kai_zer.buhgalter.widgets;

import android.content.Context;

import androidx.work.Constraints;
import androidx.work.ExistingPeriodicWorkPolicy;
import androidx.work.NetworkType;
import androidx.work.PeriodicWorkRequest;
import androidx.work.WorkManager;

import java.util.concurrent.TimeUnit;

public final class WidgetRefreshScheduler {
    private static final String UNIQUE = "buhgalter_widget_refresh";

    private WidgetRefreshScheduler() {}

    public static void ensurePeriodic(Context context) {
        Constraints constraints =
                new Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build();
        PeriodicWorkRequest request =
                new PeriodicWorkRequest.Builder(WidgetRefreshWorker.class, 60, TimeUnit.MINUTES)
                        .setConstraints(constraints)
                        .build();
        WorkManager.getInstance(context)
                .enqueueUniquePeriodicWork(UNIQUE, ExistingPeriodicWorkPolicy.KEEP, request);
    }

    public static void runOnce(Context context) {
        ensurePeriodic(context);
        WorkManager.getInstance(context)
                .enqueue(new androidx.work.OneTimeWorkRequest.Builder(WidgetRefreshWorker.class).build());
    }
}
