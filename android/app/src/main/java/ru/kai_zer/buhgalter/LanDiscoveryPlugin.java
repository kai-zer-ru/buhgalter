package ru.kai_zer.buhgalter;

import android.content.Context;
import android.net.nsd.NsdManager;
import android.net.nsd.NsdServiceInfo;
import android.net.wifi.WifiManager;
import android.os.Handler;
import android.os.Looper;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.net.InetAddress;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.atomic.AtomicBoolean;

@CapacitorPlugin(name = "LanDiscovery")
public class LanDiscoveryPlugin extends Plugin {

    static final String SERVICE_TYPE = "_buhgalter._tcp.";

    @PluginMethod
    public void discover(PluginCall call) {
        int timeoutMs = call.getInt("timeoutMs", 4000);
        if (timeoutMs < 500) {
            timeoutMs = 500;
        }
        if (timeoutMs > 15000) {
            timeoutMs = 15000;
        }

        Context context = getContext();
        NsdManager nsdManager = (NsdManager) context.getSystemService(Context.NSD_SERVICE);
        if (nsdManager == null) {
            call.resolve(emptyResult());
            return;
        }

        WifiManager wifiManager =
                (WifiManager) context.getApplicationContext().getSystemService(Context.WIFI_SERVICE);
        final WifiManager.MulticastLock multicastLock;
        if (wifiManager != null) {
            WifiManager.MulticastLock lock = wifiManager.createMulticastLock("buhgalter-mdns");
            lock.setReferenceCounted(true);
            lock.acquire();
            multicastLock = lock;
        } else {
            multicastLock = null;
        }

        Handler handler = new Handler(Looper.getMainLooper());
        Set<String> seen = new LinkedHashSet<>();
        List<JSObject> results = new ArrayList<>();
        AtomicBoolean finished = new AtomicBoolean(false);

        NsdManager.DiscoveryListener listener =
                new NsdManager.DiscoveryListener() {
                    @Override
                    public void onStartDiscoveryFailed(String serviceType, int errorCode) {
                        finish(call, results, nsdManager, this, multicastLock, finished);
                    }

                    @Override
                    public void onStopDiscoveryFailed(String serviceType, int errorCode) {
                        // ignore
                    }

                    @Override
                    public void onDiscoveryStarted(String serviceType) {}

                    @Override
                    public void onDiscoveryStopped(String serviceType) {}

                    @Override
                    public void onServiceFound(NsdServiceInfo serviceInfo) {
                        if (finished.get()) {
                            return;
                        }
                        nsdManager.resolveService(
                                serviceInfo,
                                new NsdManager.ResolveListener() {
                                    @Override
                                    public void onResolveFailed(NsdServiceInfo info, int errorCode) {}

                                    @Override
                                    public void onServiceResolved(NsdServiceInfo info) {
                                        if (finished.get()) {
                                            return;
                                        }
                                        InetAddress host = info.getHost();
                                        int port = info.getPort();
                                        if (host == null || port <= 0) {
                                            return;
                                        }
                                        String hostAddress = host.getHostAddress();
                                        if (hostAddress == null || hostAddress.contains(":")) {
                                            return;
                                        }
                                        String key = hostAddress + ":" + port;
                                        if (!seen.add(key)) {
                                            return;
                                        }
                                        JSObject row = new JSObject();
                                        row.put("host", hostAddress);
                                        row.put("port", port);
                                        synchronized (results) {
                                            results.add(row);
                                        }
                                    }
                                });
                    }

                    @Override
                    public void onServiceLost(NsdServiceInfo serviceInfo) {}
                };

        try {
            nsdManager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, listener);
        } catch (Exception e) {
            releaseMulticast(multicastLock);
            call.resolve(emptyResult());
            return;
        }

        int finalTimeoutMs = timeoutMs;
        handler.postDelayed(
                () ->
                        finish(
                                call,
                                results,
                                nsdManager,
                                listener,
                                multicastLock,
                                finished),
                finalTimeoutMs);
    }

    private void finish(
            PluginCall call,
            List<JSObject> results,
            NsdManager nsdManager,
            NsdManager.DiscoveryListener listener,
            WifiManager.MulticastLock multicastLock,
            AtomicBoolean finished) {
        if (!finished.compareAndSet(false, true)) {
            return;
        }
        try {
            nsdManager.stopServiceDiscovery(listener);
        } catch (Exception ignored) {
            // discovery may already be stopped
        }
        releaseMulticast(multicastLock);
        JSObject ret = new JSObject();
        JSArray array = new JSArray();
        synchronized (results) {
            for (JSObject row : results) {
                array.put(row);
            }
        }
        ret.put("servers", array);
        call.resolve(ret);
    }

    private static JSObject emptyResult() {
        JSObject ret = new JSObject();
        ret.put("servers", new JSArray());
        return ret;
    }

    private static void releaseMulticast(WifiManager.MulticastLock lock) {
        if (lock == null) {
            return;
        }
        try {
            if (lock.isHeld()) {
                lock.release();
            }
        } catch (Exception ignored) {
            // ignore
        }
    }
}
