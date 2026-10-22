package ru.kai_zer.buhgalter.widgets;

import android.content.SharedPreferences;

import org.junit.Test;

import java.util.Collections;
import java.util.Map;
import java.util.Set;

import static org.junit.Assert.assertSame;

public class WidgetSnapshotStoreTest {
    @Test
    public void resolvePrefs_cachedWins() {
        StubPrefs cached = new StubPrefs();
        StubPrefs fallback = new StubPrefs();
        StubPrefs encrypted = new StubPrefs();
        assertSame(cached, WidgetSnapshotStore.resolvePrefs(cached, fallback, encrypted, true));
    }

    @Test
    public void resolvePrefs_mainThreadNeverTouchesEncrypted() {
        StubPrefs fallback = new StubPrefs();
        StubPrefs encrypted = new StubPrefs();
        assertSame(fallback, WidgetSnapshotStore.resolvePrefs(null, fallback, encrypted, true));
    }

    @Test
    public void resolvePrefs_workerPrefersEncrypted() {
        StubPrefs fallback = new StubPrefs();
        StubPrefs encrypted = new StubPrefs();
        assertSame(encrypted, WidgetSnapshotStore.resolvePrefs(null, fallback, encrypted, false));
    }

    @Test
    public void resolvePrefs_workerFallsBackWhenEncryptedMissing() {
        StubPrefs fallback = new StubPrefs();
        assertSame(fallback, WidgetSnapshotStore.resolvePrefs(null, fallback, null, false));
    }

    private static final class StubPrefs implements SharedPreferences {
        @Override
        public Map<String, ?> getAll() {
            return Collections.emptyMap();
        }

        @Override
        public String getString(String key, String defValue) {
            return defValue;
        }

        @Override
        public Set<String> getStringSet(String key, Set<String> defValues) {
            return defValues;
        }

        @Override
        public int getInt(String key, int defValue) {
            return defValue;
        }

        @Override
        public long getLong(String key, long defValue) {
            return defValue;
        }

        @Override
        public float getFloat(String key, float defValue) {
            return defValue;
        }

        @Override
        public boolean getBoolean(String key, boolean defValue) {
            return defValue;
        }

        @Override
        public boolean contains(String key) {
            return false;
        }

        @Override
        public Editor edit() {
            return null;
        }

        @Override
        public void registerOnSharedPreferenceChangeListener(OnSharedPreferenceChangeListener listener) {}

        @Override
        public void unregisterOnSharedPreferenceChangeListener(
                OnSharedPreferenceChangeListener listener) {}
    }
}
