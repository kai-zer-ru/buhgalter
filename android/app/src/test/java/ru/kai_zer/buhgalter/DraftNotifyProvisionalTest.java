package ru.kai_zer.buhgalter;

import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

public class DraftNotifyProvisionalTest {
    @Test
    public void extractAmount_requiresCurrency_skipsPercentAndBalance() {
        assertEquals(
                "500",
                DraftNotifyProvisional.extractAmount(
                        "Перевод по СБП 500 ₽\nОльга Викторовна К.\nДоступно 26 867,01 ₽"));
        assertEquals("1256.00", DraftNotifyProvisional.extractAmount("Списали 1256.00 ₽ . Обновлённый"));
        assertEquals("", DraftNotifyProvisional.extractAmount("выгодой до 80%. Купить сейчас"));
        assertEquals("", DraftNotifyProvisional.extractAmount("Доступно 26 867,01 ₽"));
    }

    @Test
    public void shouldShow_wbSbpAndInstallment_notMarketing() {
        assertTrue(
                DraftNotifyProvisional.shouldShow(
                        "com.wildberries.ru",
                        "WB Кошелёк",
                        "Перевод по СБП 500 ₽\nОльга Викторовна К.\nДоступно 26 867,01 ₽",
                        ""));
        assertTrue(
                DraftNotifyProvisional.shouldShow(
                        "com.wildberries.ru",
                        "Оплата частями",
                        "Списали 1256.00 ₽ . Обновлённый график платежей",
                        ""));
        assertFalse(
                DraftNotifyProvisional.shouldShow(
                        "com.wildberries.ru",
                        "Можно забирать 📦",
                        "Заказ в пункте выдачи, загляните за ним сегодня",
                        ""));
        assertFalse(
                DraftNotifyProvisional.shouldShow(
                        "com.wildberries.ru",
                        "Куртки и пуховики от 2 184 ₽",
                        "Тёплые, стильные и с хорошими отзывами",
                        ""));
        assertFalse(
                DraftNotifyProvisional.shouldShow(
                        "ru.ozon.app.android",
                        "Пиджак приталенный Befree",
                        "Или джемпер Outventure с выгодой до 80%. Купить сейчас",
                        ""));
        assertFalse(
                DraftNotifyProvisional.shouldShow(
                        "ru.ozon.app.android",
                        "Ух ты, что это? 🤩",
                        "+1000 ₽ у вас на счёте. Потратьте на покупку одежды",
                        ""));
        assertFalse(
                DraftNotifyProvisional.shouldQueuePackage(
                        "ru.ozon.app.android",
                        "Ух ты, что это? 🤩",
                        "+1000 ₽ у вас на счёте. Потратьте на покупку одежды",
                        ""));
    }

    @Test
    public void shouldQueuePackage_marketplaceNeedsWalletOrOp() {
        assertTrue(
                DraftNotifyProvisional.shouldQueuePackage(
                        "com.wildberries.ru", "WB Кошелёк", "Перевод по СБП 500 ₽", ""));
        assertFalse(
                DraftNotifyProvisional.shouldQueuePackage(
                        "com.wildberries.ru",
                        "Личное приглашение 📩",
                        "В подборку с вау-ценами ->",
                        ""));
        assertTrue(
                DraftNotifyProvisional.shouldQueuePackage(
                        "ru.sberbankmobile", "Покупка", "Покупка 100 ₽ Shop", ""));
    }
}
