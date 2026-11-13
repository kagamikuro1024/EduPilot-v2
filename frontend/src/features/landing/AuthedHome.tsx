"use client";

import { TodayScreen } from "@/features/today/TodayScreen";
import { SessionProvider } from "@/shared/session/session";
import { AppShell } from "@/shared/shell/AppShell";

/** `/` khi đã đăng nhập: khung ứng dụng + "Hôm nay". Nạp riêng (HomeGate) để mã khung, phiên và dữ liệu mô phỏng không nằm trong JS của trang giới thiệu cho khách. */
export default function AuthedHome({ fullName }: { fullName?: string }) {
  return (
    <SessionProvider fullName={fullName}>
      <AppShell>
        <TodayScreen />
      </AppShell>
    </SessionProvider>
  );
}
