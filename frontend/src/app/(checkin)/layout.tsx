import { StashGate } from "@/features/checkin/StashGate";
import { AuthGate } from "@/shared/session/AuthGate";
import { AppShell } from "@/shared/shell/AppShell";

// `/checkin` nằm trong khung ứng dụng như mọi route, nhưng layout riêng cất `#token` TRƯỚC `AuthGate` (DESIGN §14.8.2; US-AS-09 AC2).
export default function CheckinLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <StashGate />
      <AuthGate>
        <AppShell>{children}</AppShell>
      </AuthGate>
    </>
  );
}
