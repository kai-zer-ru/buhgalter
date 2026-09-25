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

        if ("transfer".equals(draft.optString("type", ""))) {
            handleTransferAccept(app, itemId, draft);
            return;
        }

        String accountId = draft.optString("accountId", "");
        if (TextUtils.isEmpty(accountId)) {
            // Cannot create without account — open form with prefill; keep draft until save.
            DraftNotifyStore.snoozeDraft(app, itemId);
            DraftNotifyHelper.cancelDraft(app, itemId);
            openPath(app, DraftNotifyHelper.contentPath(draft));
            return;
        }

        // Accept complete: try native POST, else queue for JS outbox.
        boolean ok = DraftNotifyHttp.postTransaction(app, draft);
        DraftNotifyStore.removeDraft(app, itemId);
        DraftNotifyHelper.cancelDraft(app, itemId);
        if (ok) {
            DraftNotifyStore.enqueueAccepted(app, itemId);
        } else {
            JSONObject payload = DraftNotifyHttp.offlinePayloadFromDraft(draft);
            if (payload != null) {
                DraftNotifyStore.enqueueOfflineAccept(app, payload);
            } else {
                DraftNotifyStore.enqueueRejected(app, itemId);
            }
        }
        NotificationInterceptPlugin.emitDraftsChanged();
    }

    private static void handleTransferAccept(Context app, String itemId, JSONObject draft) {
        String fromAccount = draft.optString("fromAccountId", "");
        String toAccount = draft.optString("toAccountId", "");
        String fromDraftId = draft.optString("fromDraftId", "");
        String toDraftId = draft.optString("toDraftId", "");

        if (TextUtils.isEmpty(fromAccount) || TextUtils.isEmpty(toAccount)) {
            DraftNotifyStore.snoozeDraft(app, itemId);
            DraftNotifyHelper.cancelDraft(app, itemId);
            openPath(app, DraftNotifyHelper.contentPath(draft));
            return;
        }

        boolean ok = DraftNotifyHttp.postTransfer(app, draft);
        DraftNotifyStore.removeDraft(app, itemId);
        DraftNotifyHelper.cancelDraft(app, itemId);
        if (ok) {
            if (!TextUtils.isEmpty(fromDraftId)) DraftNotifyStore.enqueueAccepted(app, fromDraftId);
            if (!TextUtils.isEmpty(toDraftId)) DraftNotifyStore.enqueueAccepted(app, toDraftId);
        } else {
            JSONObject payload = DraftNotifyHttp.offlinePayloadFromTransfer(draft);
            if (payload != null) {
                DraftNotifyStore.enqueueOfflineAccept(app, payload);
            } else {
                if (!TextUtils.isEmpty(fromDraftId)) {
                    DraftNotifyStore.enqueueRejected(app, fromDraftId);
                }
                if (!TextUtils.isEmpty(toDraftId)) {
                    DraftNotifyStore.enqueueRejected(app, toDraftId);
                }
            }
        }
        NotificationInterceptPlugin.emitDraftsChanged();
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
