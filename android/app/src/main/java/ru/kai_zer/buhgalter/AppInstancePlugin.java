package ru.kai_zer.buhgalter;

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
}
