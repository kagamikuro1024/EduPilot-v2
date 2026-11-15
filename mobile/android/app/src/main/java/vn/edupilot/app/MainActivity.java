package vn.edupilot.app;

import android.os.Bundle;
import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(EduPilotAttestPlugin.class); // trước super.onCreate: Capacitor nạp danh sách plugin ở đó
        super.onCreate(savedInstanceState);
    }
}
