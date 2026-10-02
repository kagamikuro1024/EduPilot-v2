import type { Metadata } from "next";
import { Documents } from "@/features/documents/Documents";

export const metadata: Metadata = { title: "Tài liệu" };

export default function Page() {
  return <Documents />;
}
