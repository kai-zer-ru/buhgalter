package ru.kai_zer.buhgalter;

import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

public class CloneGuardTest {
    @Test
    public void userIdFromUid_primaryAndDualApp() {
        assertEquals(0, CloneGuard.userIdFromUid(10123));
        assertEquals(999, CloneGuard.userIdFromUid(999 * 100_000 + 10123));
    }

    @Test
    public void isDualAppUserId_onlyXiaomiStyle() {
        assertTrue(CloneGuard.isDualAppUserId(999));
        assertFalse(CloneGuard.isDualAppUserId(0));
        assertFalse(CloneGuard.isDualAppUserId(10));
    }

    @Test
    public void dataDirLooksLikeClone_user999() {
        assertTrue(CloneGuard.dataDirLooksLikeClone("/data/user/999/ru.kai_zer.buhgalter"));
        assertTrue(CloneGuard.dataDirLooksLikeClone("/data/user_de/999/ru.kai_zer.buhgalter"));
        assertFalse(CloneGuard.dataDirLooksLikeClone("/data/user/0/ru.kai_zer.buhgalter"));
        assertFalse(CloneGuard.dataDirLooksLikeClone("/data/data/ru.kai_zer.buhgalter"));
        assertTrue(CloneGuard.dataDirLooksLikeClone("/data/data/com.lbe.parallel/parallel_intl/0/ru.kai_zer.buhgalter"));
        assertFalse(CloneGuard.dataDirLooksLikeClone(null));
        assertFalse(CloneGuard.dataDirLooksLikeClone(""));
    }

    @Test
    public void isUnsupportedClone_uidOrPath() {
        assertTrue(CloneGuard.isUnsupportedClone(999 * 100_000 + 50, "/data/user/0/ru.kai_zer.buhgalter"));
        assertTrue(CloneGuard.isUnsupportedClone(10123, "/data/user/999/ru.kai_zer.buhgalter"));
        assertFalse(CloneGuard.isUnsupportedClone(10123, "/data/user/0/ru.kai_zer.buhgalter"));
    }
}
