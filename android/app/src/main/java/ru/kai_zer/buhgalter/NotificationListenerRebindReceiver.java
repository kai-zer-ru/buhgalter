package ru.kai_zer.buhgalter;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

/**
 * After boot / package replace, ask the system to bind the notification listener again
 * (needed on MIUI/HyperOS when the app process was killed).
 */
public class NotificationListenerRebindReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null || intent.getAction() == null) {
            return;
        }
        if (!NotificationInterceptStore.isCaptureEnabled(context)) {
            return;
        }
        final Context app = context.getApplicationContext();
        final PendingResult pending = goAsync();
        new Thread(
                        () -> {
                            try {
                                BankNotificationListenerService.ensureConnected(app, 5000);
                            } finally {
                                pending.finish();
                            }
                        },
                        "nls-boot-rebind")
                .start();
    }
}
