import type { Metadata } from "next";
import { AttendanceScreen } from "@/features/attendance/AttendanceScreen";

export const metadata: Metadata = { title: "Điểm danh" };

export default function Page() {
  return <AttendanceScreen />;
}
