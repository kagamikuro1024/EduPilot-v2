"use client";

import { useSession } from "@/shared/session/session";
import { Page, PageHeader } from "@/shared/ui";
import { DevicesSection } from "./DevicesSection";
import { PasswordSection } from "./PasswordSection";
import { RemindersSection } from "./RemindersSection";

/** Tài khoản và bảo mật: đổi mật khẩu và các thiết bị đang đăng nhập (mọi vai, chỉ tác động tài khoản của chính mình). */
export function SecuritySettings() {
  const { role, realCourseId } = useSession();
  return (
    <Page>
      <PageHeader title="Tài khoản và bảo mật" description="Đổi mật khẩu và xem nơi tài khoản của bạn đang đăng nhập." />
      <PasswordSection />
      <DevicesSection />
      {role === "student" && realCourseId && <RemindersSection />}
    </Page>
  );
}
