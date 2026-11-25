package vn.edupilot.app;

import android.os.Bundle;
import android.webkit.WebView;
import androidx.activity.OnBackPressedCallback;
import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(EduPilotAttestPlugin.class); // trước super.onCreate: Capacitor nạp danh sách plugin ở đó
        super.onCreate(savedInstanceState);
        // Back: còn lịch sử trong WebView (khung quét /checkin đẩy một mục, US-AS-15 AC7a) thì lùi một mục; hết lịch sử thì hành vi mặc định của hệ thống.
        getOnBackPressedDispatcher().addCallback(this, new OnBackPressedCallback(true) {
            @Override
            public void handleOnBackPressed() {
                WebView web = getBridge().getWebView();
                if (web != null && web.canGoBack()) {
                    web.goBack();
                    return;
                }
                setEnabled(false);
                getOnBackPressedDispatcher().onBackPressed();
            }
        });
    }
}
