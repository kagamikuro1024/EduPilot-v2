import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Ngân hàng câu hỏi" };

export default function Page() {
  return <RouteStub title="Ngân hàng câu hỏi" />;
}
