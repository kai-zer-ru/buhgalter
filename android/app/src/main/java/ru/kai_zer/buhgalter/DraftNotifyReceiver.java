package ru.kai_zer.buhgalter;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.text.TextUtils;

import org.json.JSONObject;

import ru.kai_zer.buhgalter.widgets.WidgetDeepLinks;

/**
 * Handles Accept / Reject from intercept-draft notifications in the shade.
 */
public class DraftNotifyReceiver extends BroadcastReceiver {

    @Override
    public void onReceive(Context context, Intent intent) {
        if (intent == null || intent.getAction() == null) return;
        String itemId = intent.getStringExtra(DraftNotifyHelper.EXTRA_DRAFT_ID);
        if (TextUtils.isEmpty(itemId)) return;
        Context app = context.getApplicationContext();

        if (DraftNotifyHelper.ACTION_REJECT.equals(intent.getAction())) {
            JSONObject item = DraftNotifyStore.getDraft(app, itemId);
            DraftNotifyStore.removeDraft(app, itemId);
            DraftNotifyHelper.cancelDraft(app, itemId);
            enqueueLinkedRejected(app, item, itemId);
            NotificationInterceptPlugin.emitDraftsChanged();
            return;
        }

        if (DraftNotifyHelper.ACTION_OPEN.equals(intent.getAction())) {
            JSONObject draft = DraftNotifyStore.getDraft(app, itemId);
            DraftNotifyStore.snoozeDraft(app, itemId);
            DraftNotifyHelper.cancelDraft(app, itemId);
            openPath(app, DraftNotifyHelper.contentPath(draft));
            return;
        }

        if (!DraftNotifyHelper.ACTION_ACCEPT.equals(intent.getAction())) return;

        JSONObject draft = DraftNotifyStore.getDraft(app, itemId);
        if (draft == null) {
            DraftNotifyHelper.cancelDraft(app, itemId);
            DraftNotifyStore.enqueueRejected(app, itemId);
            return;
        }

        // Always open create/transfer form with prefill (same as ACTION_OPEN).
        // Keep draft until Save in the form — do not silent POST / removeDraft here.
        DraftNotifyStore.snoozeDraft(app, itemId);
        DraftNotifyHelper.cancelDraft(app, itemId);
        openPath(app, DraftNotifyHelper.contentPath(draft));
    }

    private static void enqueueLinkedRejected(Context app, JSONObject item, String fallbackId) {
        if (item != null && "transfer".equals(item.optString("type", ""))) {
            String fromId = item.optString("fromDraftId", "");
            String toId = item.optString("toDraftId", "");
            if (!TextUtils.isEmpty(fromId)) DraftNotifyStore.enqueueRejected(app, fromId);
            if (!TextUtils.isEmpty(toId)) DraftNotifyStore.enqueueRejected(app, toId);
            return;
        }
        DraftNotifyStore.enqueueRejected(app, fallbackId);
    }

    private static void openPath(Context app, String path) {
        Intent open = new Intent(Intent.ACTION_VIEW, WidgetDeepLinks.uri(path));
        open.setClass(app, MainActivity.class);
        open.setFlags(
                Intent.FLAG_ACTIVITY_NEW_TASK
                        | Intent.FLAG_ACTIVITY_CLEAR_TOP
                        | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        app.startActivity(open);
    }
}
