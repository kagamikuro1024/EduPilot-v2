import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Lớp học" };

export default function Page() {
  return <RouteStub title="Lớp học" />;
}
