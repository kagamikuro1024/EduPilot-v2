import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Thư viện" };

export default function Page() {
  return <RouteStub title="Thư viện" />;
}
