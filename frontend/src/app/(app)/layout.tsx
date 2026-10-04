import { ThreadsBackground } from "@/features/threads/ThreadsBackground";
import { AppShell } from "@/shared/shell/AppShell";
import { AuthGate } from "@/shared/session/AuthGate";

// Phiên đăng nhập thật nằm ở bộ nhớ trình duyệt (access token) + cookie httpOnly (refresh): server không biết được, nên cổng chạy ở client.
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGate>
      <AppShell>{children}</AppShell>
      <ThreadsBackground />
    </AuthGate>
  );
}
