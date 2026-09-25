package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;
import android.text.TextUtils;

import org.json.JSONObject;

import java.io.IOException;
import java.security.SecureRandom;
import java.security.cert.X509Certificate;
import java.util.Collections;
import java.util.Set;
import java.util.concurrent.TimeUnit;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;
import ru.kai_zer.buhgalter.widgets.WidgetSnapshotStore;

/** Native POST /api/v1/transactions for shade Accept (same auth as widgets). */
final class DraftNotifyHttp {
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    private DraftNotifyHttp() {}

    /**
     * @return true if the server accepted the create (2xx)
     */
    static boolean postTransaction(Context context, JSONObject draft) {
        if (!WidgetSnapshotStore.hasAuth(context) || draft == null) return false;
        String accountId = draft.optString("accountId", "");
        if (TextUtils.isEmpty(accountId)) return false;

        String base = WidgetSnapshotStore.getBaseUrl(context).replaceAll("/+$", "");
        String token = WidgetSnapshotStore.getToken(context);
        try {
            JSONObject body = new JSONObject();
            body.put("account_id", accountId);
            String type = "income".equals(draft.optString("type", "")) ? "income" : "expense";
            body.put("type", type);
            body.put("amount", draft.optString("amount", ""));
            String occurredAt = draft.optString("occurredAt", "");
            if (TextUtils.isEmpty(occurredAt)) {
                occurredAt = java.time.Instant.now().toString();
            }
            body.put("transaction_date", occurredAt);

            String categoryId = draft.optString("categoryId", "");
            if (!TextUtils.isEmpty(categoryId)) body.put("category_id", categoryId);
            String subcategoryId = draft.optString("subcategoryId", "");
            if (!TextUtils.isEmpty(subcategoryId)) body.put("subcategory_id", subcategoryId);

            String merchantId = draft.optString("merchantId", "");
            String merchantName = draft.optString("merchantName", "");
            if (!TextUtils.isEmpty(merchantId)) {
                body.put("merchant_id", merchantId);
            } else if ("expense".equals(type) && !TextUtils.isEmpty(merchantName)) {
                body.put("merchant_name", merchantName);
            }

            String description = draft.optString("description", "");
            if (TextUtils.isEmpty(description)
                    && "income".equals(type)
                    && TextUtils.isEmpty(merchantId)
                    && !TextUtils.isEmpty(merchantName)) {
                description = merchantName;
            }
            if (!TextUtils.isEmpty(description)) {
                body.put("description", description);
            }

            return postJson(context, base, token, "/api/v1/transactions", body);
        } catch (org.json.JSONException e) {
            return false;
        }
    }

    static boolean postTransfer(Context context, JSONObject draft) {
        if (!WidgetSnapshotStore.hasAuth(context) || draft == null) return false;
        String fromAccount = draft.optString("fromAccountId", "");
        String toAccount = draft.optString("toAccountId", "");
        if (TextUtils.isEmpty(fromAccount) || TextUtils.isEmpty(toAccount)) return false;

        String base = WidgetSnapshotStore.getBaseUrl(context).replaceAll("/+$", "");
        String token = WidgetSnapshotStore.getToken(context);
        try {
            JSONObject body = new JSONObject();
            body.put("from_account_id", fromAccount);
            body.put("to_account_id", toAccount);
            body.put("amount", draft.optString("amount", ""));
            String occurredAt = draft.optString("occurredAt", "");
            if (TextUtils.isEmpty(occurredAt)) {
                occurredAt = java.time.Instant.now().toString();
            }
            body.put("transaction_date", occurredAt);
            return postJson(context, base, token, "/api/v1/transfers", body);
        } catch (org.json.JSONException e) {
            return false;
        }
    }

    private static boolean postJson(
            Context context, String base, String token, String path, JSONObject body) {
        try {
            OkHttpClient client = buildClient(context, base);
            Request request =
                    new Request.Builder()
                            .url(base + path)
                            .header("Authorization", "Bearer " + token)
                            .header("Accept", "application/json")
                            .post(RequestBody.create(body.toString(), JSON))
                            .build();
            try (Response response = client.newCall(request).execute()) {
                return response.isSuccessful();
            }
        } catch (IOException | RuntimeException e) {
            return false;
        }
    }

    /** Payload for JS outbox when native POST fails / offline. */
    static JSONObject offlinePayloadFromDraft(JSONObject draft) {
        if (draft == null) return null;
        try {
            JSONObject body = new JSONObject();
            body.put("kind", "transaction");
            body.put("draftId", draft.optString("id", ""));
            body.put("account_id", draft.optString("accountId", ""));
            String type = "income".equals(draft.optString("type", "")) ? "income" : "expense";
            body.put("type", type);
            body.put("amount", draft.optString("amount", ""));
            String occurredAt = draft.optString("occurredAt", "");
            if (TextUtils.isEmpty(occurredAt)) {
                occurredAt = java.time.Instant.now().toString();
            }
            body.put("transaction_date", occurredAt);
            String categoryId = draft.optString("categoryId", "");
            if (!TextUtils.isEmpty(categoryId)) body.put("category_id", categoryId);
            String subcategoryId = draft.optString("subcategoryId", "");
            if (!TextUtils.isEmpty(subcategoryId)) body.put("subcategory_id", subcategoryId);
            String merchantId = draft.optString("merchantId", "");
            String merchantName = draft.optString("merchantName", "");
            if (!TextUtils.isEmpty(merchantId)) {
                body.put("merchant_id", merchantId);
            } else if ("expense".equals(type) && !TextUtils.isEmpty(merchantName)) {
                body.put("merchant_name", merchantName);
            }
            String description = draft.optString("description", "");
            if (TextUtils.isEmpty(description)
                    && "income".equals(type)
                    && TextUtils.isEmpty(merchantId)
                    && !TextUtils.isEmpty(merchantName)) {
                description = merchantName;
            }
            if (!TextUtils.isEmpty(description)) body.put("description", description);
            return body;
        } catch (org.json.JSONException e) {
            return null;
        }
    }

    static JSONObject offlinePayloadFromTransfer(JSONObject draft) {
        if (draft == null) return null;
        try {
            JSONObject body = new JSONObject();
            body.put("kind", "transfer");
            org.json.JSONArray ids = new org.json.JSONArray();
            String fromDraftId = draft.optString("fromDraftId", "");
            String toDraftId = draft.optString("toDraftId", "");
            if (!TextUtils.isEmpty(fromDraftId)) ids.put(fromDraftId);
            if (!TextUtils.isEmpty(toDraftId)) ids.put(toDraftId);
            body.put("draftIds", ids);
            body.put("from_account_id", draft.optString("fromAccountId", ""));
            body.put("to_account_id", draft.optString("toAccountId", ""));
            body.put("amount", draft.optString("amount", ""));
            String occurredAt = draft.optString("occurredAt", "");
            if (TextUtils.isEmpty(occurredAt)) {
                occurredAt = java.time.Instant.now().toString();
            }
            body.put("transaction_date", occurredAt);
            return body;
        } catch (org.json.JSONException e) {
            return null;
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
                                public void checkClientTrusted(
                                        X509Certificate[] chain, String authType) {}

                                public void checkServerTrusted(
                                        X509Certificate[] chain, String authType) {}

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
                    context.getSharedPreferences(
                            "buhgalter_ssl_trusted_origins", Context.MODE_PRIVATE);
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
