package ru.kai_zer.buhgalter;

import android.os.Bundle;
import android.webkit.WebView;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeActivity;

import ru.kai_zer.buhgalter.widgets.WidgetRefreshScheduler;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(AppInstancePlugin.class);
        registerPlugin(WifiSubnetPlugin.class);
        registerPlugin(SslTrustPlugin.class);
        registerPlugin(LanDiscoveryPlugin.class);
        registerPlugin(DebugExportPlugin.class);
        registerPlugin(WidgetBridgePlugin.class);
        registerPlugin(ShareTargetPlugin.class);
        registerPlugin(NotificationInterceptPlugin.class);
        super.onCreate(savedInstanceState);
        // Drain any stuck OneTime WidgetRefresh backlog from the old onUpdate feedback loop.
        WidgetRefreshScheduler.ensurePeriodic(this);
        attachHistoryBridge();
    }

    @Override
    public void onResume() {
        super.onResume();
        attachHistoryBridge();
        // MIUI/HyperOS often unbinds NotificationListenerService while the app is backgrounded.
        if (NotificationInterceptStore.isCaptureEnabled(this)) {
            final android.content.Context app = getApplicationContext();
            new Thread(
                    () -> BankNotificationListenerService.ensureConnected(app, 4000),
                    "nls-resume-rebind")
                    .start();
        }
    }

    private void attachHistoryBridge() {
        try {
            Bridge bridge = getBridge();
            if (bridge == null) {
                return;
            }
            WebView webView = bridge.getWebView();
            if (webView == null) {
                return;
            }
            webView.addJavascriptInterface(
                    new NotificationHistoryJsBridge(getApplicationContext()),
                    "BuhgalterNotificationHistory");
        } catch (RuntimeException ignored) {
            // WebView not ready yet
        }
    }
}
