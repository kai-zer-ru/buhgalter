package ru.kai_zer.buhgalter;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import org.junit.Test;

public class DraftNotifyHttpTest {
    @Test
    public void toApiTransactionDate_convertsIsoToLayoutUtc() {
        assertEquals(
                "2026-10-16 05:54:00",
                DraftNotifyHttp.toApiTransactionDate("2026-10-16T05:54:00.000Z"));
    }

    @Test
    public void toApiTransactionDate_passesThroughLayout() {
        assertEquals(
                "2026-10-16 05:54:00", DraftNotifyHttp.toApiTransactionDate("2026-10-16 05:54:00"));
    }

    @Test
    public void toApiTransactionDate_emptyUsesNowLayout() {
        String got = DraftNotifyHttp.toApiTransactionDate("");
        assertEquals(19, got.length());
        assertTrue(got.matches("\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}"));
    }
}
