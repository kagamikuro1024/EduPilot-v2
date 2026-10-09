"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient, useCursorList } from "@/shared/data";
import { Button, DefinitionList, Drawer, Field, Input, InlineNotice, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { fmtClock, type ExamDetail } from "../examApi";
import { OverridePanel } from "./OverridePanel";
import { ResultItems } from "./ResultItems";
import { REASON_VI, resPath, STATUS_VI, VERDICT_VI, vnum, type Detail } from "./resultsApi";
import s from "./Results.module.css";

type EventRow = { id: string; type: string; occurred_at: string; meta: Record<string, unknown> | null };
const EVENT_VI: Record<string, string> = { TAB_HIDDEN: "Rời tab", TAB_VISIBLE: "Quay lại tab", PASTE: "Dán nội dung", OFFLINE: "Mất mạng", ONLINE: "Có mạng lại", TAB_TAKEOVER: "Mở bài ở tab khác", CHAT_BLOCKED: "Chat bị chặn" };

/** Drawer chi tiết một lượt: điểm (kèm sửa tay có lý do), từng câu, mọi lần nộp kèm verdict TỪNG test (kể cả test ẩn), phúc khảo; Giảng viên thấy thêm tín hiệu liêm chính. */
export function ResultDrawer({ course, exam, attempt, teacher, e, onClose, onChanged }: { course: string; exam: string; attempt: string; teacher: boolean; e: ExamDetail; onClose: () => void; onChanged: () => void }) {
  const key = ["exam-result", course, exam, attempt];
  const d = useQuery({ queryKey: key, queryFn: async ({ signal }) => (await apiClient.get<Detail>(`${resPath(course, exam)}/results/${attempt}`, { signal })).data });
  return (
    <Drawer open onClose={onClose} title={d.data ? `${d.data.student.full_name} · ${d.data.student.student_code || "chưa có MSSV"}` : "Chi tiết bài làm"} description={d.data ? STATUS_VI[d.data.status] : undefined}>
      {d.isPending ? (
        <Skeleton lines={8} />
      ) : d.isError ? (
        <ApiErrorNotice error={d.error} onRetry={() => void d.refetch()} />
      ) : (
        <div className={s.drawer} data-part="result-drawer">
          <Score d={d.data} e={e} teacher={teacher} course={course} exam={exam} onSaved={() => { void d.refetch(); onChanged(); }} />
          {d.data.status === "GRADED" || d.data.items.length > 0 ? <ResultItems items={d.data.items} staff /> : null}
          {teacher && (e.status === "CLOSED" || e.status === "PUBLISHED") && <OverridePanel course={course} exam={exam} items={d.data.items} onDone={() => { void d.refetch(); onChanged(); }} />}
          <Submissions d={d.data} />
          {teacher && d.data.integrity && <Integrity d={d.data} course={course} exam={exam} attempt={attempt} />}
          {d.data.appeal && (
            <section className={s.sub} aria-label="Yêu cầu xem lại">
              <p className={s.subHead}>Yêu cầu xem lại · {d.data.appeal.status === "OPEN" ? "chờ trả lời" : d.data.appeal.status === "UPHELD" ? "giữ nguyên điểm" : "đã nâng / sửa điểm"}</p>
              <p>{d.data.appeal.reason}</p>
              {d.data.appeal.response && <p className={s.muted}>Phản hồi: {d.data.appeal.response}</p>}
            </section>
          )}
        </div>
      )}
    </Drawer>
  );
}

function Score({ d, e, teacher, course, exam, onSaved }: { d: Detail; e: ExamDetail; teacher: boolean; course: string; exam: string; onSaved: () => void }) {
  const [score, setScore] = useState(d.adjust?.score ?? d.score ?? "");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const can = teacher && d.status === "GRADED" && (e.status === "CLOSED" || e.status === "PUBLISHED");
  async function save(clear: boolean) {
    setBusy(true);
    setErr(null);
    try {
      await apiClient.put(`${resPath(course, exam)}/results/${d.attempt_id}/score`, { score: clear ? null : score.replace(",", ".").trim(), reason: reason.trim(), version: d.version });
      setReason("");
      onSaved();
    } catch (x) {
      setErr(x);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className={s.sub} aria-label="Điểm">
      <DefinitionList
        items={[
          { term: "Điểm chính thức", value: <strong>{vnum(d.score)} / {vnum(e.max_score)}</strong> },
          { term: "Điểm máy chấm", value: vnum(d.auto_score) },
          { term: "Nộp lúc", value: d.submitted_at ? `${fmtClock(d.submitted_at)}${d.submit_reason ? ` · ${REASON_VI[d.submit_reason] ?? d.submit_reason}` : ""}` : "—" },
        ]}
      />
      {d.adjust && <InlineNotice compact>Đã sửa tay thành {vnum(d.adjust.score)}: {d.adjust.reason}</InlineNotice>}
      {can && (
        <div className={s.form}>
          <Field label="Điểm mới (đúng bước làm tròn)">{(id, by) => <Input id={id} aria-describedby={by} inputMode="decimal" value={score.replace(".", ",")} onChange={(ev) => setScore(ev.target.value)} />}</Field>
          <Field label="Lý do (bắt buộc, tối đa 500 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={2} maxLength={500} value={reason} onChange={(ev) => setReason(ev.target.value)} />}</Field>
          {err != null && <ApiErrorNotice error={err} />}
          <div className={s.bar}>
            <Button variant="primary" onClick={() => void save(false)} loading={busy} disabled={reason.trim() === "" || score.trim() === ""}>Lưu điểm</Button>
            {d.adjust && <Button onClick={() => void save(true)} loading={busy} disabled={reason.trim() === ""}>Về điểm máy chấm</Button>}
          </div>
        </div>
      )}
    </section>
  );
}

function Submissions({ d }: { d: Detail }) {
  const byItem = d.items.filter((i) => i.type === "CODE");
  if (d.submissions.length === 0) return null;
  return (
    <section className={s.sub} aria-label="Các lần nộp">
      <p className={s.subHead}>Các lần nộp</p>
      {byItem.map((it) => (
        <div key={it.item_id} className={s.sub}>
          <p className={s.label}>Câu {it.position}</p>
          {d.submissions.filter((x) => x.item_id === it.item_id).map((x, i) => (
            <details key={x.id}>
              <summary>
                Lần {i + 1} · {fmtClock(x.created_at)} · {x.verdict ? VERDICT_VI[x.verdict] ?? x.verdict : x.status}
              </summary>
              {x.tests.length > 0 && (
                <p className={s.muted}>
                  {x.tests.map((t) => `${t.is_sample ? "mẫu" : "ẩn"} ${t.position}: ${VERDICT_VI[t.verdict] ?? t.verdict}`).join(" · ")}
                </p>
              )}
              {x.compile_ok === false && x.compile_log && <pre className={s.pre}>{x.compile_log}</pre>}
              <pre className={s.pre}>{x.source}</pre>
            </details>
          ))}
        </div>
      ))}
    </section>
  );
}

function Integrity({ d, course, exam, attempt }: { d: Detail; course: string; exam: string; attempt: string }) {
  const g = d.integrity!;
  const list = useCursorList<EventRow>(["exam-events", course, exam, attempt], `${resPath(course, exam)}/events`, { query: { attempt }, limit: 20 });
  return (
    <section className={s.sub} aria-label="Tín hiệu liêm chính" data-part="integrity">
      <p className={s.subHead}>Tín hiệu <StatusText tone="neutral">chỉ giảng viên thấy</StatusText></p>
      <p className={s.muted}>Chỉ là tín hiệu để tham khảo, không phải kết luận.</p>
      <DefinitionList
        items={[
          { term: "Rời tab", value: `${g.tab_hidden_count} lần · ${Math.round(g.tab_hidden_ms / 1000)} giây` },
          { term: "Dán nội dung", value: `${g.paste_count} lần · ${g.paste_chars} ký tự` },
          { term: "Mất mạng", value: `${g.offline_count} lần` },
          { term: "Mở ở tab khác", value: `${g.takeover_count} lần` },
        ]}
      />
      {list.items.length > 0 && (
        <ul className={s.opts}>
          {list.items.map((x) => <li key={x.id} className={s.opt}><span>{EVENT_VI[x.type] ?? x.type}</span><span className={s.tag}>{fmtClock(x.occurred_at)}</span></li>)}
        </ul>
      )}
      {list.hasNextPage && <div><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
    </section>
  );
}
