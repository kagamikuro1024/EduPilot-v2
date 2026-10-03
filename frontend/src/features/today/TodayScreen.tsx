"use client";

import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader } from "@/shared/ui";
import { AdminToday } from "./AdminToday";
import { StaffToday } from "./StaffToday";
import { StudentToday } from "./StudentToday";

/** "Hôm nay" khác nhau theo vai trong phiên (JWT): mỗi vai một dạng phản hồi và một màn riêng. */
export function TodayScreen() {
  const { role, source } = useSession();
  if (source !== "jwt") {
    // Phiên mô phỏng (chỉ build dev) không có token: "Hôm nay" thật cần đăng nhập.
    return (
      <Page>
        <PageHeader title="Hôm nay" />
        <EmptyState title="Cần đăng nhập">Đăng nhập bằng tài khoản của bạn để xem việc hôm nay.</EmptyState>
      </Page>
    );
  }
  if (role === "student") return <StudentToday />;
  if (role === "admin") return <AdminToday />;
  return <StaffToday />;
}
