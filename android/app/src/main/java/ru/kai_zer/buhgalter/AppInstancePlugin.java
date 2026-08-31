package ru.kai_zer.buhgalter;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * Identifies this app install (main vs OEM dual-app clone). Used to namespace secure
 * storage keys so parallel instances with the same package name do not overwrite each other.
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
