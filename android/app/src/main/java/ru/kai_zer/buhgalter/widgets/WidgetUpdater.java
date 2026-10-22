package ru.kai_zer.buhgalter.widgets;

import android.appwidget.AppWidgetManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;

public final class WidgetUpdater {
    private WidgetUpdater() {}

    private static final Class<?>[] PROVIDERS =
            new Class<?>[] {
                QuickActionsWidgetProvider.class,
                BalanceWidgetProvider.class,
                BudgetWidgetProvider.class,
                UpcomingWidgetProvider.class,
                AccountWidgetProvider.class
            };

    public static void updateAll(Context context) {
        AppWidgetManager manager = AppWidgetManager.getInstance(context);
        for (Class<?> provider : PROVIDERS) {
            ComponentName name = new ComponentName(context, provider);
            int[] ids = manager.getAppWidgetIds(name);
            if (ids == null || ids.length == 0) continue;
            Intent intent = new Intent(context, provider);
            intent.setAction(AppWidgetManager.ACTION_APPWIDGET_UPDATE);
            intent.putExtra(AppWidgetManager.EXTRA_APPWIDGET_IDS, ids);
            context.sendBroadcast(intent);
        }
    }
}
