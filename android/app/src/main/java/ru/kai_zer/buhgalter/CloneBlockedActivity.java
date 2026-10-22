package ru.kai_zer.buhgalter;

import android.app.Activity;
import android.os.Bundle;
import android.widget.Button;

/** Shown instead of the WebView when the process is an OEM Dual Apps clone. */
public class CloneBlockedActivity extends Activity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_clone_blocked);
        Button close = findViewById(R.id.clone_blocked_close);
        if (close != null) {
            close.setOnClickListener(v -> finishAffinity());
        }
    }
}
