package ru.kai_zer.buhgalter;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(WifiSubnetPlugin.class);
        registerPlugin(SslTrustPlugin.class);
        registerPlugin(LanDiscoveryPlugin.class);
        registerPlugin(DebugExportPlugin.class);
        registerPlugin(WidgetBridgePlugin.class);
        registerPlugin(ShareTargetPlugin.class);
        super.onCreate(savedInstanceState);
    }
}
