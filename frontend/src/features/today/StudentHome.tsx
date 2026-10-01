"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { NOW, STUDENT_B, fmtLongDate, fmtShortDate, fmtTime, studentById } from "@/mock/core";
import { QUIZ_KEY, QUIZ_SEED, type QuizState } from "@/mock/practice";
import { ASSIGNMENTS, CONTINUE_LEARNING, SESSION_HOURS, recommendationFor, sessionDate, until } from "@/mock/student";
import { CURRENT_SESSION } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  ButtonLink,
  EmptyState,
  Field,
  Input,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
} from "@/shared/ui";
import s from "./StudentHome.module.css";

/** "Hôm nay" của sinh viên: một việc nên làm, dòng thời gian hôm nay, chỗ học dở (DESIGN §14.1). */
export function StudentHome() {
  const { user, course, courses, hasCourse, studentId } = useSession();
  const [quiz] = useDemoSlice<QuizState>(QUIZ_KEY, QUIZ_SEED);
  const student = studentById(studentId ?? "") ?? STUDENT_B;
  const firstName = user.name.split(" ").slice(-1)[0];
  const hours = SESSION_HOURS[course.id] ?? SESSION_HOURS["761987"];
  const quiz01 = ASSIGNMENTS[3];
  const rec = recommendationFor(student, quiz.status === "submitted");

  return (
    <Page>
      <PageHeader
        title={`Chào ${firstName}`}
        description={
          hasCourse
            ? `${fmtLongDate(NOW)} · tuần ${course.week} · ${course.label}`
            : `${fmtLongDate(NOW)} · bạn chưa vào lớp nào`
        }
      />
      <PageState
        loading={
          <>
            <Skeleton lines={2} />
            <Skeleton lines={4} />
          </>
        }
        empty={
          <EmptyState title="Hôm nay bạn không có việc gấp" action={<ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}>
            Không có hạn nộp nào trong 24 giờ tới. Giữ nhịp bằng một lượt luyện ngắn theo chủ đề bạn hay sai.
          </EmptyState>
        }
      >
        {hasCourse ? (
          <>
            <Section title="Việc nên làm bây giờ">
              <ActionList label="Việc nên làm">
                <ActionRow
                  tone="red"
                  href={rec.href}
                  redThread
                  title={rec.title}
                  context={rec.reason}
                  meta={`Khoảng ${rec.minutes} phút`}
                  action={
                    <ButtonLink href={rec.href} variant="primary">
                      {rec.cta}
                    </ButtonLink>
                  }
                />
              </ActionList>
            </Section>

            <Section title="Hôm nay" description={`${course.room} · ${course.schedule}`}>
              <ActionList label="Lịch hôm nay">
                <ActionRow
                  tone="green"
                  title={`Buổi ${CURRENT_SESSION} · ${course.name}`}
                  context={`${hours.start}–${hours.end} · ${course.room}`}
                  meta={<StatusText tone="green">Đang diễn ra</StatusText>}
                  href="/calendar"
                />
                <ActionRow
                  tone="amber"
                  title={`${quiz01.code} đóng lúc ${fmtTime(quiz01.due)} ngày ${fmtShortDate(quiz01.due)}`}
                  context={quiz.status === "submitted" ? "Bạn đã nộp bài" : `Còn ${until(quiz01.due)} · bạn chưa làm`}
                  meta={`${quiz01.minutes} phút · tính điểm`}
                  href="/practice/at-quiz01"
                />
              </ActionList>
            </Section>

            <Section title="Học dở" description="Mở lại chỗ bạn dừng hôm qua.">
              <ActionList label="Học dở">
                {CONTINUE_LEARNING.map((c) => (
                  <ActionRow key={c.href} title={c.title} context={c.context} meta={c.meta} href={c.href} />
                ))}
              </ActionList>
            </Section>
          </>
        ) : (
          <JoinPrompt />
        )}
      </PageState>
      {hasCourse && courses.length > 1 && (
        <p className={s.hint}>Bạn đang xem {course.label}. Đổi lớp ở bộ chọn lớp phía trên.</p>
      )}
      <p className={s.hint}>
        Buổi tiếp theo: {fmtLongDate(sessionDate(CURRENT_SESSION + 1))} · {hours.start}
      </p>
    </Page>
  );
}

/** Sinh viên chưa vào lớp nào (SV D): ô nhập mã tham gia thay cho khuyến nghị. */
function JoinPrompt() {
  const router = useRouter();
  const [code, setCode] = useState("");
  return (
    <Section title="Vào lớp của bạn" description="Giảng viên gửi cho bạn một mã tham gia gồm 7 ký tự.">
      <form
        className={s.join}
        onSubmit={(e) => {
          e.preventDefault();
          if (code.trim()) router.push(`/join/${code.trim().toUpperCase()}`);
        }}
      >
        <Field label="Mã tham gia" helper="Ví dụ: AN7K2MQ. Mã không phân biệt chữ hoa, chữ thường.">
          {(id, describedBy) => (
            <Input
              id={id}
              aria-describedby={describedBy}
              value={code}
              autoComplete="off"
              spellCheck={false}
              maxLength={12}
              placeholder="Nhập mã tham gia"
              onChange={(e) => setCode(e.target.value)}
            />
          )}
        </Field>
        <Button type="submit" variant="primary" disabled={!code.trim()}>
          Xem lớp
        </Button>
      </form>
    </Section>
  );
}
