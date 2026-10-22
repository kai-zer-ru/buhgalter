package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Iterator;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Persists raw bank notification payloads until the WebView consumes them.
 */
final class NotificationInterceptStore {
    private static final String PREFS = "buhgalter.notification_intercept.v1";
    private static final String KEY_PENDING = "pending";
    /** Debug history in the same prefs file as the working pending queue. */
    private static final String KEY_HISTORY = "history";
    private static final String KEY_CAPTURE = "capture_enabled";
    /** JSON string array — avoid SharedPreferences StringSet update quirks. */
    private static final String KEY_PACKAGES_JSON = "allowed_packages_json";
    /** Legacy StringSet key; migrated on read. */
    private static final String KEY_PACKAGES_LEGACY = "allowed_packages";
    /**
     * JSON object: normalized SMS sender → primary bank packageName. Keep in sync with
     * {@code banks.ts} smsSenders.
     */
    private static final String KEY_SMS_SENDERS_JSON = "allowed_sms_senders_json";
    private static final int MAX_PENDING = 100;
    private static final int MAX_HISTORY = 80;
    private static final Object HISTORY_LOCK = new Object();

    /**
     * Fallback when WebView has not synced the allowlist yet (cold start / MIUI kill).
     * Keep in sync with {@code banks.ts} KNOWN_BANK_APPS + KNOWN_WALLET_APPS packages.
     */
    private static final Set<String> DEFAULT_BANK_PACKAGES =
            Collections.unmodifiableSet(
                    new HashSet<>(
                            Arrays.asList(
                                    "ru.sberbankmobile",
                                    "com.idamob.tinkoff.android",
                                    "ru.vtb24.mobilebanking.android",
                                    "ru.alfabank.mobile.android",
                                    "ru.gazprombank.android.mobilebank.app",
                                    "ru.raiffeisennews",
                                    "ru.rosbank.android",
                                    "ru.mkb.mobile",
                                    "ru.rshb.dbo",
                                    "com.openbank",
                                    "ru.sovcomcard.halva.v1",
                                    "ru.sovcombank.mobile",
                                    "ru.ftc.faktura.psb",
                                    "logo.com.mbanking",
                                    "ru.uralsib.mb",
                                    "ru.homecredit.mycredit",
                                    "ru.ozon.fintech.finance",
                                    "ru.ozon.app.android",
                                    "com.yandex.bank",
                                    "ru.wildberries.fintech",
                                    "com.wildberries.ru",
                                    "ru.otpbank.mobile",
                                    "ru.atb.mobilbank",
                                    "com.bssys.vbrrretail",
                                    "ru.nspk.mirpay",
                                    "ru.nspk.sbpay",
                                    "com.samsung.android.spay",
                                    "com.samsung.android.spaymini",
                                    "com.google.android.apps.walletnfcrelay",
                                    "com.huawei.wallet",
                                    "com.mipay.wallet",
                                    "com.xiaomi.payment",
                                    "ru.yandex.money")));

