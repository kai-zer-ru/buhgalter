package ru.kai_zer.buhgalter;

import android.Manifest;
import android.content.ComponentName;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.os.Handler;
import android.os.Looper;
import android.os.PowerManager;
import android.provider.Settings;
import android.text.TextUtils;

import androidx.core.content.ContextCompat;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.WeakHashMap;

/**
 * Bridge for bank notification intercept: access status, capture toggle, pending queue, SMS.
 */
@CapacitorPlugin(
        name = "NotificationIntercept",
        permissions = {
            @Permission(
                    strings = {Manifest.permission.RECEIVE_SMS},
                    alias = "sms")
        })
public class NotificationInterceptPlugin extends Plugin {

    private static final WeakHashMap<NotificationInterceptPlugin, Boolean> INSTANCES = new WeakHashMap<>();

    @Override
    public void load() {
        INSTANCES.put(this, Boolean.TRUE);
    }

    static void emitPendingAvailable() {
        for (NotificationInterceptPlugin plugin : INSTANCES.keySet()) {
            if (plugin != null) {
                plugin.notifyListeners("pendingAvailable", new JSObject());
            }
        }
    }

    @PluginMethod
    public void isNotificationAccessEnabled(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("enabled", hasNotificationAccess());
        call.resolve(ret);
    }

