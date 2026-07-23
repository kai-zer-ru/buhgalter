package ru.kai_zer.buhgalter.widgets;

import android.appwidget.AppWidgetManager;
import android.appwidget.AppWidgetProvider;
import android.content.Context;
import android.widget.RemoteViews;

import ru.kai_zer.buhgalter.R;

public class UpcomingWidgetProvider extends AppWidgetProvider {
    @Override
    public void onUpdate(Context context, AppWidgetManager appWidgetManager, int[] appWidgetIds) {
        for (int id : appWidgetIds) {
            RemoteViews views = new RemoteViews(context.getPackageName(), R.layout.widget_upcoming);
            WidgetViews.bindUpcoming(context, views);
            appWidgetManager.updateAppWidget(id, views);
        }
        WidgetRefreshScheduler.runOnce(context);
    }

    @Override
    public void onEnabled(Context context) {
        WidgetRefreshScheduler.ensurePeriodic(context);
    }
}
