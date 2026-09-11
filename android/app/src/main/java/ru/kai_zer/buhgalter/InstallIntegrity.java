package ru.kai_zer.buhgalter;

import android.content.Context;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;

/**
 * Detects Auto Backup / OEM restore after uninstall. Marker lives in
 * {@link Context#getNoBackupFilesDir()} so a compliant backup cannot bring it back
 * together with WebView/localStorage — restored session is wiped.
 */
public final class InstallIntegrity {
    static final String STAMP_NAME = "install.stamp";
    static final long NEW_INSTALL_SLACK_MS = 1000L;

    private InstallIntegrity() {}

    public static boolean isNewInstallGeneration(long firstInstallTime, long lastUpdateTime) {
        return Math.abs(lastUpdateTime - firstInstallTime) <= NEW_INSTALL_SLACK_MS;
    }

    /**
     * @param storedInstallTime {@code firstInstallTime} from the no-backup stamp, or null if missing
     * @param hasRestoredUserData WebView/prefs already present (OEM backup restore), not an empty first run
     */
    public static boolean shouldWipe(
            Long storedInstallTime,
            long firstInstallTime,
            long lastUpdateTime,
            boolean hasRestoredUserData) {
        if (storedInstallTime != null && storedInstallTime == firstInstallTime) {
            return false;
        }
        if (!hasRestoredUserData) {
            return false;
        }
        if (storedInstallTime != null) {
            return true;
        }
        return isNewInstallGeneration(firstInstallTime, lastUpdateTime);
    }

    public static boolean dirHasEntries(File dir) {
        if (dir == null || !dir.isDirectory()) {
            return false;
        }
        File[] children = dir.listFiles();
        return children != null && children.length > 0;
    }

    public static void ensureFreshInstall(Context context) {
        PackageTimes times = readPackageTimes(context);
        if (times == null) {
            return;
        }
        File stamp = new File(context.getNoBackupFilesDir(), STAMP_NAME);
        Long stored = readStamp(stamp);
        if (shouldWipe(
                stored,
                times.firstInstallTime,
                times.lastUpdateTime,
                hasRestoredUserData(context))) {
            wipeAppData(context);
        }
        writeStamp(stamp, times.firstInstallTime);
    }

    static boolean hasRestoredUserData(Context context) {
        String dataDirPath = context.getApplicationInfo().dataDir;
        if (dataDirPath == null) {
            return false;
        }
        File dataDir = new File(dataDirPath);
        return dirHasEntries(new File(dataDir, "app_webview"))
                || dirHasEntries(new File(dataDir, "shared_prefs"));
    }

    static void wipeAppData(Context context) {
        String dataDirPath = context.getApplicationInfo().dataDir;
        if (dataDirPath != null) {
            File dataDir = new File(dataDirPath);
            File[] children = dataDir.listFiles();
            if (children != null) {
                for (File child : children) {
                    String name = child.getName();
                    if ("lib".equals(name)
                            || "no_backup".equals(name)
                            || "app_no_backup".equals(name)) {
                        continue;
                    }
                    deleteRecursively(child);
                }
            }
        }
        deleteRecursively(context.getCacheDir());
        // HyperOS can block tens of seconds on getExternalFilesDir — never on the start thread.
        final Context app = context.getApplicationContext();
        new Thread(
                () -> {
                    File ext = app.getExternalFilesDir(null);
                    if (ext != null) {
                        deleteRecursively(ext);
                    }
                    File extCache = app.getExternalCacheDir();
                    if (extCache != null) {
                        deleteRecursively(extCache);
                    }
                },
                "wipe-ext")
                .start();
    }

    private static void deleteRecursively(File file) {
        if (file == null || !file.exists()) {
            return;
        }
        if (file.isDirectory()) {
            File[] children = file.listFiles();
            if (children != null) {
                for (File child : children) {
                    deleteRecursively(child);
                }
            }
        }
        //noinspection ResultOfMethodCallIgnored
        file.delete();
    }

    private static PackageTimes readPackageTimes(Context context) {
        try {
            PackageInfo info =
                    context.getPackageManager().getPackageInfo(context.getPackageName(), 0);
            return new PackageTimes(info.firstInstallTime, info.lastUpdateTime);
        } catch (PackageManager.NameNotFoundException e) {
            return null;
        }
    }

    private static Long readStamp(File stamp) {
        if (!stamp.isFile()) {
            return null;
        }
        try (BufferedReader reader =
                new BufferedReader(
                        new InputStreamReader(new FileInputStream(stamp), StandardCharsets.UTF_8))) {
            String line = reader.readLine();
            if (line == null || line.isEmpty()) {
                return null;
            }
            return Long.parseLong(line.trim());
        } catch (IOException | NumberFormatException e) {
            return null;
        }
    }

    private static void writeStamp(File stamp, long firstInstallTime) {
        File parent = stamp.getParentFile();
        if (parent != null && !parent.exists()) {
            //noinspection ResultOfMethodCallIgnored
            parent.mkdirs();
        }
        try (FileOutputStream out = new FileOutputStream(stamp)) {
            out.write(Long.toString(firstInstallTime).getBytes(StandardCharsets.UTF_8));
        } catch (IOException ignored) {
            // Next launch will retry; failing open is safer than wiping in a loop.
        }
    }

    private static final class PackageTimes {
        final long firstInstallTime;
        final long lastUpdateTime;

        PackageTimes(long firstInstallTime, long lastUpdateTime) {
            this.firstInstallTime = firstInstallTime;
            this.lastUpdateTime = lastUpdateTime;
        }
    }
}
