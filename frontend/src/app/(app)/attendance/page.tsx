import type { Metadata } from "next";
import { AttendanceView } from "@/features/attendance/AttendanceView";

export const metadata: Metadata = { title: "Điểm danh" };

export default function Page() {
  return <AttendanceView />;
}
