package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.SharedPreferences;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

/**
 * Native mirror of intercept drafts for shade quick-actions, plus accept/reject queues for JS sync.
 */
final class DraftNotifyStore {
    private static final String PREFS = "buhgalter.draft_notify.v1";
    private static final String KEY_DRAFTS = "drafts_json";
    private static final String KEY_SHADE = "shade_enabled";
    private static final String KEY_REJECTED = "rejected_ids_json";
    private static final String KEY_ACCEPTED = "accepted_ids_json";
    private static final String KEY_OFFLINE_ACCEPT = "offline_accept_json";
    private static final String KEY_SNOOZED = "snoozed_ids_json";
    private static final int MAX_DRAFTS = 50;
    private static final int MAX_QUEUE = 80;

    private DraftNotifyStore() {}

    private static SharedPreferences prefs(Context context) {
        return context.getApplicationContext().getSharedPreferences(PREFS, Context.MODE_PRIVATE);
    }

    static boolean isShadeEnabled(Context context) {
        return prefs(context).getBoolean(KEY_SHADE, true);
    }

    static void setShadeEnabled(Context context, boolean enabled) {
        prefs(context).edit().putBoolean(KEY_SHADE, enabled).apply();
    }

    static JSONArray listDrafts(Context context) {
        try {
            return new JSONArray(prefs(context).getString(KEY_DRAFTS, "[]"));
        } catch (JSONException e) {
            return new JSONArray();
        }
    }

    static JSONObject getDraft(Context context, String draftId) {
        if (draftId == null || draftId.isEmpty()) return null;
        JSONArray arr = listDrafts(context);
        for (int i = 0; i < arr.length(); i++) {
            JSONObject o = arr.optJSONObject(i);
            if (o != null && draftId.equals(o.optString("id", ""))) {
                return o;
            }
        }
        return null;
    }

    /** Replace the whole draft list (JS is source of truth for UI). */
    static void replaceDrafts(Context context, JSONArray drafts) {
        JSONArray trimmed = new JSONArray();
        int n = Math.min(drafts != null ? drafts.length() : 0, MAX_DRAFTS);
        Set<String> nextIds = new HashSet<>();
        for (int i = 0; i < n; i++) {
            JSONObject o = drafts.optJSONObject(i);
            if (o != null && !o.optString("id", "").isEmpty()) {
                trimmed.put(o);
                nextIds.add(o.optString("id", ""));
            }
        }
        // Drop snooze for drafts that no longer exist.
        JSONArray snoozed = readArray(context, KEY_SNOOZED);
        JSONArray keepSnooze = new JSONArray();
        for (int i = 0; i < snoozed.length(); i++) {
            String id = snoozed.optString(i, "");
            if (!id.isEmpty() && nextIds.contains(id)) {
                keepSnooze.put(id);
            }
        }
        prefs(context)
                .edit()
                .putString(KEY_DRAFTS, trimmed.toString())
                .putString(KEY_SNOOZED, keepSnooze.toString())
                .apply();
    }

    static void upsertDraft(Context context, JSONObject draft) {
        if (draft == null) return;
        String id = draft.optString("id", "");
        if (id.isEmpty()) return;
        JSONArray arr = listDrafts(context);
        JSONArray next = new JSONArray();
        next.put(draft);
        for (int i = 0; i < arr.length(); i++) {
            JSONObject o = arr.optJSONObject(i);
            if (o == null) continue;
            if (id.equals(o.optString("id", ""))) continue;
            next.put(o);
            if (next.length() >= MAX_DRAFTS) break;
        }
        prefs(context).edit().putString(KEY_DRAFTS, next.toString()).apply();
    }

    static boolean removeDraft(Context context, String draftId) {
        if (draftId == null || draftId.isEmpty()) return false;
        JSONArray arr = listDrafts(context);
        JSONArray next = new JSONArray();
        boolean removed = false;
        for (int i = 0; i < arr.length(); i++) {
            JSONObject o = arr.optJSONObject(i);
            if (o == null) continue;
            if (draftId.equals(o.optString("id", ""))) {
                removed = true;
                continue;
            }
            next.put(o);
        }
        if (removed) {
            prefs(context).edit().putString(KEY_DRAFTS, next.toString()).apply();
            clearSnoozed(context, draftId);
        }
        return removed;
    }

