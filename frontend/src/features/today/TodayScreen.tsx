"use client";

import { useSession } from "@/shared/session/session";
import { AdminToday } from "./AdminToday";
import { StaffToday } from "./StaffToday";
import { StudentToday } from "./StudentToday";

/** "Hôm nay" khác nhau theo vai trong phiên (JWT): mỗi vai một dạng phản hồi và một màn riêng. */
export function TodayScreen() {
  const { role } = useSession();
  if (role === "student") return <StudentToday />;
  if (role === "admin") return <AdminToday />;
  return <StaffToday />;
}