    @PluginMethod
    public void openNotificationAccessSettings(PluginCall call) {
        Intent intent = new Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS);
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        getContext().startActivity(intent);
        call.resolve();
    }

    /**
     * Open OEM autostart / battery screens so the notification listener survives process death
     * (especially Xiaomi/MIUI/HyperOS).
     */
    @PluginMethod
    public void openBackgroundRestrictionsSettings(PluginCall call) {
        android.content.Context ctx = getContext();
        String pkg = ctx.getPackageName();
        Intent[] candidates =
                new Intent[] {
                    // MIUI / HyperOS autostart
                    new Intent("miui.intent.action.OP_AUTO_START")
                            .addCategory(Intent.CATEGORY_DEFAULT),
                    new Intent()
                            .setComponent(
                                    new ComponentName(
                                            "com.miui.securitycenter",
                                            "com.miui.permcenter.autostart.AutoStartManagementActivity")),
                    new Intent()
                            .setComponent(
                                    new ComponentName(
                                            "com.miui.securitycenter",
                                            "com.miui.powercenter.PowerSettings")),
                    // Generic battery optimization for this app
                    new Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS)
                            .setData(Uri.parse("package:" + pkg)),
                    new Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS),
                    new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS)
                            .setData(Uri.parse("package:" + pkg))
                };
        boolean opened = false;
        for (Intent intent : candidates) {
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            try {
                if (intent.resolveActivity(ctx.getPackageManager()) != null) {
                    ctx.startActivity(intent);
                    opened = true;
                    break;
                }
            } catch (RuntimeException ignored) {
                // try next
            }
        }
        JSObject ret = new JSObject();
        ret.put("opened", opened);
        PowerManager pm = (PowerManager) ctx.getSystemService(android.content.Context.POWER_SERVICE);
        boolean ignoring =
                pm != null && pm.isIgnoringBatteryOptimizations(pkg);
        ret.put("batteryOptimizationIgnored", ignoring);
        call.resolve(ret);
    }

    @PluginMethod
    public void setCaptureEnabled(PluginCall call) {
        Boolean enabled = call.getBoolean("enabled");
        if (enabled == null) {
            call.reject("enabled required");
            return;
        }
        NotificationInterceptStore.setCaptureEnabled(getContext(), enabled);
        call.resolve();
        if (enabled) {
            // Permission may already be granted while the service is unbound (common on MIUI).
            // Do not block the Capacitor bridge thread on rebind waits.
            final android.content.Context app = getContext().getApplicationContext();
            new Thread(
                    () -> BankNotificationListenerService.ensureConnected(app, 4000),
                    "nls-rebind")
                    .start();
        }
    }

    @PluginMethod
    public void setAllowedPackages(PluginCall call) {
        JSArray packages = call.getArray("packages");
        Set<String> set = new HashSet<>();
        if (packages != null) {
            try {
                for (int i = 0; i < packages.length(); i++) {
                    String pkg = packages.getString(i);
                    if (pkg != null && !pkg.trim().isEmpty()) {
                        set.add(pkg.trim());
                    }
                }
            } catch (JSONException e) {
                call.reject(e.getMessage());
                return;
            }
        }
        NotificationInterceptStore.setAllowedPackages(getContext(), set);
        call.resolve();
    }

    /**
     * Sync SMS sender → package map from JS ({@code senders: [{sender, packageName}, ...]}).
     */
    @PluginMethod
    public void setAllowedSmsSenders(PluginCall call) {
        JSArray senders = call.getArray("senders");
        Map<String, String> map = new HashMap<>();
        if (senders != null) {
            try {
                for (int i = 0; i < senders.length(); i++) {
                    JSONObject row = senders.getJSONObject(i);
                    String sender = row.optString("sender", "");
                    String pkg = row.optString("packageName", "");
                    if (!sender.trim().isEmpty() && !pkg.trim().isEmpty()) {
                        map.put(sender.trim(), pkg.trim());
                    }
                }
            } catch (JSONException e) {
                call.reject(e.getMessage());
                return;
            }
        }
        NotificationInterceptStore.setAllowedSmsSenders(getContext(), map);
        call.resolve();
    }

    @PluginMethod
    public void getSmsPermissionStatus(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("granted", hasSmsPermission());
        call.resolve(ret);
    }

    @PluginMethod
    public void requestSmsPermission(PluginCall call) {
        if (hasSmsPermission()) {
            JSObject ret = new JSObject();
            ret.put("granted", true);
            call.resolve(ret);
            return;
        }
        requestPermissionForAlias("sms", call, "smsPermissionCallback");
    }

    @PermissionCallback
    private void smsPermissionCallback(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("granted", getPermissionState("sms") == PermissionState.GRANTED);
        call.resolve(ret);
    }

    @PluginMethod
    public void openAppPermissionSettings(PluginCall call) {
        Intent intent = new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS);
        intent.setData(Uri.parse("package:" + getContext().getPackageName()));
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        getContext().startActivity(intent);
        call.resolve();
    }

    @PluginMethod
    public void getCaptureState(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("captureEnabled", NotificationInterceptStore.isCaptureEnabled(getContext()));
        ret.put("notificationAccess", hasNotificationAccess());
        ret.put("smsPermission", hasSmsPermission());
        JSArray pkgs = new JSArray();
        for (String pkg : NotificationInterceptStore.allowedPackages(getContext())) {
            pkgs.put(pkg);
        }
        ret.put("allowedPackages", pkgs);
        call.resolve(ret);
    }

    @PluginMethod
    public void consumePending(PluginCall call) {
        JSONArray arr = NotificationInterceptStore.consumeAll(getContext());
        call.resolve(wrapItems(arr));
    }

    @PluginMethod
    public void peekPending(PluginCall call) {
        JSONArray arr = NotificationInterceptStore.peekAll(getContext());
        call.resolve(wrapItems(arr));
    }

    @PluginMethod
    public void acknowledgePending(PluginCall call) {
        JSArray keys = call.getArray("dedupeKeys");
        Set<String> set = new HashSet<>();
        if (keys != null) {
            try {
                for (int i = 0; i < keys.length(); i++) {
                    String key = keys.getString(i);
                    if (key != null && !key.trim().isEmpty()) {
                        set.add(key.trim());
                    }
                }
            } catch (JSONException e) {
                call.reject(e.getMessage());
                return;
            }
        }
        NotificationInterceptStore.acknowledge(getContext(), set);
        call.resolve();
    }

    /**
     * Returns history as a JSON array string ({@code itemsJson}) so the WebView
     * does not depend on nested JSArray bridging (which can hang on some OEM WebViews).
     */
    @PluginMethod
    public void listHistory(PluginCall call) {
        JSObject ret = new JSObject();
        try {
            // Same prefs as pending queue (+ legacy history file merged in JS bridge).
            ret.put("itemsJson", NotificationInterceptStore.peekHistoryJson(getContext()));
        } catch (RuntimeException e) {
            ret.put("itemsJson", "[]");
        }
        call.resolve(ret);
    }

    /** @deprecated use {@link #listHistory}; kept so older WebView bundles do not hang forever. */
    @PluginMethod
    public void getHistory(PluginCall call) {
        listHistory(call);
    }

    @PluginMethod
    public void clearHistory(PluginCall call) {
        try {
            NotificationInterceptStore.clearHistory(getContext());
            NotificationHistoryStore.clear(getContext());
        } catch (RuntimeException ignored) {
            // ignore
        }
        call.resolve();
    }

    /**
     * Scan notifications currently shown in the shade (not yet dismissed).
     * Rebinds the listener if needed (access granted ≠ service connected).
     */
    @PluginMethod
    public void scanActiveNotifications(PluginCall call) {
        final android.content.Context app = getContext().getApplicationContext();
        new Thread(
                        () -> {
                            int scanned =
                                    BankNotificationListenerService.scanActiveNotifications(app);
                            final JSObject ret = new JSObject();
                            ret.put("scanned", Math.max(scanned, 0));
                            ret.put("listenerConnected", scanned >= 0);
                            ret.put("notificationAccess", hasNotificationAccess());
                            resolveOnMain(call, ret);
                        },
                        "nls-scan")
                .start();
    }

    @PluginMethod
    public void reconnectListener(PluginCall call) {
        final android.content.Context app = getContext().getApplicationContext();
        new Thread(
                        () -> {
                            boolean connected =
                                    BankNotificationListenerService.ensureConnected(app, 5000);
                            final JSObject ret = new JSObject();
                            ret.put("listenerConnected", connected);
                            ret.put("notificationAccess", hasNotificationAccess());
                            resolveOnMain(call, ret);
                        },
                        "nls-rebind")
                .start();
    }

    private static void resolveOnMain(PluginCall call, JSObject ret) {
        new Handler(Looper.getMainLooper()).post(() -> call.resolve(ret));
    }

    @PluginMethod
    public void getListenerState(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("listenerConnected", BankNotificationListenerService.isConnected());
        ret.put("notificationAccess", hasNotificationAccess());
        ret.put("captureEnabled", NotificationInterceptStore.isCaptureEnabled(getContext()));
        ret.put("smsPermission", hasSmsPermission());
        call.resolve(ret);
    }

    private JSObject wrapItems(JSONArray arr) {
        JSObject ret = new JSObject();
        JSArray items = new JSArray();
        for (int i = 0; i < arr.length(); i++) {
            try {
                JSONObject obj = arr.getJSONObject(i);
                JSObject item = new JSObject();
                item.put("packageName", obj.optString("packageName", ""));
                item.put("title", obj.optString("title", ""));
                item.put("text", obj.optString("text", ""));
                item.put("bigText", obj.optString("bigText", ""));
                item.put("postedAt", obj.optLong("postedAt", 0L));
                item.put("dedupeKey", obj.optString("dedupeKey", ""));
                String channel = obj.optString("channel", "");
                if (!channel.isEmpty()) {
                    item.put("channel", channel);
                }
                items.put(item);
            } catch (JSONException ignored) {
                // skip bad row
            }
        }
        ret.put("items", items);
        return ret;
    }

    private boolean hasSmsPermission() {
        return ContextCompat.checkSelfPermission(getContext(), Manifest.permission.RECEIVE_SMS)
                == PackageManager.PERMISSION_GRANTED;
    }

    private boolean hasNotificationAccess() {
        String flat = Settings.Secure.getString(
                getContext().getContentResolver(), "enabled_notification_listeners");
        if (TextUtils.isEmpty(flat)) {
            return false;
        }
        ComponentName expected = new ComponentName(getContext(), BankNotificationListenerService.class);
        for (String entry : flat.split(":")) {
            ComponentName cn = ComponentName.unflattenFromString(entry);
            if (cn != null && cn.equals(expected)) {
                return true;
            }
        }
        return false;
    }
}
