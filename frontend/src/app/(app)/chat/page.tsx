import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Chat riêng" };

export default function Page() {
  return <RouteStub title="Chat riêng" />;
}
