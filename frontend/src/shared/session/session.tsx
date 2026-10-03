"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { COURSES, COURSE_1, STAFF, STUDENTS, type Course, type Person, type Role } from "@/mock/core";
import { isSeedAccount, mockStudentFor } from "@/mock/identity";
import { KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
import { tokenStore } from "@/shared/data/tokenStore";
import { useDemoSlice } from "@/shared/state/demo";
import { useAuth } from "./AuthProvider";
import { ALL_COURSES, COURSE_COOKIE, PERSON_COOKIE, ROLE_COOKIE, writeDemoCookie } from "./cookies";
import { checkToken, type Claims } from "./jwt";

type Session = {
  role: Role;
  user: Person;
  /** mã sinh viên trong mock (sv-1…) khi role = student, ngược lại undefined */
  studentId?: string;
  /** lớp đang chọn; khi `isAll` là lớp đầu tiên (các màn không gộp được dùng nó) */
  course: Course;
  /** các lớp người này có (SV: theo thành viên thật, kể cả vừa được duyệt) */
  courses: Course[];
  /** GV chọn "Tất cả lớp của tôi" */
  isAll: boolean;
  /** false với SV chưa vào lớp nào (D) */
  hasCourse: boolean;
  /** đổi vai (và người, nếu là sinh viên) tại chỗ */
  switchTo: (role: Role, personId?: string) => void;
  setCourse: (id: string) => void;
  /** `jwt`: phiên đăng nhập thật (vai từ claim, cookie `ep_demo_*` bị bỏ qua); `demo`: cookie mô phỏng, chỉ ở build NEXT_PUBLIC_DEV_TOOLS=1. */
  source: "jwt" | "demo";
  /** claim của phiên `jwt` (sub, email, role) — chỉ giải mã, không xác minh */
  identity: Claims | null;
  /** đăng xuất thiết bị này rồi về /login */
  logout: () => void;
};

const SessionContext = createContext<Session | null>(null);

function studentCourseIds(studentId: string, members: MembersState): string[] {
  const base = STUDENTS.find((s) => s.id === studentId)?.courseIds ?? [];
  const extra = Object.entries(members.joined)
    .filter(([, ids]) => ids.includes(studentId))
    .map(([courseId]) => courseId);
  return [...new Set([...base, ...extra])];
}

export function SessionProvider({
  demo,
  fullName,
  children,
}: {
  /** phiên mô phỏng từ cookie (chỉ khi chưa có phiên thật); null/undefined khi đã đăng nhập thật */
  demo?: { role: Role; person: string; course: string } | null;
  /** họ tên thật từ phiên đăng nhập (dùng cho người không có trong bảng ánh xạ mock) */
  fullName?: string;
  children: React.ReactNode;
}) {
  const auth = useAuth();
  const [cookieRole, setRole] = useState<Role>(demo?.role ?? "student");
  const [cookiePersonId, setPersonId] = useState(demo?.person ?? "sv-2");
  const [cookieCourseId, setCourseId] = useState(demo?.course ?? COURSE_1);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const token = useSyncExternalStore(tokenStore.subscribe, tokenStore.get, () => null);
  const identity = useMemo(() => {
    const c = token ? checkToken(token) : null;
    return c?.ok ? c.claims : null;
  }, [token]);
  const [jwtCourseId, setJwtCourseId] = useState(COURSE_1);

  const switchTo = useCallback((next: Role, person?: string) => {
    if (tokenStore.get()) return; // phiên jwt: vai theo claim, không đổi vai mô phỏng
    writeDemoCookie(ROLE_COOKIE, next);
    setRole(next);
    if (next === "student") {
      const p = person ?? "sv-2";
      writeDemoCookie(PERSON_COOKIE, p);
      setPersonId(p);
    }
  }, []);

  const setCourse = useCallback((id: string) => {
    if (tokenStore.get()) {
      setJwtCourseId(id); // không đụng cookie ep_demo_*
      return;
    }
    writeDemoCookie(COURSE_COOKIE, id);
    setCourseId(id);
  }, []);

  const logout = useCallback(() => {
    void auth.logout();
  }, [auth]);

  const value = useMemo<Session>(() => {
    // phiên jwt: vai từ claim, người mock theo email đã xác minh (mock/identity.ts; email lạ → sv-2 / người mock cùng vai) và lớp mock đầu;
    // cookie ep_demo_* bị bỏ qua (SRS FEAT-account-security 7.4). Họ tên thật chỉ thay cho người không có trong bảng ánh xạ.
    const role: Role = identity?.role ?? cookieRole;
    const personId = identity ? mockStudentFor(identity.email) : cookiePersonId;
    const courseId = identity ? jwtCourseId : cookieCourseId;
    const student = STUDENTS.find((s) => s.id === personId) ?? STUDENTS[1];
    const named = identity && fullName && !isSeedAccount(identity.email) ? fullName : undefined;
    const user: Person =
      role === "student"
        ? { id: student.id, name: named ?? student.name, email: identity?.email || student.email }
        : { ...STAFF[role], name: named ?? STAFF[role].name, email: identity?.email || STAFF[role].email };
    // TA chỉ phụ trách lớp 1 (SRS 4.1); GV phụ trách cả hai; Admin thấy tất cả.
    const ids = role === "student" ? studentCourseIds(student.id, members) : role === "ta" ? [COURSE_1] : COURSES.map((c) => c.id);
    const courses = COURSES.filter((c) => ids.includes(c.id));
    const isAll = courseId === ALL_COURSES && role === "teacher";
    const course = courses.find((c) => c.id === courseId) ?? courses[0] ?? COURSES[0];
    return {
      role, user, studentId: role === "student" ? student.id : undefined, course, courses, isAll, hasCourse: courses.length > 0, switchTo, setCourse,
      source: identity ? "jwt" : "demo", identity, logout,
    };
  }, [identity, fullName, cookieRole, cookiePersonId, cookieCourseId, jwtCourseId, members, switchTo, setCourse, logout]);

  useCourseDeepLink(value.courses, setCourse);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

/**
 * Liên kết sâu `?course=<id>` (SRS 4.9): lớp đó thành lớp đang chọn rồi BỎ tham số khỏi URL
 * (giữ các tham số khác, ví dụ `?tab=pending`). Id lạ hoặc ngoài quyền của vai: chỉ bỏ tham số.
 */
function useCourseDeepLink(courses: Course[], setCourse: (id: string) => void) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const wanted = params.get("course");
  const allowed = courses.some((c) => c.id === wanted);

  useEffect(() => {
    if (!wanted) return;
    if (allowed) setCourse(wanted);
    const rest = new URLSearchParams(params.toString());
    rest.delete("course");
    const query = rest.toString();
    router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
  }, [wanted, allowed, params, pathname, router, setCourse]);
}

export function useSession() {
  const s = useContext(SessionContext);
  if (!s) throw new Error("useSession cần nằm trong SessionProvider");
  return s;
}
