import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Làm bài" };

export default function Page() {
  return <RouteStub title="Làm bài" />;
}
