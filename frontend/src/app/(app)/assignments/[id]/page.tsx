import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Bài tập" };

export default function Page() {
  return <RouteStub title="Bài tập" />;
}
