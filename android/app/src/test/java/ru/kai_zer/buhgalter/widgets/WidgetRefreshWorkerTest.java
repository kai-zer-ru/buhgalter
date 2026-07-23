package ru.kai_zer.buhgalter.widgets;

import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

public class WidgetRefreshWorkerTest {
    @Test
    public void formatMoney_usesTwoDecimals() {
        assertEquals("100.50 RUB", WidgetRefreshWorker.formatMoney(10050, "RUB"));
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
                        "RUB",
                        "ru");
        JSONArray upcoming = snap.getJSONArray("upcoming");
        assertEquals(2, upcoming.length());
        assertEquals("d1", upcoming.getJSONObject(0).getString("id"));
        assertTrue(snap.getString("total_balance_display").contains("RUB"));
    }
}
