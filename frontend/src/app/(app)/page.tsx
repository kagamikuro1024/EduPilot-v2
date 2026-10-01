import type { Metadata } from "next";
import { cookies } from "next/headers";
import { AdminHome } from "@/features/today/AdminHome";
import { StaffHome } from "@/features/today/StaffHome";
import { StudentHome } from "@/features/today/StudentHome";
import { ROLE_COOKIE, parseRole } from "@/shared/session/cookies";

export const metadata: Metadata = { title: "Hôm nay" };

// "Hôm nay" khác nhau theo vai: mỗi vai một component riêng ở features/today/.
export default async function Page() {
  const role = parseRole((await cookies()).get(ROLE_COOKIE)?.value);
  if (role === "student") return <StudentHome />;
  if (role === "admin") return <AdminHome />;
  return <StaffHome />;
}
