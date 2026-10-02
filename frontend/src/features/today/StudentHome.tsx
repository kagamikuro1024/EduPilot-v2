"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { JOIN_SENT_KEY } from "@/features/join/JoinScreen";
import { agoLabel } from "@/mock/derive";
import { COURSES, COURSE_1, COURSE_2, NOW, STUDENT_B, fmtLongDate, fmtShortDate, fmtTime, studentById } from "@/mock/core";
import { QUIZ_SEED, quizKey, type QuizState } from "@/mock/practice";
import { ASSIGNMENTS, CONTINUE_LEARNING, SESSION_HOURS, recommendationFor, sessionDate, until } from "@/mock/student";
import { CURRENT_SESSION, KEYS, MEMBERS_SEED, type MembersState } from "@/mock/state";
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

/** Buổi gần nhất của lớp không học vào hôm nay (lớp 2 học Thứ Ba: hôm nay Thứ Năm 29/10 → Thứ Ba 03/11). */
const NEXT_MEETING: Record<string, Date> = { [COURSE_2]: new Date("2026-11-03T07:00:00+07:00") };

/** "Hôm nay" của sinh viên: một việc nên làm, dòng thời gian hôm nay, chỗ học dở (DESIGN §14.1). */
export function StudentHome() {
  const { user, course, courses, hasCourse, studentId } = useSession();
  const [quiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
  const student = studentById(studentId ?? "") ?? STUDENT_B;
  const firstName = user.name.split(" ").slice(-1)[0];
  const hours = SESSION_HOURS[course.id] ?? SESSION_HOURS[COURSE_1];
  const quiz01 = ASSIGNMENTS[3];
  const rec = recommendationFor(student, quiz.status === "submitted");
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [sentAt] = useDemoSlice<Record<string, number>>(JOIN_SENT_KEY, {});
  // Dữ liệu học (buổi 10, Quiz 01, "học dở") là của lớp 1 — lớp học Thứ Năm, hôm nay (SRS 4.8). Lớp khác: hôm nay không có buổi.
  const meetsToday = course.id === COURSE_1;
  const pendingCourse = COURSES.find((c) => (members.pending[c.id] ?? []).includes(studentId ?? ""));

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
                {meetsToday ? (
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
                ) : (
                  <ActionRow
                    tone="red"
                    href="/library"
                    redThread
                    title="Xem tài liệu và quy chế của lớp"
                    context={`Bạn mới vào lớp ${course.code}: đọc quy chế môn học và bài giảng để biết lớp tính điểm và học thế nào.`}
                    meta="Khoảng 5 phút"
                    action={
                      <ButtonLink href="/library" variant="primary">
                        Mở Thư viện
                      </ButtonLink>
                    }
                  />
                )}
              </ActionList>
            </Section>

            <Section title="Hôm nay" description={`${course.room} · ${course.schedule}`}>
              <ActionList label="Lịch hôm nay">
                {meetsToday ? (
                  <>
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
                  </>
                ) : (
                  <ActionRow tone="neutral" title="Hôm nay lớp không có buổi học" context={`Lớp học ${course.schedule} · ${course.room}`} href="/calendar" />
                )}
              </ActionList>
            </Section>

            {meetsToday && (
              <Section title="Học dở" description="Mở lại chỗ bạn dừng hôm qua.">
                <ActionList label="Học dở">
                  {CONTINUE_LEARNING.map((c) => (
                    <ActionRow key={c.href} title={c.title} context={c.context} meta={agoLabel(c.at)} href={c.href} />
                  ))}
                </ActionList>
              </Section>
            )}
          </>
        ) : pendingCourse ? (
          <Section title="Vào lớp của bạn">
            <StatusText tone="amber">
              Yêu cầu vào lớp {pendingCourse.code} đang chờ giảng viên duyệt · gửi lúc{" "}
              {fmtTime(new Date(sentAt[pendingCourse.id] ?? NOW.getTime()))}
            </StatusText>
            <p className={s.hint}>Khi giảng viên duyệt, lớp sẽ hiện trong bộ chọn lớp và bạn nhận được thông báo.</p>
          </Section>
        ) : (
          <JoinPrompt />
        )}
      </PageState>
      {hasCourse && courses.length > 1 && (
        <p className={s.hint}>Bạn đang xem {course.label}. Đổi lớp ở bộ chọn lớp phía trên.</p>
      )}
      {hasCourse && (
        <p className={s.hint}>
          Buổi tiếp theo: {fmtLongDate(meetsToday ? sessionDate(CURRENT_SESSION + 1) : NEXT_MEETING[course.id] ?? sessionDate(CURRENT_SESSION + 1))} · {hours.start}
        </p>
      )}
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