    /** Default sender → package map (mirror of banks.ts smsSenders). */
    private static final String DEFAULT_SMS_SENDERS_JSON =
            "{"
                    + "\"900\":\"ru.sberbankmobile\","
                    + "\"t-bank\":\"com.idamob.tinkoff.android\","
                    + "\"tbank\":\"com.idamob.tinkoff.android\","
                    + "\"tinkoff\":\"com.idamob.tinkoff.android\","
                    + "\"7555\":\"com.idamob.tinkoff.android\","
                    + "\"vtb\":\"ru.vtb24.mobilebanking.android\","
                    + "\"1000\":\"ru.vtb24.mobilebanking.android\","
                    + "\"alfabank\":\"ru.alfabank.mobile.android\","
                    + "\"alfa-bank\":\"ru.alfabank.mobile.android\","
                    + "\"2265\":\"ru.alfabank.mobile.android\","
                    + "\"gazprombank\":\"ru.gazprombank.android.mobilebank.app\","
                    + "\"raiffeisen\":\"ru.raiffeisennews\","
                    + "\"rosbank\":\"ru.rosbank.android\","
                    + "\"mkb\":\"ru.mkb.mobile\","
                    + "\"rshb\":\"ru.rshb.dbo\","
                    + "\"otkritie\":\"com.openbank\","
                    + "\"open\":\"com.openbank\","
                    + "\"sovcombank\":\"ru.sovcombank.mobile\","
                    + "\"halva\":\"ru.sovcomcard.halva.v1\","
                    + "\"psb\":\"ru.ftc.faktura.psb\","
                    + "\"uralsib\":\"ru.uralsib.mb\","
                    + "\"homecredit\":\"ru.homecredit.mycredit\","
                    + "\"ozon\":\"ru.ozon.fintech.finance\","
                    + "\"yandex\":\"com.yandex.bank\","
                    + "\"wb\":\"ru.wildberries.fintech\","
                    + "\"wbbank\":\"ru.wildberries.fintech\","
                    + "\"otpbank\":\"ru.otpbank.mobile\","
                    + "\"atb\":\"ru.atb.mobilbank\","
                    + "\"vbrr\":\"com.bssys.vbrrretail\","
                    + "\"вбрр\":\"com.bssys.vbrrretail\""
                    + "}";

    private NotificationInterceptStore() {}

