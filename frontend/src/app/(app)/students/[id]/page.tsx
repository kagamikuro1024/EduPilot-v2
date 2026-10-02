import type { Metadata } from "next";
import { StudentProfile } from "@/features/students/StudentProfile";

export const metadata: Metadata = { title: "Hồ sơ 360" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <StudentProfile id={id} />;
}
