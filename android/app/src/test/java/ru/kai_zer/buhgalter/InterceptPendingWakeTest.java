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
}
