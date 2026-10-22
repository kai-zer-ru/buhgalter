package ru.kai_zer.buhgalter.widgets;

import android.content.Context;
import android.content.SharedPreferences;

public final class AccountWidgetPrefs {
    private static final String PREFS = "buhgalter_account_widget";

    private AccountWidgetPrefs() {}

    private static SharedPreferences prefs(Context context) {
        return context.getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    public static void setAccountId(Context context, int appWidgetId, String accountId) {
        prefs(context).edit().putString("account_" + appWidgetId, accountId).apply();
    }

    public static String getAccountId(Context context, int appWidgetId) {
        return prefs(context).getString("account_" + appWidgetId, "");
    }

    public static void delete(Context context, int appWidgetId) {
        prefs(context).edit().remove("account_" + appWidgetId).apply();
    }
}
