import "@/shared/styles/panel-variants.css"; // US-UI-01: tệp tạm, chỉ nạp ở /dev/panels
import { AuthGate } from "@/shared/session/AuthGate";
import { AppShell } from "@/shared/shell/AppShell";

// Trang mẫu nằm trong khung ứng dụng THẬT (cùng AuthGate + AppShell với nhóm (app)): đăng nhập đúng vai để khung khớp mẫu.
export default function PanelsLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate>
      <AppShell>{children}</AppShell>
    </AuthGate>
  );
}
