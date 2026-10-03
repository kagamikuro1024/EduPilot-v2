import type { Metadata } from "next";
import { GradeScheme } from "@/features/gradebook/GradeScheme";

export const metadata: Metadata = { title: "Công thức điểm" };

export default function Page() {
  return <GradeScheme />;
}
