package ru.kai_zer.buhgalter;

import android.content.Intent;
import android.os.Build;
import android.os.Bundle;
import android.webkit.WebView;

import androidx.core.splashscreen.SplashScreen;

import com.getcapacitor.Bridge;
import com.getcapacitor.BridgeActivity;

import ru.kai_zer.buhgalter.widgets.WidgetRefreshScheduler;

public class MainActivity extends BridgeActivity {
    private boolean skipCapacitor;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        // HyperOS keeps the Android 12 splash black until timeout unless we
        // install it and immediately release. First native frame is ~1s; the
        // remaining ~30s was empty WebView waiting on Google Fonts / renderer.
        SplashScreen splash = SplashScreen.installSplashScreen(this);
        splash.setKeepOnScreenCondition(() -> false);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            getSplashScreen().setOnExitAnimationListener(view -> view.remove());
        }
        if (CloneGuard.isUnsupportedClone(this)) {
            skipCapacitor = true;
            super.onCreate(savedInstanceState);
            startActivity(new Intent(this, CloneBlockedActivity.class));
            finish();
            return;
        }
        registerPlugin(AppInstancePlugin.class);
        registerPlugin(WifiSubnetPlugin.class);
        registerPlugin(SslTrustPlugin.class);
        registerPlugin(LanDiscoveryPlugin.class);
        registerPlugin(DebugExportPlugin.class);
        registerPlugin(WidgetBridgePlugin.class);
        registerPlugin(ShareTargetPlugin.class);
        registerPlugin(NotificationInterceptPlugin.class);
        super.onCreate(savedInstanceState);
        WidgetRefreshScheduler.ensurePeriodicAsync(this);
        prepareWebView();
        attachHistoryBridge();
    }

    @Override
    public void setContentView(int layoutResID) {
        if (skipCapacitor) {
            super.setContentView(R.layout.activity_clone_blocked);
            return;
        }
        super.setContentView(layoutResID);
    }

    @Override
    protected void load() {
        if (skipCapacitor) {
            return;
        }
        super.load();
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

    private void prepareWebView() {
        try {
            Bridge bridge = getBridge();
            if (bridge == null) {
                return;
            }
            WebView webView = bridge.getWebView();
            if (webView == null) {
                return;
            }
            webView.setBackgroundColor(getColor(R.color.splash_background));
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                // Xiaomi SmartPower otherwise backgrounds the renderer under the splash.
                webView.setRendererPriorityPolicy(WebView.RENDERER_PRIORITY_IMPORTANT, false);
            }
        } catch (RuntimeException ignored) {
            // Bridge not ready
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
