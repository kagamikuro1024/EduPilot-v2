import type { Metadata } from "next";
import { ObservabilityScreen } from "@/features/observability/ObservabilityScreen";

export const metadata: Metadata = { title: "Quan sát AI" };

export default function Page() {
  return <ObservabilityScreen />;
}
