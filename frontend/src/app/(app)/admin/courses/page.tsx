import type { Metadata } from "next";
import { AdminCourses } from "@/features/admin/AdminCourses";

export const metadata: Metadata = { title: "Lớp học" };

export default function Page() {
  return <AdminCourses />;
}
