import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Tài liệu" };

export default function Page() {
  return <RouteStub title="Tài liệu" />;
}
