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
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;
import java.util.Set;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;

public class WidgetRefreshWorker extends Worker {
    /** Guard against stacked OneTime work / accidental re-entry storms. */
    private static final long MIN_REFRESH_INTERVAL_MS = 30_000L;
    private static final AtomicLong lastRefreshElapsedMs = new AtomicLong(0);

    public WidgetRefreshWorker(@NonNull Context context, @NonNull WorkerParameters params) {
        super(context, params);
    }

    @NonNull
    @Override
    public Result doWork() {
        long now = android.os.SystemClock.elapsedRealtime();
        long prev = lastRefreshElapsedMs.get();
        if (prev > 0 && now - prev < MIN_REFRESH_INTERVAL_MS) {
            return Result.success();
        }
        if (!lastRefreshElapsedMs.compareAndSet(prev, now)) {
            return Result.success();
        }

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
            // Optional modules — empty if feature is off or endpoint fails.
            JSONArray subscriptions = getJsonArrayOptional(client, base + "/api/v1/subscriptions", token);
            JSONArray recurring =
                    getJsonArrayOptional(client, base + "/api/v1/recurring-operations", token);

            String currency = "RUB";
            JSONObject prevSnap = WidgetSnapshotStore.getSnapshot(ctx);
            if (prevSnap != null) currency = prevSnap.optString("currency", "RUB");
            String language = prevSnap != null ? prevSnap.optString("language", "ru") : "ru";

            JSONObject snapshot =
                    buildSnapshot(
                            dashboard,
                            accounts,
                            budget,
                            credits,
                            debts,
                            future,
                            subscriptions,
                            recurring,
                            currency,
                            language);
            WidgetSnapshotStore.publish(ctx, base, token, lock, snapshot.toString());
            WidgetUpdater.updateAll(ctx);
            return Result.success();
        } catch (Exception e) {
            // Allow a quicker retry after a failed attempt.
            lastRefreshElapsedMs.set(0);
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
            JSONArray subscriptions,
            JSONArray recurring,
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
        long cashCents = sumBalanceByType(accounts, "cash");
        long bankCents = sumBalanceByType(accounts, "bank");
        long creditCents =
                cards != null
                        ? cards.optLong("total_balance", 0)
                        : sumBalanceByType(accounts, "credit_card");
        String cashDisplay = formatMoney(cashCents, currency);
        String bankDisplay = formatMoney(bankCents, currency);
        String creditDisplay = formatMoney(creditCents, currency);
        snap.put("cash_display", cashDisplay);
        snap.put("bank_display", bankDisplay);
        snap.put("credit_funds_display", creditDisplay);
        if (cards != null || creditCents != 0) {
            snap.put("credit_cards_display", creditDisplay);
        } else {
            snap.put("credit_cards_display", JSONObject.NULL);
        }
        snap.put("budget", pickBudget(budget.optJSONArray("items"), currency));
        snap.put("upcoming", buildUpcoming(credits, debts, future, subscriptions, recurring, currency));
        JSONArray accountsOut = new JSONArray();
        if (accounts != null) {
            for (int i = 0; i < accounts.length(); i++) {
                JSONObject a = accounts.getJSONObject(i);
                if (!"active".equals(a.optString("status", "active"))) continue;
                JSONObject item = new JSONObject();
                item.put("id", a.optString("id"));
                item.put("name", a.optString("name"));
                item.put("balance_display", formatMoney(a.optLong("balance", 0), currency));
                item.put("is_primary", a.optBoolean("is_primary", false));
                accountsOut.put(item);
            }
        }
        snap.put("accounts", accountsOut);
        return snap;
    }

    private static long sumBalanceByType(JSONArray accounts, String type) {
        if (accounts == null) return 0;
        long sum = 0;
        for (int i = 0; i < accounts.length(); i++) {
            JSONObject a = accounts.optJSONObject(i);
            if (a == null) continue;
            if (!"active".equals(a.optString("status", "active"))) continue;
            if (!type.equals(a.optString("type"))) continue;
            sum += a.optLong("balance", 0);
        }
        return sum;
    }

    private static Object pickBudget(JSONArray items, String currency) throws Exception {
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
        out.put("spent_display", formatMoney(pick.optLong("spent", 0), currency));
        out.put("planned_display", formatMoney(pick.optLong("planned", 0), currency));
        out.put("remaining_display", formatMoney(pick.optLong("remaining", 0), currency));
        out.put("percent", pick.optInt("percent", 0));
        out.put("status", pick.optString("status", "ok"));
        return out;
    }

