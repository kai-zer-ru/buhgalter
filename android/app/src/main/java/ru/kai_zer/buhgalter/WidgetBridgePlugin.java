package ru.kai_zer.buhgalter;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import org.json.JSONObject;

import ru.kai_zer.buhgalter.widgets.WidgetRefreshScheduler;
import ru.kai_zer.buhgalter.widgets.WidgetSnapshotStore;
import ru.kai_zer.buhgalter.widgets.WidgetUpdater;

@CapacitorPlugin(name = "WidgetBridge")
public class WidgetBridgePlugin extends Plugin {

    @PluginMethod
    public void publish(PluginCall call) {
        String baseUrl = call.getString("baseUrl", "");
        String token = call.getString("token", "");
        boolean lockEnabled = Boolean.TRUE.equals(call.getBoolean("lockEnabled", false));
        JSObject snapshot = call.getObject("snapshot");
        if (snapshot == null) {
            call.reject("snapshot required");
            return;
        }
        try {
            String json = new JSONObject(snapshot.toString()).toString();
            WidgetSnapshotStore.publish(getContext(), baseUrl, token, lockEnabled, json);
            WidgetRefreshScheduler.ensurePeriodic(getContext());
            WidgetUpdater.updateAll(getContext());
            call.resolve();
        } catch (Exception e) {
            call.reject(e.getMessage());
        }
    }

    @PluginMethod
    public void setLockEnabled(PluginCall call) {
        boolean lockEnabled = Boolean.TRUE.equals(call.getBoolean("lockEnabled", false));
        WidgetSnapshotStore.setLockEnabled(getContext(), lockEnabled);
        WidgetUpdater.updateAll(getContext());
        call.resolve();
    }

    @PluginMethod
    public void clear(PluginCall call) {
        WidgetSnapshotStore.clear(getContext());
        WidgetUpdater.updateAll(getContext());
        call.resolve();
    }
}
