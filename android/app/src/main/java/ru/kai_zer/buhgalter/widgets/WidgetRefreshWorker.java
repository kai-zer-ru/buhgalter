package ru.kai_zer.buhgalter.widgets;

import android.content.Context;
import android.content.SharedPreferences;

import androidx.annotation.NonNull;
import androidx.work.Worker;
import androidx.work.WorkerParameters;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.IOException;
import java.security.SecureRandom;
import java.security.cert.X509Certificate;
import java.text.DecimalFormat;
import java.text.DecimalFormatSymbols;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.concurrent.TimeUnit;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;

public class WidgetRefreshWorker extends Worker {
    public WidgetRefreshWorker(@NonNull Context context, @NonNull WorkerParameters params) {
        super(context, params);
    }

    @NonNull
    @Override
    public Result doWork() {
        Context ctx = getApplicationContext();
        if (!WidgetSnapshotStore.hasAuth(ctx)) {
            return Result.success();
        }
        String base = WidgetSnapshotStore.getBaseUrl(ctx).replaceAll("/+$", "");
        String token = WidgetSnapshotStore.getToken(ctx);
        boolean lock = WidgetSnapshotStore.isLockEnabled(ctx);
        try {
            OkHttpClient client = buildClient(ctx, base);
            JSONObject dashboard = getJson(client, base + "/api/v1/dashboard", token);
            JSONArray accounts = getJsonArray(client, base + "/api/v1/accounts?status=active", token);
            JSONObject budget = getJson(client, base + "/api/v1/budgets/summary", token);
            JSONArray credits = getJsonArray(client, base + "/api/v1/credits?status=active", token);
            JSONArray debts = getJsonArray(client, base + "/api/v1/debts?settled=false", token);
            JSONObject futurePage =
                    getJson(
                            client,
                            base + "/api/v1/transactions?kind=future&sort=date_asc&page=1&limit=10",
                            token);
            JSONArray future = futurePage.optJSONArray("data");
            if (future == null) future = new JSONArray();

            String currency = "RUB";
            JSONObject prev = WidgetSnapshotStore.getSnapshot(ctx);
            if (prev != null) currency = prev.optString("currency", "RUB");
            String language = prev != null ? prev.optString("language", "ru") : "ru";

            JSONObject snapshot =
                    buildSnapshot(dashboard, accounts, budget, credits, debts, future, currency, language);
            WidgetSnapshotStore.publish(ctx, base, token, lock, snapshot.toString());
            WidgetUpdater.updateAll(ctx);
            return Result.success();
        } catch (Exception e) {
            return Result.retry();
        }
    }

    static JSONObject buildSnapshot(
            JSONObject dashboard,
            JSONArray accounts,
            JSONObject budget,
            JSONArray credits,
            JSONArray debts,
            JSONArray future,
            String currency,
            String language)
            throws Exception {
        long total = dashboard.optLong("total_balance", 0);
        long forecast = dashboard.optLong("total_forecast", 0);
        JSONObject snap = new JSONObject();
        snap.put("updated_at", java.time.Instant.now().toString());
        snap.put("currency", currency);
        snap.put("language", language);
        snap.put("total_balance_display", formatMoney(total, currency));
        snap.put("total_forecast_display", formatMoney(forecast, currency));
        snap.put("show_forecast", total != forecast);
        JSONObject cards = dashboard.optJSONObject("credit_cards_summary");
        if (cards != null) {
            snap.put("credit_cards_display", formatMoney(cards.optLong("total_balance", 0), currency));
        } else {
            snap.put("credit_cards_display", JSONObject.NULL);
        }
        snap.put("budget", pickBudget(budget.optJSONArray("items")));
        snap.put("upcoming", buildUpcoming(credits, debts, future, currency));
        JSONArray accountsOut = new JSONArray();
        if (accounts != null) {
            for (int i = 0; i < accounts.length(); i++) {
                JSONObject a = accounts.getJSONObject(i);
                JSONObject item = new JSONObject();
                item.put("id", a.optString("id"));
                item.put("name", a.optString("name"));
                item.put("balance_display", a.optString("balance_display"));
                item.put("is_primary", a.optBoolean("is_primary", false));
                accountsOut.put(item);
            }
        }
        snap.put("accounts", accountsOut);
        return snap;
    }

