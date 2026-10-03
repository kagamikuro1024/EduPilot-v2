"use client";

import { FileText } from "lucide-react";
import { useState } from "react";
import { BT03_CRITERIA, BT03_EVIDENCE, BT03_MAX_PER_CRITERION, BT03_SEED } from "@/mock/assess";
import { STUDENT_B, fmtLongDate, fmtScore, fmtTime, studentById } from "@/mock/core";
import { agoLabel } from "@/mock/derive";
import { baseBtScores, bt03Total } from "@/mock/grades";
import { KEYS, type Bt03State } from "@/mock/state";
import { QUIZ_SEED, quizKey, type QuizState } from "@/mock/practice";
import { BT03_SUBMISSION, assignmentById, until } from "@/mock/student";
import { useSession } from "@/shared/session/session";
import { useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  Checkbox,
  DefinitionList,
  EmptyState,
  Field,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
  Textarea,
} from "@/shared/ui";
import s from "./Assignment.module.css";

/** Bài tập của sinh viên: đã nộp gì, chấm tới đâu, nhận xét ra sao (INTEGRATION mục 2). */
export function AssignmentScreen({ id }: { id: string }) {
  const { studentId } = useSession();
  const now = useSimNow();
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [quiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
  const quizDone = quiz.status === "submitted";
  const assignment = assignmentById(id);
  const student = studentById(studentId ?? "") ?? STUDENT_B;

  if (!assignment) {
    return (
      <Page>
        <PageHeader title="Bài tập" back={{ href: "/me", label: "Kết quả của tôi" }} />
        <EmptyState title="Không tìm thấy bài tập này" action={<ButtonLink href="/me" variant="primary">Về Kết quả của tôi</ButtonLink>}>
          Bài có thể đã bị gỡ, hoặc thuộc lớp khác lớp bạn đang xem.
        </EmptyState>
      </Page>
    );
  }

  const isBt03 = assignment.id === "bt03";
  const mineBt03 = isBt03 && student.id === STUDENT_B.id;
  const published = bt03.status === "published";
  const total = bt03Total(bt03);
  const base = baseBtScores(student);
  const oldScore = assignment.id === "bt01" ? base.bt01 : assignment.id === "bt02" ? base.bt02 : null;

  return (
    <Page>
      <PageHeader
        title={`${assignment.code} — ${assignment.title}`}
        back={{ href: "/me", label: "Kết quả của tôi" }}
        description={assignment.description}
        meta={
          <>
            <span>Hạn {fmtLongDate(assignment.due)} {fmtTime(assignment.due)}</span>
            {assignment.lateDays ? <span>Cho nộp muộn tối đa {assignment.lateDays} ngày, trừ 0,5 điểm mỗi ngày</span> : null}
          </>
        }
        actions={
          assignment.kind === "quiz" ? (
            <ButtonLink href="/practice/at-quiz01" variant="primary">
              {quizDone ? "Xem kết quả" : "Làm bài"}
            </ButtonLink>
          ) : undefined
        }
      />

      <PageState
        loading={<Skeleton lines={7} />}
        empty={
          <EmptyState title="Bài này chưa mở cho lớp của bạn" action={<ButtonLink href="/calendar" variant="primary">Xem lịch</ButtonLink>}>
            Khi giảng viên mở bài, bạn sẽ thấy đề bài và nút nộp ở đây.
          </EmptyState>
        }
      >
        {assignment.kind === "quiz" ? (
          <Section title="Bài kiểm tra trên lớp">
            <DefinitionList
              items={[
                { term: "Hình thức", value: `Trắc nghiệm · ${assignment.minutes} phút · tính điểm` },
                { term: "Đóng lúc", value: `${fmtTime(assignment.due)} ${fmtLongDate(assignment.due)} · còn ${until(assignment.due)}` },
                { term: "Trạng thái", value: quizDone ? <StatusText tone="green">Bạn đã nộp bài</StatusText> : <StatusText tone="amber">Bạn chưa làm</StatusText> },
              ]}
            />
            <InlineNotice compact>Trong lúc làm bài, Chat riêng chỉ trả lời câu hỏi thủ tục.</InlineNotice>
          </Section>
        ) : (
          <>
            <Section title="Bài nộp của bạn">
              {mineBt03 ? (
                <>
                  <p className={s.file}>
                    <FileText aria-hidden />
                    {BT03_SUBMISSION.file}
                    <span className={s.fileMeta}>{BT03_SUBMISSION.sizeKb} KB</span>
                  </p>
                  <DefinitionList
                    items={[
                      { term: "Nộp lúc", value: `${fmtTime(BT03_SUBMISSION.at)} ${fmtLongDate(BT03_SUBMISSION.at)} · ${agoLabel(BT03_SUBMISSION.at, now)}` },
                      { term: "Hạn nộp", value: `${fmtTime(assignment.due)} ${fmtLongDate(assignment.due)}` },
                      { term: "Ghi nhận", value: <StatusText tone="amber">Nộp muộn {BT03_SUBMISSION.lateDays} ngày · trừ {fmtScore(total.late, 1)} điểm</StatusText> },
                    ]}
                  />
                </>
              ) : (
                <DefinitionList
                  items={[
                    { term: "Trạng thái", value: oldScore !== null ? "Đã nộp đúng hạn" : "Chưa có bài nộp" },
                    { term: "Hạn nộp", value: `${fmtTime(assignment.due)} ${fmtLongDate(assignment.due)}` },
                  ]}
                />
              )}
            </Section>

            <Section title="Kết quả">
              {mineBt03 && !published ? (
                <InlineNotice title="Đang chấm">
                  Giảng viên đang xem lại bài của bạn. Điểm và nhận xét sẽ hiện ở đây ngay khi được công bố — thường trong vài ngày
                  sau hạn nộp.
                </InlineNotice>
              ) : mineBt03 && published ? (
                <>
                  <p className={s.score}>
                    Điểm bài này: <strong>{fmtScore(total.total)}</strong>
                    <span className={s.scoreNote}>
                      {fmtScore(total.raw)} theo rubric − {fmtScore(total.late)} nộp muộn
                    </span>
                  </p>
                  <ol className={s.criteria}>
                    {BT03_CRITERIA.map((c, i) => (
                      <li key={c} className={s.criterion}>
                        <div className={s.criterionHead}>
                          <span className={s.criterionName}>{c}</span>
                          <span className={s.criterionScore}>
                            {fmtScore(bt03.scores[i], 2)} / {fmtScore(BT03_MAX_PER_CRITERION, 1)}
                          </span>
                        </div>
                        <p className={s.comment}>{bt03.comments[i]}</p>
                        <p className={s.excerpt}>“{BT03_EVIDENCE[i].quote}”</p>
                      </li>
                    ))}
                  </ol>
                  <RecheckRequest />
                </>
              ) : oldScore !== null ? (
                <p className={s.score}>
                  Điểm bài này: <strong>{fmtScore(oldScore)}</strong>
                  <span className={s.scoreNote}>Đã công bố · tính vào điểm quá trình</span>
                </p>
              ) : (
                <InlineNotice title="Bài này chưa có điểm">Khi giảng viên công bố, điểm và nhận xét sẽ hiện tại đây.</InlineNotice>
              )}
            </Section>
          </>
        )}
      </PageState>
    </Page>
  );
}

/** Phúc khảo: chọn tiêu chí + nêu lý do, gửi trong 7 ngày kể từ khi công bố. */
function RecheckRequest() {
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [sent, setSent] = useState(false);

  if (sent) {
    return (
      <div className={s.recheck}>
        <StatusText tone="green">Đã gửi yêu cầu xem lại</StatusText>
        <p className={s.recheckNote}>
          Giảng viên sẽ trả lời trong hộp thư hỗ trợ. Bạn vẫn xem được điểm hiện tại trong lúc chờ.
        </p>
      </div>
    );
  }

  if (!open) {
    return (
      <div className={s.recheck}>
        <Button onClick={() => setOpen(true)}>Yêu cầu xem lại</Button>
        <p className={s.recheckNote}>Gửi được trong 7 ngày kể từ khi điểm được công bố.</p>
      </div>
    );
  }

  return (
    <form
      className={s.recheckForm}
      onSubmit={(e) => {
        e.preventDefault();
        setSent(true);
      }}
    >
      <fieldset className={s.fieldset}>
        <legend className={s.legend}>Tiêu chí bạn muốn xem lại</legend>
        {BT03_CRITERIA.map((c) => (
          <Checkbox
            key={c}
            label={c}
            checked={picked.includes(c)}
            onChange={() => setPicked((p) => (p.includes(c) ? p.filter((x) => x !== c) : [...p, c]))}
          />
        ))}
      </fieldset>
      <Field label="Lý do" helper="Nêu cụ thể phần nào trong bài bạn cho là đã đáp ứng tiêu chí.">
        {(id, describedBy) => (
          <Textarea id={id} aria-describedby={describedBy} rows={4} value={reason} onChange={(e) => setReason(e.target.value)} />
        )}
      </Field>
      <div className={s.recheckActions}>
        <Button type="submit" variant="primary" disabled={picked.length === 0 || reason.trim().length < 10}>
          Gửi yêu cầu
        </Button>
        <Button variant="ghost" onClick={() => setOpen(false)}>
          Để sau
        </Button>
      </div>
    </form>
  );
}

