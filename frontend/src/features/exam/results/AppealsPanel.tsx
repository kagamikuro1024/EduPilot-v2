"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient, useCursorList, useIdempotentMutation } from "@/shared/data";
import { Button, EmptyState, Field, Input, InlineNotice, SegmentedControl, Select, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { fmtClock } from "../examApi";
import { resPath, vnum, type Appeal } from "./resultsApi";
import s from "./Results.module.css";

const STATUS: Record<Appeal["status"], string> = { OPEN: "Chờ trả lời", UPHELD: "Giữ nguyên điểm", ADJUSTED: "Đã sửa điểm" };

/** Tab "Xem lại điểm": yêu cầu của sinh viên, cũ nhất trước ở trên. Chỉ Giảng viên trả lời; TA chỉ đọc. Không có AI trong luồng này. */
export function AppealsPanel({ course, exam, teacher }: { course: string; exam: string; teacher: boolean }) {
  const qc = useQueryClient();
  const [status, setStatus] = useState<"" | "OPEN" | "UPHELD" | "ADJUSTED">("OPEN");
  const list = useCursorList<Appeal>(["exam-appeals", course, exam, status], `${resPath(course, exam)}/appeals`, { query: { status: status || undefined }, limit: 50 });
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ["exam-appeals", course, exam] });
    void qc.invalidateQueries({ queryKey: ["exam-results", course, exam] });
    void qc.invalidateQueries({ queryKey: ["today"] });
  };
  return (
    <>
      <div className={s.bar}>
        <SegmentedControl label="Lọc yêu cầu" value={status} onChange={setStatus} options={[{ value: "OPEN", label: "Chờ trả lời" }, { value: "", label: "Tất cả" }]} />
      </div>
      {list.isPending ? (
        <Skeleton lines={4} />
      ) : list.isError ? (
        <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} />
      ) : list.items.length === 0 ? (
        <EmptyState title="Chưa có yêu cầu nào">Khi sinh viên gửi yêu cầu xem lại điểm, yêu cầu hiện ở đây.</EmptyState>
      ) : (
        <div data-part="appeals">
          {list.items.map((a) => <AppealRow key={a.id} a={a} course={course} exam={exam} teacher={teacher} onDone={refresh} />)}
          {list.hasNextPage && <div><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
        </div>
      )}
    </>
  );
}

function AppealRow({ a, course, exam, teacher, onDone }: { a: Appeal; course: string; exam: string; teacher: boolean; onDone: () => void }) {
  const [decision, setDecision] = useState<"UPHELD" | "ADJUSTED">("UPHELD");
  const [score, setScore] = useState("");
  const [response, setResponse] = useState("");
  const answer = useIdempotentMutation<void, Appeal>((_v, key) =>
    apiClient.post<Appeal>(`${resPath(course, exam)}/appeals/${a.id}/answer`, { decision, response: response.trim(), score: decision === "ADJUSTED" ? score.replace(",", ".").trim() : null, version: a.version }, { idempotencyKey: key }),
  );
  return (
    <article className={s.appeal} data-part="appeal">
      <p className={s.subHead}>{a.student?.full_name} · {a.student?.student_code || "chưa có MSSV"} <StatusText tone={a.status === "OPEN" ? "amber" : "green"}>{STATUS[a.status]}</StatusText></p>
      <p className={s.muted}>Gửi lúc {fmtClock(a.created_at)}</p>
      <p>{a.reason}</p>
      {a.status !== "OPEN" && (
        <InlineNotice compact>
          {a.response}
          {a.status === "ADJUSTED" ? ` (điểm ${vnum(a.score_before)} → ${vnum(a.score_after)})` : ""}
        </InlineNotice>
      )}
      {a.status === "OPEN" && teacher && (
        <div className={s.form}>
          <Field label="Quyết định">{(id, by) => <Select id={id} aria-describedby={by} value={decision} onChange={(ev) => setDecision(ev.target.value as "UPHELD" | "ADJUSTED")}><option value="UPHELD">Giữ nguyên điểm</option><option value="ADJUSTED">Sửa điểm</option></Select>}</Field>
          {decision === "ADJUSTED" && <Field label="Điểm mới">{(id, by) => <Input id={id} aria-describedby={by} inputMode="decimal" value={score} onChange={(ev) => setScore(ev.target.value)} />}</Field>}
          <Field label="Phản hồi cho sinh viên (bắt buộc, tối đa 1.000 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={3} maxLength={1000} value={response} onChange={(ev) => setResponse(ev.target.value)} />}</Field>
          {answer.error && <ApiErrorNotice error={answer.error} />}
          <div>
            <Button variant="primary" loading={answer.pending} disabled={response.trim() === "" || (decision === "ADJUSTED" && score.trim() === "")} onClick={() => void answer.mutate().then(onDone, () => undefined)}>Gửi phản hồi</Button>
          </div>
        </div>
      )}
    </article>
  );
}
