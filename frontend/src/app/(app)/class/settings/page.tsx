import type { Metadata } from "next";
import { ClassSettings } from "@/features/members/ClassSettings";

export const metadata: Metadata = { title: "Mã và cài đặt tham gia" };

export default function Page() {
  return <ClassSettings />;
}
