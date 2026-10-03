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
import { mockCourseFor, readStoredCourse, useMyCourses, writeStoredCourse, type RealCourse } from "./myCourses";

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
  /** lớp THẬT (`GET /me/courses`, chỉ ACTIVE) của phiên `jwt`; null khi phiên mô phỏng hoặc chưa tải được — khi đó dùng `courses` mô phỏng */
  realCourses: RealCourse[] | null;
  /** id lớp thật đang chọn, hoặc "all"; null khi `realCourses` là null */
  realCourseId: string | null;
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
  const [pickedCourseId, setJwtCourseId] = useState<string | null>(null); // id lớp thật hoặc "all"; null = chưa chọn trong phiên này → localStorage, rồi lớp đầu
  const mine = useMyCourses(!!identity);
  const real = useMemo(() => (identity && mine.data ? mine.data.filter((c) => c.enrollment_status === "ACTIVE") : null), [identity, mine.data]);

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
      writeStoredCourse(id);
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
    const student = STUDENTS.find((s) => s.id === personId) ?? STUDENTS[1];
    const named = identity && fullName && !isSeedAccount(identity.email) ? fullName : undefined;
    const user: Person =
      role === "student"
        ? { id: student.id, name: named ?? student.name, email: identity?.email || student.email }
        : { ...STAFF[role], name: named ?? STAFF[role].name, email: identity?.email || STAFF[role].email };
    let courses: Course[];
    let course: Course;
    let isAll: boolean;
    let realCourseId: string | null = null;
    if (identity && real && role !== "admin") {
      const jwtCourseId = pickedCourseId ?? readStoredCourse(); // chỉ chạy ở client sau khi có phiên thật (identity ≠ null)
      // Lớp THẬT quyết định ai thấy lớp nào; màn mô phỏng dùng lớp mô phỏng tương ứng theo class_code (AC13), không gọi API lớp.
      const pick = jwtCourseId === ALL_COURSES && role !== "student" && real.length > 1 ? ALL_COURSES : (real.find((c) => c.id === jwtCourseId) ?? real[0])?.id ?? null;
      realCourseId = pick;
      courses = [...new Map(real.map((c) => mockCourseFor(c.class_code)).map((c) => [c.id, c])).values()];
      isAll = pick === ALL_COURSES;
      const chosen = real.find((c) => c.id === pick);
      course = chosen ? mockCourseFor(chosen.class_code) : (courses[0] ?? COURSES[0]);
    } else {
      // TA chỉ phụ trách lớp 1 (SRS 4.1); GV phụ trách cả hai; Admin thấy tất cả.
      const courseId = identity ? COURSE_1 : cookieCourseId;
      const ids = role === "student" ? studentCourseIds(student.id, members) : role === "ta" ? [COURSE_1] : COURSES.map((c) => c.id);
      courses = COURSES.filter((c) => ids.includes(c.id));
      isAll = courseId === ALL_COURSES && role === "teacher";
      course = courses.find((c) => c.id === courseId) ?? courses[0] ?? COURSES[0];
    }
    return {
      role, user, studentId: role === "student" ? student.id : undefined, course, courses, isAll, hasCourse: courses.length > 0, realCourses: identity ? real : null, realCourseId,
      switchTo, setCourse, source: identity ? "jwt" : "demo", identity, logout,
    };
  }, [identity, fullName, cookieRole, cookiePersonId, cookieCourseId, pickedCourseId, real, members, switchTo, setCourse, logout]);

  // phiên thật chưa tải xong lớp thật (đang tải / lỗi) → chưa xử lý tham số, tránh bỏ nhầm một id hợp lệ
  useCourseDeepLink(value.realCourses ? value.realCourses.map((c) => c.id) : identity && value.role !== "admin" ? null : value.courses.map((c) => c.id), setCourse);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

/**
 * Liên kết sâu `?course=<id>` (SRS 4.9): lớp đó thành lớp đang chọn rồi BỎ tham số khỏi URL
 * (giữ các tham số khác, ví dụ `?tab=pending`). Id lạ hoặc ngoài quyền của vai: chỉ bỏ tham số.
 */
function useCourseDeepLink(courseIds: string[] | null, setCourse: (id: string) => void) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const wanted = params.get("course");
  const ready = courseIds !== null;
  const allowed = !!courseIds?.some((id) => id === wanted);

  useEffect(() => {
    if (!wanted || !ready) return;
    if (allowed) setCourse(wanted);
    const rest = new URLSearchParams(params.toString());
    rest.delete("course");
    const query = rest.toString();
    router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
  }, [wanted, allowed, ready, params, pathname, router, setCourse]);
}

export function useSession() {
  const s = useContext(SessionContext);
  if (!s) throw new Error("useSession cần nằm trong SessionProvider");
  return s;
}
