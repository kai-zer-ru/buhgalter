package ru.kai_zer.buhgalter.widgets;

import android.appwidget.AppWidgetManager;
import android.content.Intent;
import android.os.Bundle;
import android.util.TypedValue;
import android.view.View;
import android.widget.AdapterView;
import android.widget.ArrayAdapter;
import android.widget.ListView;
import android.widget.TextView;

import androidx.appcompat.app.AppCompatActivity;
import androidx.core.graphics.Insets;
import androidx.core.view.ViewCompat;
import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsCompat;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.ArrayList;
import java.util.List;

import ru.kai_zer.buhgalter.R;

public class AccountWidgetConfigureActivity extends AppCompatActivity {
    private int appWidgetId = AppWidgetManager.INVALID_APPWIDGET_ID;
    private final List<String> accountIds = new ArrayList<>();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setResult(RESULT_CANCELED);

        Intent intent = getIntent();
        Bundle extras = intent.getExtras();
        if (extras != null) {
            appWidgetId =
                    extras.getInt(
                            AppWidgetManager.EXTRA_APPWIDGET_ID, AppWidgetManager.INVALID_APPWIDGET_ID);
        }
        if (appWidgetId == AppWidgetManager.INVALID_APPWIDGET_ID) {
            finish();
            return;
        }

        WindowCompat.setDecorFitsSystemWindows(getWindow(), false);
        setContentView(R.layout.widget_account_configure);

        View root = findViewById(R.id.widget_configure_root);
        final int pad =
                Math.round(
                        TypedValue.applyDimension(
                                TypedValue.COMPLEX_UNIT_DIP, 16f, getResources().getDisplayMetrics()));
        ViewCompat.setOnApplyWindowInsetsListener(
                root,
                (v, windowInsets) -> {
                    Insets bars = windowInsets.getInsets(WindowInsetsCompat.Type.systemBars());
                    v.setPadding(
                            bars.left + pad, bars.top + pad, bars.right + pad, bars.bottom + pad);
                    return WindowInsetsCompat.CONSUMED;
                });

        TextView hint = findViewById(R.id.widget_configure_hint);
        ListView list = findViewById(R.id.widget_configure_list);

        JSONObject snap = WidgetSnapshotStore.getSnapshot(this);
        JSONArray accounts = snap != null ? snap.optJSONArray("accounts") : null;
        List<String> labels = new ArrayList<>();
        if (accounts == null || accounts.length() == 0) {
            hint.setText(R.string.widget_account_configure_empty);
            list.setVisibility(View.GONE);
            return;
        }
        hint.setText(R.string.widget_account_configure_hint);
        for (int i = 0; i < accounts.length(); i++) {
            JSONObject a = accounts.optJSONObject(i);
            if (a == null) continue;
            accountIds.add(a.optString("id"));
            String name = a.optString("name", "");
            if (a.optBoolean("is_primary", false)) {
                name = name + " ★";
            }
            labels.add(name);
        }
        list.setAdapter(new ArrayAdapter<>(this, R.layout.widget_account_configure_item, labels));
        list.setOnItemClickListener(this::onPick);
    }

    private void onPick(AdapterView<?> parent, View view, int position, long id) {
        if (position < 0 || position >= accountIds.size()) return;
        String accountId = accountIds.get(position);
        AccountWidgetPrefs.setAccountId(this, appWidgetId, accountId);
        AppWidgetManager manager = AppWidgetManager.getInstance(this);
        new AccountWidgetProvider().onUpdate(this, manager, new int[] {appWidgetId});
        Intent result = new Intent();
        result.putExtra(AppWidgetManager.EXTRA_APPWIDGET_ID, appWidgetId);
        setResult(RESULT_OK, result);
        finish();
    }
}
