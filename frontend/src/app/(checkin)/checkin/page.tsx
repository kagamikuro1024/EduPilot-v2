import type { Metadata } from "next";
import { Suspense } from "react";
import { CheckinScreen } from "@/features/checkin/CheckinScreen";

export const metadata: Metadata = { title: "Điểm danh" };

export default function Page() {
  return (
    <Suspense fallback={null}>
      <CheckinScreen />
    </Suspense>
  );
}
