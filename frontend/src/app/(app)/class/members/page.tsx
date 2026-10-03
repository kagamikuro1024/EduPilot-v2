import type { Metadata } from "next";
import { MembersView } from "@/features/members/MembersView";

export const metadata: Metadata = { title: "Thành viên lớp" };

export default function Page() {
  return <MembersView />;
}
