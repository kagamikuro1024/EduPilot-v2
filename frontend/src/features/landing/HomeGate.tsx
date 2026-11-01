"use client";

import { useEffect } from "react";
import { TodayScreen } from "@/features/today/TodayScreen";
import { AppShell } from "@/shared/shell/AppShell";
import { useAuth } from "@/shared/session/AuthProvider";
import { SessionProvider } from "@/shared/session/session";
import s from "./Landing.module.css";
import { setReturning } from "@/shared/session/returning";

/**
 * `/`: đã đăng nhập → khung ứng dụng + "Hôm nay"; còn lại → giới thiệu (máy chủ đã vẽ sẵn, truyền vào dạng `landing`) hoặc khung "Hôm nay" chờ phiên (`pre`) cho người quay lại.
 * Hai khung cùng nằm trong HTML; CSS chọn khung nào hiện theo `html[data-returning]` (đặt bởi script trước khi vẽ). Phiên xác định là chưa đăng nhập thì gỡ dấu → giới thiệu hiện.
 */
export function HomeGate({ landing, pre }: { landing: React.ReactNode; pre: React.ReactNode }) {
  const auth = useAuth();
  const anonymous = auth.status === "anonymous" || auth.status === "revoked";
  useEffect(() => {
    if (anonymous) setReturning(false);
  }, [anonymous]);
  useEffect(() => {
    if (auth.status === "authenticated") document.title = "Hôm nay · EduPilot";
  }, [auth.status]);

  if (auth.status === "authenticated") {
    return (
      <SessionProvider fullName={auth.user?.full_name}>
        <AppShell>
          <TodayScreen />
        </AppShell>
      </SessionProvider>
    );
  }
  return (
    <>
      <div className={s.landingRoot} data-part="landing-root">{landing}</div>
      <div className={s.preRoot} data-part="home-pre">{pre}</div>
    </>
  );
}
