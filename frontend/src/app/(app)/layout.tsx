import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ThreadsBackground } from "@/features/threads/ThreadsBackground";
import { AppShell } from "@/shared/shell/AppShell";
import { COURSE_COOKIE, PERSON_COOKIE, ROLE_COOKIE, parseCourse, parsePerson, parseRole } from "@/shared/session/cookies";
import { SessionProvider } from "@/shared/session/session";

// Phiên mô phỏng đọc từ cookie ở server: không có vai trò → về /login (bản thật: JWT, phase P2).
export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const jar = await cookies();
  const role = parseRole(jar.get(ROLE_COOKIE)?.value);
  if (!role) redirect("/login");
  return (
    <SessionProvider initialRole={role} initialPersonId={parsePerson(jar.get(PERSON_COOKIE)?.value)} initialCourseId={parseCourse(jar.get(COURSE_COOKIE)?.value)}>
      <AppShell>{children}</AppShell>
      <ThreadsBackground />
    </SessionProvider>
  );
}
