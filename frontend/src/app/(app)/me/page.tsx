import type { Metadata } from "next";
import { RouteStub } from "@/app/(app)/RouteStub";

export const metadata: Metadata = { title: "Kết quả của tôi" };

export default function Page() {
  return <RouteStub title="Kết quả của tôi" />;
}
