import type { Metadata } from "next";
import { Suspense } from "react";
import { AttendanceDisplay } from "@/features/attendance/display/AttendanceDisplay";

export const metadata: Metadata = { title: "Màn chiếu điểm danh" };

export default function Page() {
  return (
    <Suspense fallback={null}>
      <AttendanceDisplay />
    </Suspense>
  );
}
