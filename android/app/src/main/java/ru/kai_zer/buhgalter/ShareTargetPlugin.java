package ru.kai_zer.buhgalter;

import android.content.Intent;
import android.net.Uri;
import android.os.Build;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * Incoming Android share targets ({@link Intent#ACTION_SEND}).
 * JS consumes via {@code consumePending} and/or {@code shareReceived} listener.
 */
@CapacitorPlugin(name = "ShareTarget")
public class ShareTargetPlugin extends Plugin {

    private JSObject pending;

    @Override
    public void load() {
        Intent intent = getActivity() != null ? getActivity().getIntent() : null;
        captureShare(intent);
    }

    @Override
    protected void handleOnNewIntent(Intent intent) {
        super.handleOnNewIntent(intent);
        if (getActivity() != null && intent != null) {
            getActivity().setIntent(intent);
        }
        captureShare(intent);
    }

    private void captureShare(Intent intent) {
        if (intent == null) {
            return;
        }
        String action = intent.getAction();
        if (!Intent.ACTION_SEND.equals(action)) {
            return;
        }

        JSObject payload = new JSObject();
        String text = intent.getStringExtra(Intent.EXTRA_TEXT);
        if (text != null && !text.trim().isEmpty()) {
            payload.put("text", text.trim());
        }
        String subject = intent.getStringExtra(Intent.EXTRA_SUBJECT);
        if (subject != null && !subject.trim().isEmpty()) {
            payload.put("subject", subject.trim());
        }

        Uri stream = readStreamUri(intent);
        if (stream != null) {
            payload.put("streamUri", stream.toString());
            String type = intent.getType();
            if (type != null && !type.isEmpty()) {
                payload.put("mimeType", type);
            }
        }

        if (!payload.keys().hasNext()) {
            return;
        }

        pending = payload;
        notifyListeners("shareReceived", payload);
    }

    @SuppressWarnings("deprecation")
    private Uri readStreamUri(Intent intent) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            return intent.getParcelableExtra(Intent.EXTRA_STREAM, Uri.class);
        }
        return intent.getParcelableExtra(Intent.EXTRA_STREAM);
    }

    @PluginMethod
    public void consumePending(PluginCall call) {
        JSObject out = pending;
        pending = null;
        if (out == null) {
            call.resolve(new JSObject());
            return;
        }
        call.resolve(out);
    }
}
