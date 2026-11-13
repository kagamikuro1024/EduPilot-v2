import type { Metadata } from "next";
import { InsightsScreen } from "@/features/insights/InsightsScreen";

export const metadata: Metadata = { title: "Lỗ hổng kiến thức" };

export default function Page() {
  return <InsightsScreen />;
}
