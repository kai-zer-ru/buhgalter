package ru.kai_zer.buhgalter.widgets;

import android.content.Context;
import android.view.View;
import android.widget.RemoteViews;

import org.json.JSONArray;
import org.json.JSONObject;

import ru.kai_zer.buhgalter.R;

final class WidgetViews {
    private WidgetViews() {}

    static boolean needLogin(Context context) {
        return !WidgetSnapshotStore.hasAuth(context);
    }

    static boolean amountsHidden(Context context) {
        return WidgetSnapshotStore.isLockEnabled(context);
    }

    static void bindQuickActions(Context context, RemoteViews views) {
        views.setOnClickPendingIntent(
                R.id.widget_action_expense, WidgetDeepLinks.open(context, 101, "/transactions/new?type=expense"));
        views.setOnClickPendingIntent(
                R.id.widget_action_income, WidgetDeepLinks.open(context, 102, "/transactions/new?type=income"));
        views.setOnClickPendingIntent(
                R.id.widget_action_transfer, WidgetDeepLinks.open(context, 103, "/transfers/new"));
    }

    static void bindBalance(Context context, RemoteViews views) {
        views.setOnClickPendingIntent(R.id.widget_root, WidgetDeepLinks.open(context, 201, "/"));
        if (needLogin(context)) {
            views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_balance_title));
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_need_login));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_tertiary, View.GONE);
            return;
        }
        if (amountsHidden(context)) {
            views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_balance_title));
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_locked));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_tertiary, View.GONE);
            return;
        }
        JSONObject snap = WidgetSnapshotStore.getSnapshot(context);
        if (snap == null) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_need_login));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_tertiary, View.GONE);
            return;
        }
        views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_balance_title));
        views.setTextViewText(R.id.widget_primary, snap.optString("total_balance_display", "—"));
        if (snap.optBoolean("show_forecast", false)) {
            views.setViewVisibility(R.id.widget_secondary, View.VISIBLE);
            views.setTextViewText(
                    R.id.widget_secondary,
                    context.getString(R.string.widget_with_plans, snap.optString("total_forecast_display", "")));
        } else {
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
        }
        if (!snap.isNull("credit_cards_display")) {
            views.setViewVisibility(R.id.widget_tertiary, View.VISIBLE);
            views.setTextViewText(
                    R.id.widget_tertiary,
                    context.getString(R.string.widget_credit_cards, snap.optString("credit_cards_display", "")));
        } else {
            views.setViewVisibility(R.id.widget_tertiary, View.GONE);
        }
    }

    static void bindBudget(Context context, RemoteViews views) {
        views.setOnClickPendingIntent(R.id.widget_root, WidgetDeepLinks.open(context, 301, "/budget"));
        views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_budget_title));
        if (needLogin(context)) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_need_login));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_progress, View.GONE);
            return;
        }
        if (amountsHidden(context)) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_locked));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_progress, View.GONE);
            return;
        }
        JSONObject snap = WidgetSnapshotStore.getSnapshot(context);
        JSONObject budget = snap != null ? snap.optJSONObject("budget") : null;
        if (budget == null) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_budget_empty));
            views.setViewVisibility(R.id.widget_secondary, View.GONE);
            views.setViewVisibility(R.id.widget_progress, View.GONE);
            return;
        }
        views.setTextViewText(R.id.widget_primary, budget.optString("name", ""));
        views.setViewVisibility(R.id.widget_secondary, View.VISIBLE);
        views.setTextViewText(
                R.id.widget_secondary,
                budget.optString("spent_display", "") + " / " + budget.optString("planned_display", ""));
        views.setViewVisibility(R.id.widget_progress, View.VISIBLE);
        int pct = Math.max(0, Math.min(100, budget.optInt("percent", 0)));
        views.setProgressBar(R.id.widget_progress, 100, pct, false);
    }

    static void bindUpcoming(Context context, RemoteViews views) {
        views.setOnClickPendingIntent(R.id.widget_root, WidgetDeepLinks.open(context, 401, "/"));
        views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_upcoming_title));
        int[] rowIds = {
            R.id.widget_row1, R.id.widget_row2, R.id.widget_row3, R.id.widget_row4, R.id.widget_row5
        };
        int[] titleIds = {
            R.id.widget_row1_title,
            R.id.widget_row2_title,
            R.id.widget_row3_title,
            R.id.widget_row4_title,
            R.id.widget_row5_title
        };
        int[] amountIds = {
            R.id.widget_row1_amount,
            R.id.widget_row2_amount,
            R.id.widget_row3_amount,
            R.id.widget_row4_amount,
            R.id.widget_row5_amount
        };
        for (int rowId : rowIds) views.setViewVisibility(rowId, View.GONE);

        if (needLogin(context)) {
            views.setViewVisibility(R.id.widget_empty, View.VISIBLE);
            views.setTextViewText(R.id.widget_empty, context.getString(R.string.widget_need_login));
            return;
        }
        JSONObject snap = WidgetSnapshotStore.getSnapshot(context);
        JSONArray upcoming = snap != null ? snap.optJSONArray("upcoming") : null;
        if (upcoming == null || upcoming.length() == 0) {
            views.setViewVisibility(R.id.widget_empty, View.VISIBLE);
            views.setTextViewText(R.id.widget_empty, context.getString(R.string.widget_upcoming_empty));
            return;
        }
        views.setViewVisibility(R.id.widget_empty, View.GONE);
        boolean hide = amountsHidden(context);
        int n = Math.min(5, upcoming.length());
        for (int i = 0; i < n; i++) {
            JSONObject item = upcoming.optJSONObject(i);
            if (item == null) continue;
            views.setViewVisibility(rowIds[i], View.VISIBLE);
            String date = item.optString("date", "");
            if (date.length() >= 10) date = date.substring(0, 10);
            views.setTextViewText(titleIds[i], date + " · " + item.optString("title", ""));
            views.setTextViewText(
                    amountIds[i], hide ? context.getString(R.string.widget_amount_hidden) : item.optString("amount_display", ""));
            String route = item.optString("route", "/");
            views.setOnClickPendingIntent(rowIds[i], WidgetDeepLinks.open(context, 410 + i, route));
        }
    }

    static void bindAccount(Context context, RemoteViews views, int appWidgetId) {
        String accountId = AccountWidgetPrefs.getAccountId(context, appWidgetId);
        views.setTextViewText(R.id.widget_title, context.getString(R.string.widget_account_title));
        if (needLogin(context)) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_need_login));
            views.setOnClickPendingIntent(R.id.widget_root, WidgetDeepLinks.open(context, 500 + appWidgetId, "/"));
            return;
        }
        JSONObject snap = WidgetSnapshotStore.getSnapshot(context);
        JSONObject account = findAccount(snap, accountId);
        if (account == null) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_account_pick));
            views.setOnClickPendingIntent(R.id.widget_root, WidgetDeepLinks.open(context, 500 + appWidgetId, "/accounts"));
            return;
        }
        views.setTextViewText(R.id.widget_title, account.optString("name", ""));
        if (amountsHidden(context)) {
            views.setTextViewText(R.id.widget_primary, context.getString(R.string.widget_locked));
        } else {
            views.setTextViewText(R.id.widget_primary, account.optString("balance_display", "—"));
        }
        views.setOnClickPendingIntent(
                R.id.widget_root,
                WidgetDeepLinks.open(context, 500 + appWidgetId, "/accounts/" + account.optString("id")));
    }

    private static JSONObject findAccount(JSONObject snap, String accountId) {
        if (snap == null) return null;
        JSONArray accounts = snap.optJSONArray("accounts");
        if (accounts == null || accounts.length() == 0) return null;
        if (accountId != null && !accountId.isEmpty()) {
            for (int i = 0; i < accounts.length(); i++) {
                JSONObject a = accounts.optJSONObject(i);
                if (a != null && accountId.equals(a.optString("id"))) return a;
            }
        }
        for (int i = 0; i < accounts.length(); i++) {
            JSONObject a = accounts.optJSONObject(i);
            if (a != null && a.optBoolean("is_primary", false)) return a;
        }
        return accounts.optJSONObject(0);
    }
}
