import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Sinh viên" };

export default function Page() {
  return <RouteStub title="Sinh viên" />;
}