    private static JSONArray buildUpcoming(
            JSONArray credits,
            JSONArray debts,
            JSONArray future,
            JSONArray subscriptions,
            JSONArray recurring,
            String currency)
            throws Exception {
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
                } else if (c.has("monthly_payment") && !c.isNull("monthly_payment")) {
                    item.put("amount_display", formatMoney(c.optLong("monthly_payment"), currency));
                } else {
                    item.put(
                            "amount_display",
                            formatMoneyDisplay(c.optString("monthly_payment_display", ""), currency));
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
                item.put("amount_display", formatMoney(d.optLong("amount", 0), currency));
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
                item.put("amount_display", formatMoney(tx.optLong("amount", 0), currency));
                item.put("route", "/transactions");
                list.add(item);
            }
        }
        if (subscriptions != null) {
            for (int i = 0; i < subscriptions.length(); i++) {
                JSONObject s = subscriptions.getJSONObject(i);
                if (!s.optBoolean("active", false)) continue;
                String date = s.optString("next_run_at", "");
                if (date.isEmpty() || "null".equals(date)) continue;
                JSONObject item = new JSONObject();
                item.put("kind", "subscription");
                item.put("id", s.optString("id"));
                String name = s.optString("name", "").trim();
                item.put("title", name.isEmpty() ? "Subscription" : name);
                item.put("subtitle", s.optString("account_name", ""));
                item.put("date", date);
                item.put("amount_display", formatMoney(s.optLong("amount", 0), currency));
                item.put("route", "/subscriptions");
                list.add(item);
            }
        }
        if (recurring != null) {
            for (int i = 0; i < recurring.length(); i++) {
                JSONObject r = recurring.getJSONObject(i);
                if (!r.optBoolean("active", false)) continue;
                String date = r.optString("next_run_at", "");
                if (date.isEmpty() || "null".equals(date)) continue;
                JSONObject item = new JSONObject();
                item.put("kind", "recurring");
                item.put("id", r.optString("id"));
                String title = r.optString("description", "").trim();
                if (title.isEmpty()) title = r.optString("category_name", "Recurring");
                item.put("title", title);
                item.put("subtitle", r.optString("account_name", ""));
                item.put("date", date);
                item.put("amount_display", formatMoney(r.optLong("amount", 0), currency));
                item.put("route", "/recurring-operations");
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
        boolean negative = cents < 0;
        long abs = Math.abs(cents);
        long whole = abs / 100;
        long frac = abs % 100;
        StringBuilder sb = new StringBuilder();
        if (negative) sb.append('-');
        sb.append(groupThousands(whole));
        sb.append('.');
        if (frac < 10) sb.append('0');
        sb.append(frac);
        sb.append(' ');
        sb.append(currencySymbol(currency));
        return sb.toString();
    }

    /** Re-format API `*_display` strings (e.g. {@code 1500.00}) like in-app money UI. */
    static String formatMoneyDisplay(String raw, String currency) {
        if (raw == null) return "";
        String cleaned =
                raw.trim()
                        .replace('\u00A0', ' ')
                        .replace(" ", "")
                        .replace("₽", "")
                        .replace(",", ".");
        cleaned = cleaned.replaceAll("(?i)[A-Z]{3}$", "").trim();
        if (cleaned.isEmpty()) return "";
        try {
            java.math.BigDecimal bd = new java.math.BigDecimal(cleaned);
            long cents =
                    bd.movePointRight(2).setScale(0, java.math.RoundingMode.HALF_UP).longValue();
            return formatMoney(cents, currency);
        } catch (Exception e) {
            return raw.trim();
        }
    }

    private static String currencySymbol(String currency) {
        if (currency != null && currency.equalsIgnoreCase("RUB")) return "₽";
        return currency != null && !currency.isEmpty() ? currency : "₽";
    }

    private static String groupThousands(long n) {
        String s = Long.toString(n);
        StringBuilder out = new StringBuilder(s.length() + s.length() / 3);
        int len = s.length();
        for (int i = 0; i < len; i++) {
            if (i > 0 && (len - i) % 3 == 0) out.append(' ');
            out.append(s.charAt(i));
        }
        return out.toString();
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

    private static JSONArray getJsonArrayOptional(OkHttpClient client, String url, String token) {
        try {
            return getJsonArray(client, url, token);
        } catch (Exception e) {
            return new JSONArray();
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
