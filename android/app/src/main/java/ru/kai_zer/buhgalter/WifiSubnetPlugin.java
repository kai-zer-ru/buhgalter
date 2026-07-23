package ru.kai_zer.buhgalter;

import android.Manifest;
import android.content.Context;
import android.net.wifi.WifiInfo;
import android.net.wifi.WifiManager;
import android.os.Build;

import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

@CapacitorPlugin(
        name = "WifiSubnet",
        permissions = {
            @Permission(strings = {Manifest.permission.ACCESS_FINE_LOCATION}, alias = "location")
        })
public class WifiSubnetPlugin extends Plugin {

    @PluginMethod
    public void getIpv4Subnet(PluginCall call) {
        try {
            WifiManager wifiManager =
                    (WifiManager) getContext().getApplicationContext().getSystemService(Context.WIFI_SERVICE);
            if (wifiManager == null) {
                call.resolve(null);
                return;
            }
            WifiInfo wifiInfo = wifiManager.getConnectionInfo();
            if (wifiInfo == null) {
                call.resolve(null);
                return;
            }
            int ip = wifiInfo.getIpAddress();
            if (ip == 0) {
                call.resolve(null);
                return;
            }
            String ipStr =
                    String.format(
                            "%d.%d.%d.%d",
                            (ip & 0xff),
                            (ip >> 8 & 0xff),
                            (ip >> 16 & 0xff),
                            (ip >> 24 & 0xff));
            JSObject ret = new JSObject();
            ret.put("ip", ipStr);
            ret.put("prefix", 24);
            call.resolve(ret);
        } catch (Exception e) {
            call.reject(e.getMessage());
        }
    }

    @PluginMethod
    public void requestLocationPermission(PluginCall call) {
        if (getPermissionState("location") == PermissionState.GRANTED) {
            call.resolve();
            return;
        }
        requestPermissionForAlias("location", call, "locationPermissionCallback");
    }

    @PermissionCallback
    private void locationPermissionCallback(PluginCall call) {
        call.resolve();
    }

    @PluginMethod
    public void getSsid(PluginCall call) {
        boolean requestPermission =
                call.getBoolean("requestPermission", false);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            if (getPermissionState("location") != PermissionState.GRANTED) {
                if (requestPermission) {
                    requestPermissionForAlias("location", call, "ssidPermissionCallback");
                    return;
                }
                JSObject denied = new JSObject();
                denied.put("ssid", null);
                denied.put("permissionDenied", true);
                call.resolve(denied);
                return;
            }
        }
        resolveSsid(call);
    }

    @PermissionCallback
    private void ssidPermissionCallback(PluginCall call) {
        if (getPermissionState("location") == PermissionState.GRANTED) {
            resolveSsid(call);
            return;
        }
        JSObject denied = new JSObject();
        denied.put("ssid", null);
        denied.put("permissionDenied", true);
        call.resolve(denied);
    }

    private void resolveSsid(PluginCall call) {
        JSObject ret = new JSObject();
        try {
            WifiManager wifiManager =
                    (WifiManager) getContext().getApplicationContext().getSystemService(Context.WIFI_SERVICE);
            if (wifiManager == null) {
                ret.put("ssid", null);
                ret.put("permissionDenied", false);
                call.resolve(ret);
                return;
            }
            WifiInfo wifiInfo = wifiManager.getConnectionInfo();
            String ssid = wifiInfo != null ? wifiInfo.getSSID() : null;
            if (ssid != null && ssid.length() >= 2 && ssid.startsWith("\"") && ssid.endsWith("\"")) {
                ssid = ssid.substring(1, ssid.length() - 1);
            }
            if (ssid == null
                    || ssid.isEmpty()
                    || "<unknown ssid>".equalsIgnoreCase(ssid)
                    || "0x".equals(ssid)
                    || "unknown ssid".equalsIgnoreCase(ssid)) {
                ssid = null;
            }
            ret.put("ssid", ssid);
            ret.put("permissionDenied", false);
            call.resolve(ret);
        } catch (Exception e) {
            call.reject(e.getMessage());
        }
    }
}
