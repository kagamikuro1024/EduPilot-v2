import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Analytics" };

export default function Page() {
  return <RouteStub title="Analytics" />;
}