    static void clearDrafts(Context context) {
        prefs(context)
                .edit()
                .putString(KEY_DRAFTS, "[]")
                .putString(KEY_SNOOZED, "[]")
                .apply();
    }

    /** Hide shade notification until draft is removed (e.g. Accept opened the form). */
    static void snoozeDraft(Context context, String draftId) {
        enqueueId(context, KEY_SNOOZED, draftId);
    }

    static boolean isSnoozed(Context context, String draftId) {
        if (draftId == null || draftId.isEmpty()) return false;
        JSONArray arr = readArray(context, KEY_SNOOZED);
        for (int i = 0; i < arr.length(); i++) {
            if (draftId.equals(arr.optString(i, ""))) return true;
        }
        return false;
    }

    static void clearSnoozed(Context context, String draftId) {
        if (draftId == null || draftId.isEmpty()) return;
        JSONArray arr = readArray(context, KEY_SNOOZED);
        JSONArray next = new JSONArray();
        for (int i = 0; i < arr.length(); i++) {
            String id = arr.optString(i, "");
            if (!id.isEmpty() && !draftId.equals(id)) next.put(id);
        }
        prefs(context).edit().putString(KEY_SNOOZED, next.toString()).apply();
    }

    static void enqueueRejected(Context context, String draftId) {
        enqueueId(context, KEY_REJECTED, draftId);
    }

    static void enqueueAccepted(Context context, String draftId) {
        enqueueId(context, KEY_ACCEPTED, draftId);
    }

    static void enqueueOfflineAccept(Context context, JSONObject payload) {
        if (payload == null) return;
        JSONArray arr = readArray(context, KEY_OFFLINE_ACCEPT);
        arr.put(payload);
        while (arr.length() > MAX_QUEUE) {
            JSONArray trimmed = new JSONArray();
            for (int i = 1; i < arr.length(); i++) {
                trimmed.put(arr.opt(i));
            }
            arr = trimmed;
        }
        prefs(context).edit().putString(KEY_OFFLINE_ACCEPT, arr.toString()).apply();
    }

    /** Consume rejected + accepted ids and offline-accept payloads for JS. */
    static JSONObject consumeSync(Context context) {
        JSONObject out = new JSONObject();
        try {
            out.put("rejectedIds", readArray(context, KEY_REJECTED));
            out.put("acceptedIds", readArray(context, KEY_ACCEPTED));
            out.put("offlineAccepts", readArray(context, KEY_OFFLINE_ACCEPT));
        } catch (JSONException ignored) {
            // empty
        }
        prefs(context)
                .edit()
                .putString(KEY_REJECTED, "[]")
                .putString(KEY_ACCEPTED, "[]")
                .putString(KEY_OFFLINE_ACCEPT, "[]")
                .apply();
        return out;
    }

    private static void enqueueId(Context context, String key, String draftId) {
        if (draftId == null || draftId.trim().isEmpty()) return;
        JSONArray arr = readArray(context, key);
        for (int i = 0; i < arr.length(); i++) {
            if (draftId.equals(arr.optString(i, ""))) return;
        }
        arr.put(draftId);
        while (arr.length() > MAX_QUEUE) {
            JSONArray trimmed = new JSONArray();
            for (int i = 1; i < arr.length(); i++) {
                trimmed.put(arr.opt(i));
            }
            arr = trimmed;
        }
        prefs(context).edit().putString(key, arr.toString()).apply();
    }

    private static JSONArray readArray(Context context, String key) {
        try {
            return new JSONArray(prefs(context).getString(key, "[]"));
        } catch (JSONException e) {
            return new JSONArray();
        }
    }

    /** Draft ids currently mirrored (for tests / debug). */
    static Set<String> draftIds(Context context) {
        Set<String> ids = new HashSet<>();
        JSONArray arr = listDrafts(context);
        for (int i = 0; i < arr.length(); i++) {
            JSONObject o = arr.optJSONObject(i);
            if (o != null) {
                String id = o.optString("id", "");
                if (!id.isEmpty()) ids.add(id);
            }
        }
        return ids;
    }

    static List<String> listDraftIdList(Context context) {
        return new ArrayList<>(draftIds(context));
    }

    /** Package-private for unit tests. */
    static void clearAllForTests(Context context) {
        prefs(context).edit().clear().apply();
    }
}
