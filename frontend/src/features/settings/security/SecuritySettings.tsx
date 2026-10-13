"use client";

import { useSession } from "@/shared/session/session";
import { Page, PageHeader } from "@/shared/ui";
import { DevicesSection } from "./DevicesSection";
import { MailSection } from "./MailSection";
import { PasswordSection } from "./PasswordSection";
import { RemindersSection } from "./RemindersSection";
import { StudyTimeSection } from "./StudyTimeSection";

/** Tài khoản và bảo mật: đổi mật khẩu và các thiết bị đang đăng nhập (mọi vai, chỉ tác động tài khoản của chính mình). */
export function SecuritySettings() {
  const { role, realCourseId } = useSession();
  return (
    <Page>
      <PageHeader title="Tài khoản và bảo mật" description="Đổi mật khẩu và xem nơi tài khoản của bạn đang đăng nhập." />
      <PasswordSection />
      <DevicesSection />
      {(role === "student" || role === "ta" || role === "teacher") && realCourseId && <MailSection role={role} />}
      {role === "student" && realCourseId && <RemindersSection />}
      {role === "student" && realCourseId && <StudyTimeSection />}
    </Page>
  );
}
