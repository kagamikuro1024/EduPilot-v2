"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Check, FileSearch, Minus, Plus } from "lucide-react";
import {
  Button,
  ButtonLink,
  EmptyState,
  Field,
  IconButton,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  Skeleton,
  Split,
  StatusText,
  Textarea,
} from "@/shared/ui";
import { fmtScore, fmtShortDate, fmtTime, studentById, studentsOf } from "@/mock/core";
import {
  BT03,
  BT03_CRITERIA,
  BT03_ESSAY,
  BT03_EVIDENCE,
  BT03_MAX_PER_CRITERION,
  BT03_SECOND_PASS_CRITERION_2,
  BT03_SEED,
  BT03_SUBMISSION_ID,
  BT03_SUBMITTED_AT,
  bt03Submissions,
  submissionDetail,
} from "@/mock/assess";
import { LATE_PENALTY_BT03, bt03Total } from "@/mock/grades";
import { KEYS, type Bt03State } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import s from "./SubmissionReview.module.css";

const STEP = 0.25;

export function SubmissionReview({ submissionId }: { submissionId: string }) {
  const router = useRouter();
  const { course } = useSession();
  const [bt03, setBt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [approvedIds, setApprovedIds] = useDemoSlice<string[]>("grading.approved", []);
  const [active, setActive] = useState<string | null>(null);

  const sub = useMemo(() => bt03Submissions(studentsOf(course.id)).find((x) => x.id === submissionId), [course.id, submissionId]);
  const isB = submissionId === BT03_SUBMISSION_ID;
  const other = sub && !isB ? submissionDetail(sub) : null;
  const [otherScores, setOtherScores] = useState<[number, number, number, number]>(other?.scores ?? [0, 0, 0, 0]);
  const [otherComments, setOtherComments] = useState<[string, string, string, string]>(other?.comments ?? ["", "", "", ""]);

  if (!sub) {
    return (
      <Page>
        <PageHeader title="Duyệt bài" back={{ href: "/grading", label: "Hàng chờ chấm" }} />
        <EmptyState title="Không tìm thấy bài nộp này" icon={<FileSearch aria-hidden />} action={<ButtonLink href="/grading">Về hàng chờ chấm</ButtonLink>}>
          Bài nộp có thể đã bị gỡ, hoặc đường dẫn không đúng. Mở lại từ hàng chờ chấm để chọn bài cần duyệt.
        </EmptyState>
      </Page>
    );
  }
  const student = studentById(sub.studentId);
  const subId = sub.id;
  const scores = isB ? bt03.scores : otherScores;
  const comments = isB ? bt03.comments : otherComments;
  const paras = isB ? BT03_ESSAY : (other?.paras ?? []);
  const evidence = isB ? BT03_EVIDENCE : (other?.evidence ?? []);
  const title = isB ? "Chiến dịch tấn công chuỗi cung ứng SolarWinds Orion (2020)" : (other?.title ?? "");
  const approved = isB ? bt03.status !== "draft" : sub.approved || approvedIds.includes(sub.id);
  const published = isB && bt03.status === "published";

  const raw = scores.reduce((a, b) => a + b, 0);
  const late = isB ? bt03Total(bt03).late : sub.lateDays * LATE_PENALTY_BT03;
  const total = isB ? bt03Total({ scores: bt03.scores }).total : Math.max(0, Math.round((raw - late) * 100) / 100);

  function setScore(i: number, next: number) {
    const v = Math.max(0, Math.min(BT03_MAX_PER_CRITERION, Math.round(next * 4) / 4));
    if (isB) {
      setBt03((prev) => {
        const s4 = [...prev.scores] as Bt03State["scores"];
        s4[i] = v;
        return { ...prev, scores: s4 };
      });
    } else {
      setOtherScores((prev) => {
        const s4 = [...prev] as [number, number, number, number];
        s4[i] = v;
        return s4;
      });
    }
  }

  function setComment(i: number, text: string) {
    if (isB) {
      setBt03((prev) => {
        const c4 = [...prev.comments] as Bt03State["comments"];
        c4[i] = text;
        return { ...prev, comments: c4 };
      });
    } else {
      setOtherComments((prev) => {
        const c4 = [...prev] as [string, string, string, string];
        c4[i] = text;
        return c4;
      });
    }
  }

  function approve() {
    if (isB) setBt03((prev) => ({ ...prev, status: prev.status === "published" ? "published" : "approved" }));
    else setApprovedIds((prev) => [...new Set([...prev, subId])]);
    router.push("/grading");
  }

  function focusPara(id: string) {
    setActive(id);
    document.getElementById(`para-${id}`)?.scrollIntoView({ block: "center" });
  }

  const submittedAt = isB ? BT03_SUBMITTED_AT : sub.submittedAt;

  return (
    <Page width="full">
      <PageHeader
        title={`Bài tập 03 · ${student?.name}`}
        description={BT03.title}
        back={{ href: "/grading", label: "Hàng chờ chấm" }}
        meta={
          <>
            <span>{student?.code}</span>
            <span>
              Nộp {fmtShortDate(submittedAt)} {fmtTime(submittedAt)}
              {sub.lateDays > 0 ? ` · muộn ${sub.lateDays} ngày` : ""}
            </span>
            <span>{sub.source}</span>
            {published ? <StatusText tone="green">Đã công bố</StatusText> : approved ? <StatusText tone="blue">Đã duyệt</StatusText> : <StatusText tone="amber">Chưa duyệt</StatusText>}
          </>
        }
      />

      <PageState
        loading={<Skeleton lines={14} />}
        empty={
          <EmptyState title="Không tìm thấy bài nộp này" action={<ButtonLink href="/grading">Về hàng chờ chấm</ButtonLink>}>
            Bài nộp có thể đã bị gỡ. Mở lại từ hàng chờ chấm để chọn bài cần duyệt.
          </EmptyState>
        }
        error={{ problem: "Không tải được bài nộp.", recovery: "Điểm và nhận xét bạn đã sửa vẫn được giữ. Thử lại sau ít phút." }}
      >
        <Split
          ratio="half"
          main={
            <div className={s.doc}>
              <p className={s.docTitle}>{title}</p>
              <div className={s.docMeta}>
                <span>{paras.reduce((a, p) => a + p.text.split(" ").length, 0)} từ</span>
                <span>{isB ? "2 trang" : "1 trang"}</span>
                <span>Bài làm của sinh viên, không chỉnh sửa</span>
              </div>
              {paras.map((p, i) => (
                <div key={p.id}>
                  {i > 0 && paras[i - 1].page !== p.page && <p className={s.pageMark}>Trang {p.page}</p>}
                  <p id={`para-${p.id}`} className={s.para} data-active={active === p.id}>
                    {p.text}
                  </p>
                </div>
              ))}
            </div>
          }
          aside={
            <div className={s.panel}>
              <div className={s.total}>
                <span className={s.totalLabel}>
                  Tổng điểm bài tập
                  <span className={s.totalCalc}>
                    {fmtScore(raw, 2)} điểm rubric {late > 0 ? `− ${fmtScore(late)} nộp muộn` : ""}
                  </span>
                </span>
                <span className={s.totalValue}>{fmtScore(total)}</span>
              </div>

              <InlineNotice tone="info" compact>
                Điểm và nhận xét dưới đây là bản nháp của AI. Giảng viên sửa rồi bấm `Duyệt bài`; sinh viên chỉ thấy sau khi điểm được công bố.
              </InlineNotice>

              {BT03_CRITERIA.map((name, i) => (
                <section key={name} className={s.crit}>
                  <div className={s.critHead}>
                    <span className={s.critName}>{name}</span>
                    <span className={s.stepper} role="group" aria-label={`Điểm tiêu chí ${name}`}>
                      <IconButton size="sm" label={`Giảm điểm ${name}`} onClick={() => setScore(i, scores[i] - STEP)}>
                        <Minus aria-hidden />
                      </IconButton>
                      <span className={s.stepValue}>
                        {fmtScore(scores[i], 2)}
                        <span className={s.stepMax}> / {fmtScore(BT03_MAX_PER_CRITERION, 1)}</span>
                      </span>
                      <IconButton size="sm" label={`Tăng điểm ${name}`} onClick={() => setScore(i, scores[i] + STEP)}>
                        <Plus aria-hidden />
                      </IconButton>
                    </span>
                  </div>

                  {isB && i === 1 && (
                    <InlineNotice tone="warning" compact title={`Hai lượt chấm lệch ${fmtScore(BT03_SECOND_PASS_CRITERION_2 - BT03_SEED.scores[1])} điểm`}>
                      Lượt 1 chấm {fmtScore(BT03_SEED.scores[1], 2)}, lượt 2 chấm {fmtScore(BT03_SECOND_PASS_CRITERION_2, 2)}. Đọc đoạn trích rồi quyết định — việc duyệt bài không bị chặn.
                    </InlineNotice>
                  )}

                  {evidence[i] && (
                    <button type="button" className={s.evidence} onClick={() => focusPara(evidence[i].para)}>
                      <span className={s.evidenceTag}>Đoạn trích từ bài làm · bấm để xem trong bài</span>“{evidence[i].quote}”
                    </button>
                  )}

                  <Field label="Nhận xét cho sinh viên" helper="Nhận xét do AI soạn nháp; bạn sửa trước khi công bố.">
                    {(id) => <Textarea id={id} rows={2} value={comments[i]} onChange={(e) => setComment(i, e.target.value)} />}
                  </Field>
                </section>
              ))}

              {late > 0 && (
                <p className={s.aiTag}>
                  Trừ {fmtScore(late)} điểm do nộp muộn {sub.lateDays} ngày so với hạn {fmtShortDate(BT03.due)} {fmtTime(BT03.due)} (quy định trừ {fmtScore(BT03.latePerDay)} mỗi ngày, tối đa{" "}
                  {BT03.lateMaxDays} ngày).
                </p>
              )}

              <div className={s.foot}>
                <p className={s.footText}>
                  {approved
                    ? published
                      ? "Bài đã công bố. Sinh viên thấy điểm, nhận xét và đoạn trích từ bài của mình."
                      : "Bài đã duyệt. Việc công bố điểm cho sinh viên làm ở hàng chờ chấm."
                    : `Duyệt bài sẽ chốt ${fmtScore(total)} điểm làm điểm của sinh viên; điểm chỉ hiện với sinh viên sau khi công bố.`}
                </p>
                <Button variant="primary" icon={<Check aria-hidden />} onClick={approve} disabled={approved}>
                  {approved ? "Đã duyệt" : "Duyệt bài"}
                </Button>
              </div>
            </div>
          }
        />
      </PageState>
    </Page>
  );
}