    private static Object pickBudget(JSONArray items) throws Exception {
        if (items == null || items.length() == 0) return JSONObject.NULL;
        JSONObject all = null;
        JSONObject top = null;
        int topPct = -1;
        for (int i = 0; i < items.length(); i++) {
            JSONObject it = items.getJSONObject(i);
            String scope = it.optString("scope");
            if ("all_expense".equals(scope)) {
                all = it;
            } else {
                int pct = it.optInt("percent", 0);
                if (pct > topPct) {
                    topPct = pct;
                    top = it;
                }
            }
        }
        JSONObject pick = all != null ? all : top;
        if (pick == null) return JSONObject.NULL;
        JSONObject out = new JSONObject();
        out.put("name", pick.optString("name"));
        out.put("spent_display", pick.optString("spent_display"));
        out.put("planned_display", pick.optString("planned_display"));
        out.put("remaining_display", pick.optString("remaining_display"));
        out.put("percent", pick.optInt("percent", 0));
        out.put("status", pick.optString("status", "ok"));
        return out;
    }

    private static JSONArray buildUpcoming(
            JSONArray credits, JSONArray debts, JSONArray future, String currency) throws Exception {
        List<JSONObject> list = new ArrayList<>();
        if (credits != null) {
            for (int i = 0; i < credits.length(); i++) {
                JSONObject c = credits.getJSONObject(i);
                if (!"active".equals(c.optString("status"))) continue;
                String date = c.optString("next_payment_date", "");
                if (date.isEmpty() || "null".equals(date)) continue;
                JSONObject item = new JSONObject();
                item.put("kind", "credit");
                item.put("id", c.optString("id"));
                String name = c.optString("name", "").trim();
                item.put("title", name.isEmpty() ? "Credit" : name);
                item.put("subtitle", c.optString("debit_account_name", ""));
                item.put("date", date);
                if (c.has("next_payment_amount") && !c.isNull("next_payment_amount")) {
                    item.put("amount_display", formatMoney(c.optLong("next_payment_amount"), currency));
                } else {
                    item.put("amount_display", c.optString("monthly_payment_display", ""));
                }
                item.put("route", "/credits/" + c.optString("id"));
                list.add(item);
            }
        }
        if (debts != null) {
            for (int i = 0; i < debts.length(); i++) {
                JSONObject d = debts.getJSONObject(i);
                if (d.optBoolean("is_settled", false)) continue;
                String date = d.optString("due_date", "");
                if (date.isEmpty()) continue;
                JSONObject item = new JSONObject();
                item.put("kind", "debt");
                item.put("id", d.optString("id"));
                item.put("title", d.optString("debtor_name", ""));
                item.put(
                        "subtitle",
                        "borrowed".equals(d.optString("direction")) ? "i_owe" : "owed_to_me");
                item.put("date", date);
                item.put("amount_display", d.optString("amount_display", ""));
                item.put("route", "/debtors/" + d.optString("debtor_id"));
                list.add(item);
            }
        }
        if (future != null) {
            for (int i = 0; i < future.length(); i++) {
                JSONObject tx = future.getJSONObject(i);
                JSONObject item = new JSONObject();
                item.put("kind", "future");
                item.put("id", tx.optString("id"));
                String title = tx.optString("description", "").trim();
                if (title.isEmpty()) title = tx.optString("category_name", "Payment");
                item.put("title", title);
                item.put("subtitle", tx.optString("account_name", ""));
                item.put("date", tx.optString("transaction_date", ""));
                item.put("amount_display", tx.optString("amount_display", ""));
                item.put("route", "/transactions");
                list.add(item);
            }
        }
        Collections.sort(
                list,
                Comparator.comparingLong(
                        o -> {
                            try {
                                return java.time.Instant.parse(normalizeDate(o.optString("date"))).toEpochMilli();
                            } catch (Exception e) {
                                return Long.MAX_VALUE;
                            }
                        }));
        JSONArray out = new JSONArray();
        for (int i = 0; i < Math.min(5, list.size()); i++) out.put(list.get(i));
        return out;
    }

