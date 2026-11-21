import type { Metadata } from "next";
import { AdminAudit } from "@/features/admin/AdminAudit";

export const metadata: Metadata = { title: "Nhật ký" };

export default function Page() {
  return <AdminAudit />;
}
