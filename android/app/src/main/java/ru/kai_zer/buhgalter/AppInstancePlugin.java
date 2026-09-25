package ru.kai_zer.buhgalter;

import android.content.Context;
import android.view.View;
import android.view.inputmethod.InputMethodManager;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * Identifies this process (UID). Namespaces secure-storage keys for work profile /
 * secondary users. OEM Dual Apps clones are blocked in {@link CloneBlockedActivity}.
 */
@CapacitorPlugin(name = "AppInstance")
public class AppInstancePlugin extends Plugin {

    @PluginMethod
    public void getStorageNamespace(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("namespace", "u" + android.os.Process.myUid());
        call.resolve(ret);
    }

    /** Hide the system IME — used while MoneyInput shows the in-app keypad. */
    @PluginMethod
    public void hideSoftKeyboard(PluginCall call) {
        getActivity()
                .runOnUiThread(
                        () -> {
                            try {
                                View focus = getActivity().getCurrentFocus();
                                View webView = getBridge().getWebView();
                                View tokenView = focus != null ? focus : webView;
                                if (tokenView != null) {
                                    InputMethodManager imm =
                                            (InputMethodManager)
                                                    getContext()
                                                            .getSystemService(
                                                                    Context.INPUT_METHOD_SERVICE);
                                    if (imm != null) {
                                        imm.hideSoftInputFromWindow(tokenView.getWindowToken(), 0);
                                    }
                                }
                            } catch (RuntimeException ignored) {
                                // Bridge/WebView not ready
                            }
                            call.resolve();
                        });
    }
}