    private static String normalizeDate(String raw) {
        if (raw == null || raw.isEmpty()) return "9999-12-31T00:00:00Z";
        if (raw.length() == 10) return raw + "T00:00:00Z";
        return raw;
    }

    static String formatMoney(long cents, String currency) {
        DecimalFormatSymbols symbols = new DecimalFormatSymbols(Locale.US);
        DecimalFormat df = new DecimalFormat("#,##0.00", symbols);
        return df.format(cents / 100.0) + " " + currency;
    }

    private static JSONObject getJson(OkHttpClient client, String url, String token) throws IOException {
        Request request =
                new Request.Builder()
                        .url(url)
                        .header("Authorization", "Bearer " + token)
                        .header("Accept", "application/json")
                        .get()
                        .build();
        try (Response response = client.newCall(request).execute()) {
            if (!response.isSuccessful() || response.body() == null) {
                throw new IOException("HTTP " + response.code());
            }
            return new JSONObject(response.body().string());
        } catch (IOException e) {
            throw e;
        } catch (Exception e) {
            throw new IOException(e);
        }
    }

    private static JSONArray getJsonArray(OkHttpClient client, String url, String token)
            throws IOException {
        Request request =
                new Request.Builder()
                        .url(url)
                        .header("Authorization", "Bearer " + token)
                        .header("Accept", "application/json")
                        .get()
                        .build();
        try (Response response = client.newCall(request).execute()) {
            if (!response.isSuccessful() || response.body() == null) {
                throw new IOException("HTTP " + response.code());
            }
            return new JSONArray(response.body().string());
        } catch (IOException e) {
            throw e;
        } catch (Exception e) {
            throw new IOException(e);
        }
    }

    private static OkHttpClient buildClient(Context context, String baseUrl) {
        boolean skip = isTrustedOrigin(context, baseUrl) || baseUrl.startsWith("http://");
        OkHttpClient.Builder builder =
                new OkHttpClient.Builder()
                        .connectTimeout(15, TimeUnit.SECONDS)
                        .readTimeout(20, TimeUnit.SECONDS);
        if (skip && baseUrl.startsWith("https://")) {
            try {
                TrustManager[] trustAll =
                        new TrustManager[] {
                            new X509TrustManager() {
                                public void checkClientTrusted(X509Certificate[] chain, String authType) {}

                                public void checkServerTrusted(X509Certificate[] chain, String authType) {}

                                public X509Certificate[] getAcceptedIssuers() {
                                    return new X509Certificate[0];
                                }
                            }
                        };
                SSLContext ssl = SSLContext.getInstance("TLS");
                ssl.init(null, trustAll, new SecureRandom());
                builder.sslSocketFactory(ssl.getSocketFactory(), (X509TrustManager) trustAll[0]);
                builder.hostnameVerifier((hostname, session) -> true);
            } catch (Exception ignored) {
            }
        }
        return builder.build();
    }

    private static boolean isTrustedOrigin(Context context, String baseUrl) {
        try {
            SharedPreferences prefs =
                    context.getSharedPreferences("buhgalter_ssl_trusted_origins", Context.MODE_PRIVATE);
            Set<String> origins = prefs.getStringSet("origins", Collections.emptySet());
            String origin = baseUrl;
            int scheme = origin.indexOf("://");
            if (scheme >= 0) {
                int slash = origin.indexOf('/', scheme + 3);
                origin = slash > 0 ? origin.substring(0, slash) : origin;
            }
            return origins != null && origins.contains(origin);
        } catch (Exception e) {
            return false;
        }
    }
}
