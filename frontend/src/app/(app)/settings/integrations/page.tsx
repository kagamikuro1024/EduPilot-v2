import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Tích hợp" };

export default function Page() {
  return <RouteStub title="Tích hợp" />;
}
