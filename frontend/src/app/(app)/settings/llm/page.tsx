import type { Metadata } from "next";
import { LlmSettings } from "@/features/settings/LlmSettings";

export const metadata: Metadata = { title: "Cấu hình LLM" };

export default function Page() {
  return <LlmSettings />;
}
