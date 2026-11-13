import type { Metadata } from "next";
import { AnalyticsScreen } from "@/features/analytics/AnalyticsScreen";

export const metadata: Metadata = { title: "Số liệu lớp học" };

export default function Page() {
  return <AnalyticsScreen />;
}
