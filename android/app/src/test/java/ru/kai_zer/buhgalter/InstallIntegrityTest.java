package ru.kai_zer.buhgalter;

import org.junit.Test;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

public class InstallIntegrityTest {
    @Test
    public void newInstallGeneration_sameTimestamps() {
        assertTrue(InstallIntegrity.isNewInstallGeneration(1_000L, 1_000L));
        assertTrue(InstallIntegrity.isNewInstallGeneration(1_000L, 1_500L));
        assertFalse(InstallIntegrity.isNewInstallGeneration(1_000L, 3_000L));
    }

    @Test
    public void shouldWipe_upgradeKeepsData() {
        long first = 1_700_000_000_000L;
        long updated = first + 86_400_000L;
        assertFalse(InstallIntegrity.shouldWipe(null, first, updated, true));
    }

    @Test
    public void shouldWipe_emptyFirstRunNeverWipes() {
        long now = 1_800_000_000_000L;
        assertFalse(InstallIntegrity.shouldWipe(null, now, now, false));
    }

    @Test
    public void shouldWipe_reinstallWithRestoredData() {
        long now = 1_800_000_000_000L;
        assertTrue(InstallIntegrity.shouldWipe(null, now, now, true));
    }

    @Test
    public void shouldWipe_validMarker() {
        long first = 1_700_000_000_000L;
        assertFalse(InstallIntegrity.shouldWipe(first, first, first + 1000L, true));
    }

    @Test
    public void shouldWipe_staleMarkerFromPreviousInstall() {
        long oldFirst = 1_700_000_000_000L;
        long newFirst = 1_800_000_000_000L;
        assertTrue(InstallIntegrity.shouldWipe(oldFirst, newFirst, newFirst, true));
        assertFalse(InstallIntegrity.shouldWipe(oldFirst, newFirst, newFirst, false));
    }
}
