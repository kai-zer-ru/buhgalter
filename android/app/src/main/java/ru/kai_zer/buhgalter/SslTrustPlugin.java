package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;
import android.util.Base64;
import android.util.Log;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.net.URI;
import java.security.SecureRandom;
import java.security.cert.X509Certificate;
import java.util.HashSet;
import java.util.Iterator;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import okhttp3.MediaType;
import okhttp3.MultipartBody;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;

import org.json.JSONObject;

@CapacitorPlugin(name = "SslTrust")
public class SslTrustPlugin extends Plugin {

    private static final String PREFS = "buhgalter_ssl_trusted_origins";
    private static final String KEY_ORIGINS = "origins";
    // CapacitorPlugins is a single HandlerThread — blocking OkHttp there serializes all plugins.
    private static final ExecutorService HTTP_EXECUTOR = Executors.newCachedThreadPool();

    private SharedPreferences prefs() {
        return getContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    @PluginMethod
    public void setTrustedOrigins(PluginCall call) {
        try {
            JSArray arr = call.getArray("origins");
            Set<String> set = new HashSet<>();
            if (arr != null) {
                for (int i = 0; i < arr.length(); i++) {
                    String origin = arr.getString(i);
                    if (origin != null && !origin.isEmpty()) {
                        set.add(origin);
                    }
                }
            }
            prefs().edit().putStringSet(KEY_ORIGINS, set).apply();
            call.resolve();
        } catch (Exception e) {
            call.reject(e.getMessage());
        }
    }

    @PluginMethod
    public void request(PluginCall call) {
        String url = call.getString("url");
        if (url == null || url.isEmpty()) {
            call.reject("url required");
            return;
        }

        String method = call.getString("method", "GET").toUpperCase();
        boolean allowUntrusted = Boolean.TRUE.equals(call.getBoolean("allowUntrusted", false));
        String origin = originFromUrl(url);
        boolean skipVerify = allowUntrusted || isTrustedOrigin(origin);
        JSObject headersObj = call.getObject("headers");
        String bodyStr = call.getString("body");
        // One file as flat strings: a nested multipart array often arrives empty on Android.
        String imageBase64 = call.getString("imageBase64");
        String imageName = call.getString("imageName", "receipt.jpg");
        String imageMime = call.getString("imageMime", "image/jpeg");
        boolean flatImage = imageBase64 != null && !imageBase64.isEmpty();
        JSArray multipartArr = call.getArray("multipart");
        boolean hasMultipart =
                flatImage || (multipartArr != null && multipartArr.length() > 0);

        HTTP_EXECUTOR.execute(
                () -> {
                    try {
                        // Multipart matches the JS budget in ssl-trust.ts (photo upload).
                        OkHttpClient client =
                                buildClient(skipVerify, hasMultipart ? 180 : 12);
                        Request.Builder builder = new Request.Builder().url(url);

                        if (headersObj != null) {
                            Iterator<String> keys = headersObj.keys();
                            while (keys.hasNext()) {
                                String key = keys.next();
                                String value = headersObj.getString(key);
                                if (value != null) {
                                    if (hasMultipart
                                            && "content-type".equalsIgnoreCase(key)) {
                                        continue;
                                    }
                                    builder.addHeader(key, value);
                                }
                            }
                        }

                        if ("GET".equals(method) || "HEAD".equals(method)) {
                            builder.method(method, null);
                        } else if (flatImage) {
                            byte[] data = Base64.decode(imageBase64, Base64.DEFAULT);
                            MediaType mt =
                                    imageMime != null && !imageMime.isEmpty()
                                            ? MediaType.parse(imageMime)
                                            : MediaType.parse("image/jpeg");
                            MultipartBody body =
                                    new MultipartBody.Builder()
                                            .setType(MultipartBody.FORM)
                                            .addFormDataPart(
                                                    "image",
                                                    imageName != null && !imageName.isEmpty()
                                                            ? imageName
                                                            : "receipt.jpg",
                                                    RequestBody.create(data, mt))
                                            .build();
                            builder.method(method, body);
                        } else if (hasMultipart) {
                            MultipartBody.Builder mpBuilder =
                                    new MultipartBody.Builder().setType(MultipartBody.FORM);
                            for (int i = 0; i < multipartArr.length(); i++) {
                                JSONObject raw = multipartArr.optJSONObject(i);
                                if (raw == null) continue;
                                JSObject part = JSObject.fromJSONObject(raw);
                                String name = part.getString("name");
                                if (name == null || name.isEmpty()) continue;
                                String filename = part.getString("filename");
                                String base64 = part.getString("base64");
                                if (filename != null && base64 != null) {
                                    byte[] data = Base64.decode(base64, Base64.DEFAULT);
                                    String mime = part.getString("mime");
                                    MediaType mt =
                                            mime != null && !mime.isEmpty()
                                                    ? MediaType.parse(mime)
                                                    : MediaType.parse("application/octet-stream");
                                    mpBuilder.addFormDataPart(
                                            name, filename, RequestBody.create(data, mt));
                                } else {
                                    String value = part.getString("value");
                                    mpBuilder.addFormDataPart(
                                            name, value != null ? value : "");
                                }
                            }
                            builder.method(method, mpBuilder.build());
                        } else {
                            RequestBody body =
                                    bodyStr != null
                                            ? RequestBody.create(
                                                    bodyStr, MediaType.parse("application/json"))
                                            : RequestBody.create(
                                                    "", MediaType.parse("application/json"));
                            builder.method(method, body);
                        }

                        try (Response response = client.newCall(builder.build()).execute()) {
                            JSObject ret = new JSObject();
                            ret.put("status", response.code());
                            ret.put("ok", response.isSuccessful());
                            ret.put(
                                    "body",
                                    response.body() != null ? response.body().string() : "");
                            call.resolve(ret);
                        }
                    } catch (javax.net.ssl.SSLHandshakeException
                            | javax.net.ssl.SSLPeerUnverifiedException e) {
                        resolveError(call, "SSL_CERTIFICATE", e.getMessage());
                    } catch (Exception e) {
                        Log.e("SslTrust", "request failed", e);
                        String msg = e.getMessage() != null ? e.getMessage() : "request failed";
                        String lower = msg.toLowerCase();
                        if (lower.contains("certificate")
                                || lower.contains("ssl")
                                || lower.contains("trust")) {
                            resolveError(call, "SSL_CERTIFICATE", msg);
                        } else {
                            resolveError(call, "UNREACHABLE", msg);
                        }
                    }
                });
    }

    private void resolveError(PluginCall call, String code, String message) {
        JSObject ret = new JSObject();
        ret.put("errorCode", code);
        ret.put("message", message != null ? message : code);
        call.resolve(ret);
    }

    private boolean isTrustedOrigin(String origin) {
        if (origin == null || origin.isEmpty()) return false;
        Set<String> set = prefs().getStringSet(KEY_ORIGINS, new HashSet<>());
        return set.contains(origin);
    }

    private static String originFromUrl(String urlStr) {
        URI uri = URI.create(urlStr);
        String scheme = uri.getScheme();
        String host = uri.getHost();
        if (scheme == null || host == null) return urlStr;
        int port = uri.getPort();
        if (port > 0) {
            return scheme + "://" + host + ":" + port;
        }
        return scheme + "://" + host;
    }

    private OkHttpClient buildClient(boolean trustAll, int timeoutSec) {
        OkHttpClient.Builder builder =
                new OkHttpClient.Builder()
                        .connectTimeout(timeoutSec, TimeUnit.SECONDS)
                        .readTimeout(timeoutSec, TimeUnit.SECONDS)
                        .writeTimeout(timeoutSec, TimeUnit.SECONDS);
        if (!trustAll) {
            return builder.build();
        }
        try {
            TrustManager[] trustManagers =
                    new TrustManager[] {
                        new X509TrustManager() {
                            @Override
                            public void checkClientTrusted(X509Certificate[] chain, String authType) {}

                            @Override
                            public void checkServerTrusted(X509Certificate[] chain, String authType) {}

                            @Override
                            public X509Certificate[] getAcceptedIssuers() {
                                return new X509Certificate[0];
                            }
                        }
                    };
            SSLContext sslContext = SSLContext.getInstance("TLS");
            sslContext.init(null, trustManagers, new SecureRandom());
            builder.sslSocketFactory(sslContext.getSocketFactory(), (X509TrustManager) trustManagers[0]);
            builder.hostnameVerifier((hostname, session) -> true);
        } catch (Exception ignored) {
            // fall back to default client
        }
        return builder.build();
    }
}
