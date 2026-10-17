import type { Metadata } from "next";
import { StudentsScreen } from "@/features/students/StudentsScreen";

export const metadata: Metadata = { title: "Sinh viên" };

export default function Page() {
  return <StudentsScreen />;
}
