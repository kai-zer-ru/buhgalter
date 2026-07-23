package ru.kai_zer.buhgalter.widgets;

import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;

import ru.kai_zer.buhgalter.MainActivity;

public final class WidgetDeepLinks {
    public static final String SCHEME = "ru.kai_zer.buhgalter";

    private WidgetDeepLinks() {}

    public static Uri uri(String pathAndQuery) {
        String path = pathAndQuery == null ? "/" : pathAndQuery;
        if (!path.startsWith("/")) path = "/" + path;
        return Uri.parse(SCHEME + "://" + path.substring(1));
    }

    public static PendingIntent open(Context context, int requestCode, String pathAndQuery) {
        Intent intent = new Intent(Intent.ACTION_VIEW, uri(pathAndQuery));
        intent.setClass(context, MainActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        return PendingIntent.getActivity(
                context,
                requestCode,
                intent,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }
}
