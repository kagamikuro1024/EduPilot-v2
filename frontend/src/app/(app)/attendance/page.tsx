import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Điểm danh" };

export default function Page() {
  return <RouteStub title="Điểm danh" />;
}
