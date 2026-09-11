package ru.kai_zer.buhgalter;

import android.app.Application;
import android.content.Context;

import ru.kai_zer.buhgalter.widgets.WidgetSnapshotStore;
import ru.kai_zer.buhgalter.widgets.WidgetUpdater;

/**
 * Restore-guard after uninstall. Must not run in {@code attachBaseContext}:
 * {@code getExternalFilesDir} before attach can block for tens of seconds on HyperOS.
 */
public class BuhgalterApp extends Application {
    @Override
    public void onCreate() {
        super.onCreate();
        if (CloneGuard.isUnsupportedClone(this)) {
            return;
        }
        InstallIntegrity.ensureFreshInstall(this);
        final Context app = getApplicationContext();
        new Thread(
                () -> {
                    WidgetSnapshotStore.warmup(app);
                    WidgetUpdater.updateAll(app);
                },
                "widget-store-warmup")
                .start();
    }
}
