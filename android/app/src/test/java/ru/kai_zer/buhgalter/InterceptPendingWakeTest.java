package ru.kai_zer.buhgalter;

import org.junit.Test;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

public class InterceptPendingWakeTest {
    @Test
    public void shouldMoveTaskToBack_onlyTrueQuietWake() {
        assertTrue(InterceptPendingWake.shouldMoveTaskToBack(true, false, false));
        assertFalse(InterceptPendingWake.shouldMoveTaskToBack(true, true, false));
        assertFalse(InterceptPendingWake.shouldMoveTaskToBack(true, false, true));
        assertFalse(InterceptPendingWake.shouldMoveTaskToBack(false, false, false));
        assertFalse(InterceptPendingWake.shouldMoveTaskToBack(true, true, true));
    }

    @Test
    public void markUserEngaged_clearsQuietFlag() {
        InterceptPendingWake.resetForTests();
        InterceptPendingWake.markUserEngaged();
        assertFalse(InterceptPendingWake.isQuietWakeActive());
        assertTrue(InterceptPendingWake.isUserEngaged());
        InterceptPendingWake.resetForTests();
    }

    @Test
    public void consumeQuietWake_oneShotMatchingToken() {
        InterceptPendingWake.resetForTests();
        long token = InterceptPendingWake.armQuietWakeTokenForTests();

        assertTrue(InterceptPendingWake.consumeQuietWake(true, token));
        assertTrue(InterceptPendingWake.isQuietWakeActive());

        // Second consume / sticky replay must not re-arm.
        assertFalse(InterceptPendingWake.consumeQuietWake(true, token));
        InterceptPendingWake.resetForTests();
    }

    @Test
    public void consumeQuietWake_rejectsStaleOrMissingToken() {
        InterceptPendingWake.resetForTests();
        assertFalse(InterceptPendingWake.consumeQuietWake(true, 42L));
        assertFalse(InterceptPendingWake.isQuietWakeActive());

        InterceptPendingWake.armQuietWakeTokenForTests();
        assertFalse(InterceptPendingWake.consumeQuietWake(true, 0L));
        assertFalse(InterceptPendingWake.isQuietWakeActive());
        assertFalse(InterceptPendingWake.consumeQuietWake(false, 1L));
        InterceptPendingWake.resetForTests();
    }

    @Test
    public void consumeQuietWake_keepsExistingUserEngaged() {
        InterceptPendingWake.resetForTests();
        InterceptPendingWake.markUserEngaged();
        long token = InterceptPendingWake.armQuietWakeTokenForTests();

        assertTrue(InterceptPendingWake.consumeQuietWake(true, token));
        assertTrue(InterceptPendingWake.isQuietWakeActive());
        assertTrue(InterceptPendingWake.isUserEngaged());
        assertFalse(
                InterceptPendingWake.shouldMoveTaskToBack(
                        InterceptPendingWake.isQuietWakeActive(),
                        InterceptPendingWake.isUserEngaged(),
                        false));
        InterceptPendingWake.resetForTests();
    }

    @Test
    public void onActivityStopped_clearsEngagedUnlessQuietActive() {
        InterceptPendingWake.resetForTests();
        InterceptPendingWake.markUserEngaged();
        InterceptPendingWake.onActivityStopped();
        assertFalse(InterceptPendingWake.isUserEngaged());

        InterceptPendingWake.resetForTests();
        InterceptPendingWake.markUserEngaged();
        long token = InterceptPendingWake.armQuietWakeTokenForTests();
        assertTrue(InterceptPendingWake.consumeQuietWake(true, token));
        assertTrue(InterceptPendingWake.isQuietWakeActive());
        assertTrue(InterceptPendingWake.isUserEngaged());
        InterceptPendingWake.onActivityStopped();
        assertTrue(InterceptPendingWake.isUserEngaged());
        InterceptPendingWake.resetForTests();
    }
}
