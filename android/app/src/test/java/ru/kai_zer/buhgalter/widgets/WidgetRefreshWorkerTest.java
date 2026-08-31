package ru.kai_zer.buhgalter.widgets;

import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

public class WidgetRefreshWorkerTest {
    @Test
    public void formatMoney_usesSpacesAndRuble() {
        assertEquals("100.50 ₽", WidgetRefreshWorker.formatMoney(10050, "RUB"));
        assertEquals("1 000.50 ₽", WidgetRefreshWorker.formatMoney(100050, "RUB"));
        assertEquals("-50.00 ₽", WidgetRefreshWorker.formatMoney(-5000, "RUB"));
        assertEquals("10.00 USD", WidgetRefreshWorker.formatMoney(1000, "USD"));
    }

    @Test
    public void formatMoneyDisplay_reformatsApiStrings() {
        assertEquals("1 500.00 ₽", WidgetRefreshWorker.formatMoneyDisplay("1500.00", "RUB"));
        assertEquals("200.00 ₽", WidgetRefreshWorker.formatMoneyDisplay("200.00", "RUB"));
    }

    @Test
    public void buildSnapshot_includesUpcomingSorted() throws Exception {
        JSONObject dashboard = new JSONObject();
        dashboard.put("total_balance", 1000);
        dashboard.put("total_forecast", 1000);
        dashboard.put("accounts", new JSONArray());

        JSONObject budget = new JSONObject();
        budget.put("items", new JSONArray());

        JSONArray credits = new JSONArray();
        JSONObject c = new JSONObject();
        c.put("id", "c1");
        c.put("name", "Loan");
        c.put("status", "active");
        c.put("next_payment_date", "2026-09-01");
        c.put("next_payment_amount", 25000);
        c.put("debit_account_name", "Main");
        credits.put(c);

        JSONArray debts = new JSONArray();
        JSONObject d = new JSONObject();
        d.put("id", "d1");
        d.put("debtor_id", "p1");
        d.put("debtor_name", "Ivan");
        d.put("direction", "borrowed");
        d.put("due_date", "2026-08-01");
        d.put("amount", 5000);
        d.put("amount_display", "50.00");
        d.put("is_settled", false);
        debts.put(d);

        JSONObject snap =
                WidgetRefreshWorker.buildSnapshot(
                        dashboard,
                        new JSONArray(),
                        budget,
                        credits,
                        debts,
                        new JSONArray(),
                        new JSONArray(),
                        new JSONArray(),
                        "RUB",
                        "ru");
        JSONArray upcoming = snap.getJSONArray("upcoming");
        assertEquals(2, upcoming.length());
        assertEquals("d1", upcoming.getJSONObject(0).getString("id"));
        assertTrue(snap.getString("total_balance_display").contains("₽"));
        assertEquals("0.00 ₽", snap.getString("cash_display"));
    }

    @Test
    public void buildSnapshot_includesSubscriptionsAndRecurring() throws Exception {
        JSONObject dashboard = new JSONObject();
        dashboard.put("total_balance", 0);
        dashboard.put("total_forecast", 0);

        JSONObject budget = new JSONObject();
        budget.put("items", new JSONArray());

        JSONArray subscriptions = new JSONArray();
        JSONObject s = new JSONObject();
        s.put("id", "sub1");
        s.put("name", "Netflix");
        s.put("active", true);
        s.put("next_run_at", "2026-08-15T10:00:00Z");
        s.put("amount", 1500);
        s.put("amount_display", "15.00");
        s.put("account_name", "Card");
        subscriptions.put(s);

        JSONArray recurring = new JSONArray();
        JSONObject r = new JSONObject();
        r.put("id", "rec1");
        r.put("description", "Salary");
        r.put("active", true);
        r.put("next_run_at", "2026-08-05T09:00:00Z");
        r.put("amount", 100000);
        r.put("amount_display", "1000.00");
        r.put("account_name", "Main");
        r.put("category_name", "Income");
        recurring.put(r);

        JSONObject snap =
                WidgetRefreshWorker.buildSnapshot(
                        dashboard,
                        new JSONArray(),
                        budget,
                        new JSONArray(),
                        new JSONArray(),
                        new JSONArray(),
                        subscriptions,
                        recurring,
                        "RUB",
                        "ru");
        JSONArray upcoming = snap.getJSONArray("upcoming");
        assertEquals(2, upcoming.length());
        assertEquals("rec1", upcoming.getJSONObject(0).getString("id"));
        assertEquals("subscription", upcoming.getJSONObject(1).getString("kind"));
        assertEquals("/subscriptions", upcoming.getJSONObject(1).getString("route"));
    }

    @Test
    public void buildSnapshot_includesFundsByType() throws Exception {
        JSONObject dashboard = new JSONObject();
        dashboard.put("total_balance", 0);
        dashboard.put("total_forecast", 0);
        JSONObject cards = new JSONObject();
        cards.put("total_balance", 1001400);
        dashboard.put("credit_cards_summary", cards);

        JSONArray accounts = new JSONArray();
        JSONObject cash = new JSONObject();
        cash.put("id", "c1");
        cash.put("name", "Cash");
        cash.put("type", "cash");
        cash.put("status", "active");
        cash.put("balance", 50000);
        cash.put("is_primary", false);
        accounts.put(cash);
        JSONObject bank = new JSONObject();
        bank.put("id", "b1");
        bank.put("name", "Bank");
        bank.put("type", "bank");
        bank.put("status", "active");
        bank.put("balance", 4099253);
        bank.put("is_primary", true);
        accounts.put(bank);

        JSONObject budget = new JSONObject();
        budget.put("items", new JSONArray());

        JSONObject snap =
                WidgetRefreshWorker.buildSnapshot(
                        dashboard,
                        accounts,
                        budget,
                        new JSONArray(),
                        new JSONArray(),
                        new JSONArray(),
                        new JSONArray(),
                        new JSONArray(),
                        "RUB",
                        "ru");
        assertEquals("500.00 ₽", snap.getString("cash_display"));
        assertEquals("40 992.53 ₽", snap.getString("bank_display"));
        assertEquals("10 014.00 ₽", snap.getString("credit_funds_display"));
    }
}
