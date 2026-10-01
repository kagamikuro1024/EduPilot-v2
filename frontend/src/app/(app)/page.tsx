import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Hôm nay" };

export default function Page() {
  return <RouteStub title="Hôm nay" />;
}
