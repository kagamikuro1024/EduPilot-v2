"use client";

import { createContext, useCallback, useContext, useMemo, useState } from "react";
import { COURSES, PEOPLE, type Course, type Person, type Role } from "@/mock/core";
import { COURSE_COOKIE, ROLE_COOKIE, writeDemoCookie } from "./cookies";

type Session = {
  role: Role;
  user: Person;
  course: Course;
  courses: Course[];
  setRole: (role: Role) => void;
  setCourse: (id: string) => void;
};

const SessionContext = createContext<Session | null>(null);

export function SessionProvider({
  initialRole,
  initialCourseId,
  children,
}: {
  initialRole: Role;
  initialCourseId: string;
  children: React.ReactNode;
}) {
  const [role, setRoleState] = useState<Role>(initialRole);
  const [courseId, setCourseId] = useState(initialCourseId);

  const setRole = useCallback((next: Role) => {
    writeDemoCookie(ROLE_COOKIE, next);
    setRoleState(next);
  }, []);

  const setCourse = useCallback((id: string) => {
    writeDemoCookie(COURSE_COOKIE, id);
    setCourseId(id);
  }, []);

  const value = useMemo<Session>(() => {
    // Sinh viên A chỉ học lớp 1; giảng viên / TA dạy cả hai lớp.
    const courses = role === "student" ? COURSES.filter((c) => c.id === "int1006-1") : COURSES;
    const course = courses.find((c) => c.id === courseId) ?? courses[0];
    return { role, user: PEOPLE[role], course, courses, setRole, setCourse };
  }, [role, courseId, setRole, setCourse]);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession() {
  const s = useContext(SessionContext);
  if (!s) throw new Error("useSession cần nằm trong SessionProvider");
  return s;
}
