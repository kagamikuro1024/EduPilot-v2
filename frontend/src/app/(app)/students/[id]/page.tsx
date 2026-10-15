import type { Metadata } from "next";
import { StudentProfileScreen } from "@/features/students/StudentProfileScreen";

export const metadata: Metadata = { title: "Hồ sơ 360" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <StudentProfileScreen id={id} />;
}
