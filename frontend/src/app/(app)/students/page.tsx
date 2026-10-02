import type { Metadata } from "next";
import { StudentsView } from "@/features/students/StudentsView";

export const metadata: Metadata = { title: "Sinh viên" };

export default function Page() {
  return <StudentsView />;
}
