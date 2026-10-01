import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { AppShell } from "@/shared/shell/AppShell";
import { COURSE_COOKIE, ROLE_COOKIE, parseCourse, parseRole } from "@/shared/session/cookies";
import { SessionProvider } from "@/shared/session/session";

// Phiên mô phỏng đọc từ cookie ở server: không có vai trò → về /login (bản thật: JWT, phase P2).
export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const jar = await cookies();
  const role = parseRole(jar.get(ROLE_COOKIE)?.value);
  if (!role) redirect("/login");
  return (
    <SessionProvider initialRole={role} initialCourseId={parseCourse(jar.get(COURSE_COOKIE)?.value)}>
      <AppShell>{children}</AppShell>
    </SessionProvider>
  );
}
