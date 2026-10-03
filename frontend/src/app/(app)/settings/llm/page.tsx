import type { Metadata } from "next";
import { LlmSettings } from "@/features/settings/llm/LlmSettings";

export const metadata: Metadata = { title: "Cấu hình LLM" };

export default function Page() {
  return <LlmSettings />;
}
