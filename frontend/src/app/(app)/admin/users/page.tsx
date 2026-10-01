import type { Metadata } from "next";
import { AdminUsers } from "@/features/admin/AdminUsers";

export const metadata: Metadata = { title: "Người dùng" };

export default function Page() {
  return <AdminUsers />;
}
