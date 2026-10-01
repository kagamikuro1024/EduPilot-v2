"use client";

import { useState } from "react";
import { BT03_SEED } from "@/mock/assess";
import { COURSE_1, NOW, STUDENT_B, fmtLongDate, fmtScore, fmtShortDate, fmtTime, studentById } from "@/mock/core";
import { SCHEME_1, calcFinal, qtOf } from "@/mock/grades";
import { ATTENDANCE_SEED, CURRENT_SESSION, KEYS, SCHEMES_SEED, type AttendanceState, type Bt03State, type SchemesState } from "@/mock/state";
import { ASSIGNMENTS, B_ABSENT_DATES, MIDTERM, sessionDate, until } from "@/mock/student";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  ButtonLink,
  DefinitionList,
  EmptyState,
  Field,
  InlineNotice,
  Input,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
  TrendChart,
} from "@/shared/ui";
import s from "./Me.module.css";

const STUDY_WEEKS = [
  { x: "T5", y: 120 },
  { x: "T6", y: 165 },
  { x: "T7", y: 90 },
  { x: "T8", y: 185 },
  { x: "T9", y: 140 },
  { x: "T10", y: 160 },
];

/** Kết quả của tôi: hiểu mình đang đứng ở đâu và điều gì ảnh hưởng tới điểm (DESIGN §14.9). */
export function MeScreen() {
  const { course, studentId } = useSession();
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const student = studentById(studentId ?? "") ?? STUDENT_B;
  const g = qtOf(student.id, attendance, bt03);
  const todaySession = attendance[COURSE_1]?.[CURRENT_SESSION];
  const absentDates =
    student.id === STUDENT_B.id
      ? [...B_ABSENT_DATES, ...(todaySession?.finalized && todaySession.marks[student.id] === "absent" ? [fmtShortDate(sessionDate(CURRENT_SESSION))] : [])]
      : [];
  const quiz01 = ASSIGNMENTS[3];
  const schemeReady = course.id === COURSE_1 || schemes[course.id]?.status === "confirmed";

  return (
    <Page>
      <PageHeader title="Kết quả của tôi" description={`${course.label} · ${course.schedule}`} meta={`Cập nhật ${fmtLongDate(NOW)} ${fmtTime(NOW)}`} />
      <PageState
        loading={
          <>
            <Skeleton lines={2} />
            <Skeleton lines={5} />
          </>
        }
        empty={
          <EmptyState title="Chưa có dữ liệu điểm cho lớp này" action={<ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}>
            Khi có bài tập được công bố hoặc buổi điểm danh đầu tiên, kết quả của bạn sẽ hiện ở đây.
          </EmptyState>
        }
      >
        {!schemeReady ? (
          <InlineNotice title="Lớp này chưa có công thức điểm chính thức">
            Giảng viên chưa xác nhận công thức tính điểm cho {course.label}, nên chưa thể giải trình điểm quá trình. Bài tập và buổi
            học của lớp vẫn xem được ở Lịch và Thư viện.
          </InlineNotice>
        ) : (
          <>
            <Section title="Điểm quá trình">
              <p className={s.headline}>
                Điểm quá trình hiện tại <strong>{fmtScore(g.qt)}</strong> (tạm tính)
              </p>
              <p className={s.sub}>
                Tính trên {[g.bt01, g.bt02, g.bt03].filter((x) => x !== null).length} bài tập đã công bố, cộng điểm phát biểu và trừ
                vắng theo quy chế môn học ({SCHEME_1.qtWeight}% quá trình · {100 - SCHEME_1.qtWeight}% cuối kỳ).
              </p>

              <div className={s.calc}>
                <p className={s.calcLine}>
                  <span>Trung bình bài tập</span>
                  <span>
                    {[["Bài tập 01", g.bt01], ["Bài tập 02", g.bt02], ["Bài tập 03", g.bt03]]
                      .filter(([, v]) => v !== null)
                      .map(([k, v]) => `${k} ${fmtScore(v as number)}`)
                      .join(" · ")}
                  </span>
                  <strong>{fmtScore(g.avg, 2)}</strong>
                </p>
                <p className={s.calcLine}>
                  <span>Điểm cộng phát biểu</span>
                  <span>
                    {g.speaks} lần × {fmtScore(SCHEME_1.speakBonus, 2)} (trần {fmtScore(SCHEME_1.speakCap, 2)})
                  </span>
                  <strong>+{fmtScore(g.bonus, 2)}</strong>
                </p>
                <p className={s.calcLine}>
                  <span>Trừ vắng không phép</span>
                  <span>
                    {g.absences} buổi vắng, trừ từ buổi vắng thứ {SCHEME_1.absenceFrom}
                  </span>
                  <strong>{g.penalty > 0 ? `−${fmtScore(g.penalty, 2)}` : "0,00"}</strong>
                </p>
                <p className={[s.calcLine, s.calcTotal].join(" ")}>
                  <span>Điểm quá trình</span>
                  <span>
                    {fmtScore(g.avg, 2)} + {fmtScore(g.bonus, 2)} − {fmtScore(g.penalty, 2)} = {fmtScore((g.avg ?? 0) + g.bonus - g.penalty, 2)} → làm tròn 0,1
                  </span>
                  <strong>{fmtScore(g.qt)}</strong>
                </p>
              </div>

              <WhatIf qt={g.qt ?? 0} />
              <p className={s.official}>Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.</p>
            </Section>

            <Section title="Chuyên cần và phát biểu">
              <DefinitionList
                items={[
                  { term: "Buổi đã ghi", value: `${g.recorded} buổi` },
                  { term: "Vắng", value: g.absences > 0 ? `${g.absences} buổi${absentDates.length ? ` (${absentDates.join(", ")})` : ""}` : "Không vắng buổi nào" },
                  { term: "Đi muộn", value: g.late > 0 ? `${g.late} buổi` : "Không" },
                  { term: "Phát biểu", value: `${g.speaks} lần · +${fmtScore(g.bonus, 2)}` },
                ]}
              />
              <p className={s.sub}>
                Vắng thêm 1 buổi không phép sẽ bị trừ {fmtScore(SCHEME_1.absencePenalty, 2)} điểm quá trình.
              </p>
            </Section>
          </>
        )}

        <Section title="Bài sắp tới">
          <ActionList label="Bài sắp tới">
            <ActionRow
              tone="amber"
              href="/practice/at-quiz01"
              title={`${quiz01.code} — ${quiz01.title}`}
              context={`Đóng ${fmtShortDate(quiz01.due)} · còn ${until(quiz01.due)}`}
              meta={`${quiz01.minutes} phút · tính điểm`}
            />
            <ActionRow
              href="/assignments/bt03"
              title="Bài tập 03 — Phân tích một vụ tấn công thực tế"
              context={bt03.status === "published" ? "Đã có điểm và nhận xét" : "Đã nộp · đang chấm"}
              meta={bt03.status === "published" ? <StatusText tone="green">Đã công bố</StatusText> : <StatusText tone="neutral">Đang chấm</StatusText>}
            />
            <ActionRow
              href="/calendar"
              title={MIDTERM.title}
              context={`${fmtLongDate(MIDTERM.at)} · ${MIDTERM.room}`}
              meta={`Tuần ${MIDTERM.week}`}
            />
          </ActionList>
        </Section>

        <Section title="Thời gian học mỗi tuần" description="Chỉ để bạn tự theo dõi nhịp học, không tính vào điểm.">
          <div className={s.trend}>
            <TrendChart points={STUDY_WEEKS} label="Thời gian học theo tuần (phút)" height={96} format={(v) => `${v} phút`} />
          </div>
        </Section>
      </PageState>
    </Page>
  );
}

