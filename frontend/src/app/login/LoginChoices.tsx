"use client";

import { ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { PEOPLE, ROLE_LABEL, type Role } from "@/mock/core";
import { COURSE_COOKIE, ROLE_COOKIE, writeDemoCookie } from "@/shared/session/cookies";
import s from "./login.module.css";

const ROLES: Array<{ role: Role; sees: string }> = [
  { role: "student", sees: "Hỏi riêng AI, thread công khai, luyện đề, xem điểm của mình" },
  { role: "ta", sees: "Hộp thư hỗ trợ, điểm danh, duyệt bài chấm nháp" },
  { role: "teacher", sees: "Hai lớp INT1006, sổ điểm, công thức điểm, công bố" },
  { role: "admin", sees: "Quan sát AI, cấu hình LLM, lớp học và người dùng" },
];

export function LoginChoices() {
  const router = useRouter();

  function enter(role: Role) {
    writeDemoCookie(ROLE_COOKIE, role);
    writeDemoCookie(COURSE_COOKIE, "int1006-1");
    router.push("/");
  }

  return (
    <ul className={s.list} aria-label="Tài khoản mô phỏng">
      {ROLES.map(({ role, sees }) => (
        <li key={role}>
          <button type="button" className={s.row} onClick={() => enter(role)}>
            <span className={s.text}>
              <span className="ep-item-title">{ROLE_LABEL[role]}</span>
              <span className={s.who}>
                {PEOPLE[role].name} · {PEOPLE[role].email}
              </span>
              <span className={s.sees}>{sees}</span>
            </span>
            <ChevronRight aria-hidden />
          </button>
        </li>
      ))}
    </ul>
  );
}