    private static SharedPreferences prefs(Context context) {
        return context.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static boolean isCaptureEnabled(Context context) {
        return prefs(context).getBoolean(KEY_CAPTURE, false);
    }

    static void setCaptureEnabled(Context context, boolean enabled) {
        prefs(context).edit().putBoolean(KEY_CAPTURE, enabled).commit();
    }

    static Set<String> allowedPackages(Context context) {
        SharedPreferences p = prefs(context);
        String json = p.getString(KEY_PACKAGES_JSON, null);
        if (json != null) {
            Set<String> fromJson = packagesFromJson(json);
            // Empty JSON means "not synced" or wiped — use built-in bank list so capture works.
            return fromJson.isEmpty() ? new HashSet<>(DEFAULT_BANK_PACKAGES) : fromJson;
        }
        // One-time migrate from legacy StringSet storage.
        Set<String> legacy = p.getStringSet(KEY_PACKAGES_LEGACY, null);
        if (legacy != null && !legacy.isEmpty()) {
            Set<String> copy = new HashSet<>(legacy);
            setAllowedPackages(context, copy);
            return copy;
        }
        return new HashSet<>(DEFAULT_BANK_PACKAGES);
    }

    static void setAllowedPackages(Context context, Set<String> packages) {
        JSONArray arr = new JSONArray();
        if (packages != null) {
            for (String pkg : packages) {
                if (pkg != null && !pkg.trim().isEmpty()) {
                    arr.put(pkg.trim());
                }
            }
        }
        prefs(context)
                .edit()
                .putString(KEY_PACKAGES_JSON, arr.toString())
                .remove(KEY_PACKAGES_LEGACY)
                .commit();
    }

    private static Set<String> packagesFromJson(String json) {
        Set<String> out = new HashSet<>();
        try {
            JSONArray arr = new JSONArray(json);
            for (int i = 0; i < arr.length(); i++) {
                String pkg = arr.optString(i, "").trim();
                if (!pkg.isEmpty()) {
                    out.add(pkg);
                }
            }
        } catch (JSONException ignored) {
            // empty
        }
        return out;
    }

    /**
     * Normalize SMS originator for allowlist lookup (trim, lower, strip «Ваш»/dashes/+7/8).
     * Keep in sync with {@code normalizeSmsSender} in banks.ts.
     */
    static String normalizeSmsSender(String raw) {
        if (raw == null) {
            return "";
        }
        String s = raw.trim().toLowerCase();
        if (s.startsWith("ваш ")) {
            s = s.substring(4).trim();
        } else if (s.startsWith("your ")) {
            s = s.substring(5).trim();
        }
        // Unicode dashes (Messages UI often uses en/em dash in «Т‑Банк») then whitespace.
        s = s.replaceAll("[\\s\\p{Pd}]+", "");
        s = s.replace("тбанк", "tbank");
        s = s.replace("тинькофф", "tinkoff");
        if (s.startsWith("+7") && s.length() > 2) {
            s = s.substring(2);
        } else if (s.startsWith("8") && s.length() == 11 && s.chars().allMatch(Character::isDigit)) {
            s = s.substring(1);
        }
        return s;
    }

    static Map<String, String> smsSenderToPackage(Context context) {
        SharedPreferences p = prefs(context);
        String json = p.getString(KEY_SMS_SENDERS_JSON, null);
        if (json == null) {
            return sendersFromJson(DEFAULT_SMS_SENDERS_JSON);
        }
        Map<String, String> map = sendersFromJson(json);
        // Empty synced map (WebView glitch / partial sync) — fall back like cold start.
        if (map.isEmpty()) {
            return sendersFromJson(DEFAULT_SMS_SENDERS_JSON);
        }
        return map;
    }

    /**
     * Replace SMS sender allowlist. Pass empty map to clear synced data (fallback defaults apply
     * on next read when JSON is missing; empty object means "synced empty" — no SMS capture).
     */
    static void setAllowedSmsSenders(Context context, Map<String, String> senderToPackage) {
        JSONObject obj = new JSONObject();
        if (senderToPackage != null) {
            for (Map.Entry<String, String> e : senderToPackage.entrySet()) {
                String sender = normalizeSmsSender(e.getKey());
                String pkg = e.getValue() != null ? e.getValue().trim() : "";
                if (!sender.isEmpty() && !pkg.isEmpty()) {
                    try {
                        obj.put(sender, pkg);
                    } catch (JSONException ignored) {
                        // skip
                    }
                }
            }
        }
        prefs(context).edit().putString(KEY_SMS_SENDERS_JSON, obj.toString()).commit();
    }

    /**
     * Fallback when {@link #packageForSmsSender} misses (OEM Messages titles, empty synced map).
     * Keep roughly in sync with {@code inferBankIdFromText} in banks.ts.
     */
    static String inferBankPackageFromText(String combined) {
        if (combined == null || combined.trim().isEmpty()) {
            return null;
        }
        String s = combined.toLowerCase(Locale.ROOT);
        s = s.replaceAll("\\p{Pd}+", "-").replaceAll("\\s+", " ");
        if (BANK_TEXT_TINKOFF.matcher(s).find()) {
            return "com.idamob.tinkoff.android";
        }
        if (BANK_TEXT_SBER.matcher(s).find()) {
            return "ru.sberbankmobile";
        }
        if (BANK_TEXT_VTB.matcher(s).find()) {
            return "ru.vtb24.mobilebanking.android";
        }
        if (BANK_TEXT_ALFA.matcher(s).find()) {
            return "ru.alfabank.mobile.android";
        }
        if (BANK_TEXT_GPB.matcher(s).find()) {
            return "ru.gazprombank.android.mobilebank.app";
        }
        if (BANK_TEXT_RAIF.matcher(s).find()) {
            return "ru.raiffeisennews";
        }
        if (BANK_TEXT_ROSBANK.matcher(s).find()) {
            return "ru.rosbank.android";
        }
        if (BANK_TEXT_RSHB.matcher(s).find()) {
            return "ru.rshb.dbo";
        }
        if (BANK_TEXT_OPEN.matcher(s).find()) {
            return "com.openbank";
        }
        if (BANK_TEXT_SOVCOM.matcher(s).find()) {
            return "ru.sovcombank.mobile";
        }
        if (BANK_TEXT_URALSIB.matcher(s).find()) {
            return "ru.uralsib.mb";
        }
        if (BANK_TEXT_HOME.matcher(s).find()) {
            return "ru.homecredit.mycredit";
        }
        if (BANK_TEXT_MKB.matcher(s).find()) {
            return "ru.mkb.mobile";
        }
        if (BANK_TEXT_PSB.matcher(s).find()) {
            return "ru.ftc.faktura.psb";
        }
        if (BANK_TEXT_OTP.matcher(s).find()) {
            return "ru.otpbank.mobile";
        }
        if (BANK_TEXT_VBRR.matcher(s).find()) {
            return "com.bssys.vbrrretail";
        }
        if (BANK_TEXT_OZON.matcher(s).find()) {
            return "ru.ozon.fintech.finance";
        }
        if (BANK_TEXT_YANDEX.matcher(s).find()) {
            return "com.yandex.bank";
        }
        if (BANK_TEXT_WB.matcher(s).find()) {
            return "ru.wildberries.fintech";
        }
        return null;
    }

    private static final Pattern BANK_TEXT_TINKOFF =
            Pattern.compile("т-?банк|тинькофф|t-?bank|tinkoff|tbank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_SBER =
            Pattern.compile("сбер(?:банк|пей|\\s*pay)?|\\bsber(?:bank|pay)?\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_VTB =
            Pattern.compile("втб|\\bvtb\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_ALFA =
            Pattern.compile("альфа[-\\s]?банк|alfa[-\\s]?bank|\\balfabank\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_GPB =
            Pattern.compile("газпромбанк|gazprombank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_RAIF =
            Pattern.compile("райффайзен|raiffeisen", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_ROSBANK =
            Pattern.compile("росбанк|rosbank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_RSHB =
            Pattern.compile("россельхоз(?:банк)?|\\brshb\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_OPEN =
            Pattern.compile("открытие|otkritie", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_SOVCOM =
            Pattern.compile("совкомбанк|sovcombank|\\bхалва\\b|\\bhalva\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_URALSIB =
            Pattern.compile("уралсиб|uralsib", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_HOME =
            Pattern.compile("хоум\\s*кредит|home\\s*credit", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_MKB = Pattern.compile("мкб|\\bmkb\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_PSB =
            Pattern.compile("псб|\\bpsb\\b|промсвязьбанк", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_OTP =
            Pattern.compile("отп\\s*банк|otp\\s*bank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_VBRR =
            Pattern.compile("вбрр|\\bvbrr\\b", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_OZON =
            Pattern.compile("озон\\s*банк|ozon\\s*bank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_YANDEX =
            Pattern.compile("яндекс\\s*банк|yandex\\s*bank", Pattern.CASE_INSENSITIVE);
    private static final Pattern BANK_TEXT_WB =
            Pattern.compile("wb\\s*банк|wildberries\\s*банк", Pattern.CASE_INSENSITIVE);

    /** @return primary package for sender, or null if not allowlisted */
    static String packageForSmsSender(Context context, String sender) {
        String key = normalizeSmsSender(sender);
        if (key.isEmpty()) {
            return null;
        }
        Map<String, String> map = smsSenderToPackage(context);
        String pkg = map.get(key);
        return pkg != null && !pkg.isEmpty() ? pkg : null;
    }

    private static Map<String, String> sendersFromJson(String json) {
        Map<String, String> out = new HashMap<>();
        if (json == null || json.isEmpty()) {
            return out;
        }
        try {
            JSONObject obj = new JSONObject(json);
            Iterator<String> keys = obj.keys();
            while (keys.hasNext()) {
                String k = keys.next();
                String pkg = obj.optString(k, "").trim();
                String norm = normalizeSmsSender(k);
                if (!norm.isEmpty() && !pkg.isEmpty()) {
                    out.put(norm, pkg);
                }
            }
        } catch (JSONException ignored) {
            // empty
        }
        return out;
    }

    static synchronized void append(Context context, JSONObject item) {
        if (!isCaptureEnabled(context)) {
            return;
        }
        String pkg = item.optString("packageName", "");
        if (pkg.isEmpty() || !allowedPackages(context).contains(pkg)) {
            return;
        }
        try {
            JSONArray arr = readArray(context);
            String dedupeKey = item.optString("dedupeKey", "");
            if (!dedupeKey.isEmpty()) {
                for (int i = 0; i < arr.length(); i++) {
                    if (dedupeKey.equals(arr.getJSONObject(i).optString("dedupeKey", ""))) {
                        return;
                    }
                }
            }
            arr.put(item);
            while (arr.length() > MAX_PENDING) {
                JSONArray trimmed = new JSONArray();
                for (int i = arr.length() - MAX_PENDING; i < arr.length(); i++) {
                    trimmed.put(arr.get(i));
                }
                arr = trimmed;
            }
            prefs(context).edit().putString(KEY_PENDING, arr.toString()).apply();
        } catch (JSONException ignored) {
            // ignore malformed queue
        }
    }

    static synchronized JSONArray consumeAll(Context context) {
        JSONArray arr = readArray(context);
        prefs(context).edit().putString(KEY_PENDING, "[]").apply();
        return arr;
    }

    static synchronized JSONArray peekAll(Context context) {
        return readArray(context);
    }

    /** Remove pending rows by dedupeKey (leave the rest for a later retry). */
    static synchronized void acknowledge(Context context, Set<String> dedupeKeys) {
        if (dedupeKeys == null || dedupeKeys.isEmpty()) {
            return;
        }
        try {
            JSONArray arr = readArray(context);
            JSONArray kept = new JSONArray();
            for (int i = 0; i < arr.length(); i++) {
                JSONObject obj = arr.getJSONObject(i);
                String key = obj.optString("dedupeKey", "");
                if (key.isEmpty() || !dedupeKeys.contains(key)) {
                    kept.put(obj);
                }
            }
            prefs(context).edit().putString(KEY_PENDING, kept.toString()).apply();
        } catch (JSONException ignored) {
            // ignore
        }
    }

    private static JSONArray readArray(Context context) {
        String raw = prefs(context).getString(KEY_PENDING, "[]");
        try {
            return new JSONArray(raw);
        } catch (JSONException e) {
            return new JSONArray();
        }
    }

    /**
     * Append a debug history row. Uses the same SharedPreferences as the pending queue
     * (known to persist on MIUI when the separate history prefs file did not show up in UI).
     */
    static void appendHistory(Context context, JSONObject item) {
        final String payload;
        synchronized (HISTORY_LOCK) {
            try {
                JSONArray arr = readHistoryArray(context);
                String dedupeKey = item.optString("dedupeKey", "");
                if (!dedupeKey.isEmpty()) {
                    for (int i = 0; i < arr.length(); i++) {
                        if (dedupeKey.equals(arr.getJSONObject(i).optString("dedupeKey", ""))) {
                            return;
                        }
                    }
                }
                JSONArray next = new JSONArray();
                next.put(item);
                for (int i = 0; i < arr.length() && next.length() < MAX_HISTORY; i++) {
                    next.put(arr.get(i));
                }
                payload = next.toString();
            } catch (JSONException e) {
                return;
            }
        }
        prefs(context).edit().putString(KEY_HISTORY, payload).commit();
    }

    /** Lock-free read for WebView. */
    static String peekHistoryJson(Context context) {
        try {
            String raw = prefs(context).getString(KEY_HISTORY, "[]");
            return raw != null ? raw : "[]";
        } catch (RuntimeException e) {
            return "[]";
        }
    }

    static void clearHistory(Context context) {
        prefs(context).edit().putString(KEY_HISTORY, "[]").commit();
    }

    private static JSONArray readHistoryArray(Context context) {
        String raw = prefs(context).getString(KEY_HISTORY, "[]");
        try {
            return new JSONArray(raw != null ? raw : "[]");
        } catch (JSONException e) {
            return new JSONArray();
        }
    }
}
