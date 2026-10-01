import type { Metadata } from "next";
import { InboxView } from "@/features/inbox/InboxView";

export const metadata: Metadata = { title: "Hộp thư hỗ trợ" };

export default function Page() {
  return <InboxView />;
}
