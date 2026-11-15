import type { Metadata } from "next";
import { StudentAssignmentList } from "@/features/assignments/StudentAssignmentList";

export const metadata: Metadata = { title: "Bài tập" };

export default function Page() {
  return <StudentAssignmentList />;
}
