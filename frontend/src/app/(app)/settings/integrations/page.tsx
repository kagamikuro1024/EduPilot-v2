import type { Metadata } from "next";
import { IntegrationsSettings } from "@/features/settings/IntegrationsSettings";

export const metadata: Metadata = { title: "Tích hợp" };

export default function Page() {
  return <IntegrationsSettings />;
}
