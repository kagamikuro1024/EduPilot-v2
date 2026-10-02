"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { COURSE_1, DEMO_STUDENT_BLURB, DEMO_STUDENT_IDS, ROLE_LABEL, STAFF, STUDENTS, type Role } from "@/mock/core";
import { ALL_COURSES, COURSE_COOKIE, PERSON_COOKIE, ROLE_COOKIE, writeDemoCookie } from "@/shared/session/cookies";
import s from "./login.module.css";

const ROLES: Array<{ role: Role; sees: string }> = [
  { role: "student", sees: "Hỏi riêng AI, thread công khai, luyện đề, xem điểm của mình" },
  { role: "ta", sees: "Hộp thư hỗ trợ, điểm danh, duyệt bài chấm nháp" },
  { role: "teacher", sees: "Hai lớp An ninh mạng, sổ điểm, công thức điểm, công bố" },
  { role: "admin", sees: "Quan sát AI, cấu hình LLM, lớp học và người dùng" },
];

export function LoginChoices() {
  const router = useRouter();
  const [pickStudent, setPickStudent] = useState(false);

  function enter(role: Role, person?: string) {
    writeDemoCookie(ROLE_COOKIE, role);
    if (person) writeDemoCookie(PERSON_COOKIE, person);
    writeDemoCookie(COURSE_COOKIE, role === "teacher" || role === "ta" ? ALL_COURSES : COURSE_1);
    router.push("/");
  }

  if (pickStudent) {
    return (
      <div className={s.step}>
        <button type="button" className={s.back} onClick={() => setPickStudent(false)}>
          <ChevronLeft aria-hidden /> Chọn vai khác
        </button>
        <ul className={s.list} aria-label="Sinh viên mô phỏng">
          {DEMO_STUDENT_IDS.map((id, i) => {
            const st = STUDENTS.find((x) => x.id === id)!;
            return (
              <li key={id}>
                <button type="button" className={s.row} onClick={() => enter("student", id)}>
                  <span className={s.text}>
                    <span className="ep-item-title">
                      Sinh viên {"ABCD"[i]} · {st.name}
                    </span>
                    <span className={s.who}>
                      {st.code} · {st.email}
                    </span>
                    <span className={s.sees}>{DEMO_STUDENT_BLURB[id]}</span>
                  </span>
                  <ChevronRight aria-hidden />
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    );
  }

  return (
    <ul className={s.list} aria-label="Vai trò mô phỏng">
      {ROLES.map(({ role, sees }) => (
        <li key={role}>
          <button type="button" className={s.row} onClick={() => (role === "student" ? setPickStudent(true) : enter(role))}>
            <span className={s.text}>
              <span className="ep-item-title">{ROLE_LABEL[role]}</span>
              {role !== "student" && (
                <span className={s.who}>
                  {STAFF[role].name} · {STAFF[role].email}
                </span>
              )}
              <span className={s.sees}>{sees}</span>
            </span>
            <ChevronRight aria-hidden />
          </button>
        </li>
      ))}
    </ul>
  );
}