/** "Nếu cuối kỳ được [ 8,0 ], điểm học phần sẽ là 8,3" — cập nhật ngay khi gõ. */
function WhatIf({ qt }: { qt: number }) {
  const [raw, setRaw] = useState("8,0");
  const value = Number(raw.replace(",", "."));
  const invalid = raw.trim() === "" || Number.isNaN(value) || value < 0 || value > 10;
  return (
    <div className={s.whatIf}>
      <Field
        label="Nếu điểm cuối kỳ của bạn là"
        error={invalid ? "Nhập một số từ 0 đến 10, ví dụ 8,0." : undefined}
        helper={invalid ? undefined : "Thử một con số để xem điểm học phần dự kiến."}
      >
        {(id, describedBy) => (
          <Input
            id={id}
            aria-describedby={describedBy}
            className={s.whatIfInput}
            inputMode="decimal"
            value={raw}
            invalid={invalid}
            onChange={(e) => setRaw(e.target.value)}
          />
        )}
      </Field>
      {!invalid && (
        <p className={s.whatIfOut}>
          điểm học phần sẽ là <strong>{fmtScore(calcFinal(qt, value))}</strong>
          <span className={s.whatIfNote}>
            {SCHEME_1.qtWeight}% × {fmtScore(qt)} + {100 - SCHEME_1.qtWeight}% × {fmtScore(value)}
          </span>
        </p>
      )}
    </div>
  );
}
