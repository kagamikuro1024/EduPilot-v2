"use client";

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import { COURSES, STAFF, STUDENTS, type Course, type Person, type Role } from "@/mock/core";
import { KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
import { useDemoSlice } from "@/shared/state/demo";
import { ALL_COURSES, COURSE_COOKIE, PERSON_COOKIE, ROLE_COOKIE, writeDemoCookie } from "./cookies";

type Session = {
  role: Role;
  user: Person;
  /** mã sinh viên trong mock (sv-1…) khi role = student, ngược lại undefined */
  studentId?: string;
  /** lớp đang chọn; khi `isAll` là lớp đầu tiên (các màn không gộp được dùng nó) */
  course: Course;
  /** các lớp người này có (SV: theo thành viên thật, kể cả vừa được duyệt) */
  courses: Course[];
  /** GV / TA chọn "Tất cả lớp của tôi" */
  isAll: boolean;
  /** false với SV chưa vào lớp nào (D) */
  hasCourse: boolean;
  /** đổi vai (và người, nếu là sinh viên) tại chỗ */
  switchTo: (role: Role, personId?: string) => void;
  setCourse: (id: string) => void;
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
  initialRole,
  initialPersonId,
  initialCourseId,
  children,
}: {
  initialRole: Role;
  initialPersonId: string;
  initialCourseId: string;
  children: React.ReactNode;
}) {
  const [role, setRole] = useState<Role>(initialRole);
  const [personId, setPersonId] = useState(initialPersonId);
  const [courseId, setCourseId] = useState(initialCourseId);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);

  const switchTo = useCallback((next: Role, person?: string) => {
    writeDemoCookie(ROLE_COOKIE, next);
    setRole(next);
    if (next === "student") {
      const p = person ?? "sv-2";
      writeDemoCookie(PERSON_COOKIE, p);
      setPersonId(p);
    }
  }, []);

  const setCourse = useCallback((id: string) => {
    writeDemoCookie(COURSE_COOKIE, id);
    setCourseId(id);
  }, []);

  const value = useMemo<Session>(() => {
    const student = STUDENTS.find((s) => s.id === personId) ?? STUDENTS[1];
    const user: Person = role === "student" ? { id: student.id, name: student.name, email: student.email } : STAFF[role];
    const ids = role === "student" ? studentCourseIds(student.id, members) : COURSES.map((c) => c.id);
    const courses = COURSES.filter((c) => ids.includes(c.id));
    const isAll = courseId === ALL_COURSES && role !== "student" && role !== "admin";
    const course = courses.find((c) => c.id === courseId) ?? courses[0] ?? COURSES[0];
    return { role, user, studentId: role === "student" ? student.id : undefined, course, courses, isAll, hasCourse: courses.length > 0, switchTo, setCourse };
  }, [role, personId, courseId, members, switchTo, setCourse]);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession() {
  const s = useContext(SessionContext);
  if (!s) throw new Error("useSession cần nằm trong SessionProvider");
  return s;
}
