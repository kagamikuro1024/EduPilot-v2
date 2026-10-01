import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Sổ điểm" };

export default function Page() {
  return <RouteStub title="Sổ điểm" />;
}
