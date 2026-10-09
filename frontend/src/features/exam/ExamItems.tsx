"use client";

import { Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, useCursorList } from "@/shared/data";
import { Button, Checkbox, Drawer, EmptyState, Field, InlineNotice, Input, Skeleton, StatusText } from "@/shared/ui";
import { DIFF_LABEL, qKey, qPath, TYPE_LABEL, type QuestionRow } from "@/features/questions/questionsApi";
import { ePath, type ExamDetail, type ExamItem } from "./examApi";
import s from "./Exam.module.css";

type Row = { question_id: string; title: string; type: ExamItem["type"]; topic: string; points: string };

const rowsOf = (d: ExamDetail): Row[] => d.items.map((i) => ({ question_id: i.question_id, title: i.title, type: i.type, topic: i.topic, points: i.points.replace(".", ",") }));
const num = (v: string) => Number(v.trim().replace(",", "."));
const fmt = (n: number) => n.toFixed(2).replace(".", ",");

/**
 * Tab `Câu hỏi`: chọn từ ngân hàng (chỉ câu đã duyệt), điểm từng câu, đổi thứ tự bằng nút `Lên` / `Xuống` (dùng được bằng bàn phím). Lưu cả danh sách một lần;
 * ngoài DRAFT danh sách khoá. Câu đã bị loại / lưu trữ sau khi thêm vẫn hiện, kèm lý do — lên lịch sẽ báo.
 */
export function ExamItems({ courseId, detail, onSaved }: { courseId: string; detail: ExamDetail; onSaved: (d: ExamDetail, note: string) => void }) {
  const draft = detail.status === "DRAFT";
  const serverRows = useMemo(() => rowsOf(detail), [detail]);
  const [edits, setEdits] = useState<Row[] | null>(null); // null = chưa sửa gì: hiện bản của máy chủ
  const rows = edits ?? serverRows;
  const setRows = (next: Row[]) => setEdits(next);
  const [picking, setPicking] = useState(false);
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [liveMsg, setLiveMsg] = useState("");
  const dirty = edits !== null && JSON.stringify(edits) !== JSON.stringify(serverRows);
  const flags = new Map(detail.items.map((i) => [i.question_id, i]));

  const total = rows.reduce((a, r) => a + (Number.isFinite(num(r.points)) ? num(r.points) : 0), 0);
  function move(i: number, by: -1 | 1) {
    const j = i + by;
    if (j < 0 || j >= rows.length) return;
    const next = rows.slice();
    [next[i], next[j]] = [next[j], next[i]];
    setRows(next);
    setLiveMsg(`Câu ${next[j].title} ở vị trí ${j + 1} trên ${rows.length}`);
  }

  async function save() {
    setPending(true);
    setErr(null);
    try {
      const r = await apiClient.put<ExamDetail>(`${ePath(courseId)}/${detail.id}/items`, { items: rows.map((x) => ({ question_id: x.question_id, points: num(x.points).toFixed(2) })), version: detail.version });
      setEdits(null);
      onSaved(r.data, "Đã lưu danh sách câu hỏi");
    } catch (x) {
      setErr(x);
    } finally {
      setPending(false);
    }
  }

  const problems = err instanceof ApiError && Array.isArray(err.details) ? (err.details as Array<{ field?: string; message?: string }>) : [];
  return (
    <div className={s.itemsTab}>
      {!draft && <InlineNotice compact>Bài đã lên lịch nên danh sách câu không sửa được. Muốn đổi, nhân bản bài thành bản nháp mới.</InlineNotice>}
      {rows.length === 0 ? (
        <EmptyState title="Chưa có câu hỏi nào">Chưa có câu nào trong bài. Chọn câu đã duyệt từ ngân hàng câu hỏi.</EmptyState>
      ) : (
        <>
          <ol className={s.items} aria-label="Câu hỏi của bài thi">
            {rows.map((r, i) => {
              const f = flags.get(r.question_id);
              const bad = f && (f.review_status !== "APPROVED" || f.archived);
              return (
                <li key={r.question_id} className={s.item}>
                  <span className={s.pos} aria-hidden>{i + 1}</span>
                  <span className={s.itemText}>
                    <span className={s.itemTitle}>{r.title}</span>
                    <span className={s.meta}>{TYPE_LABEL[r.type]} · {r.topic}</span>
                    {bad && <StatusText tone="red">{f.archived ? "Câu đã lưu trữ" : "Câu không còn ở trạng thái đã duyệt"}</StatusText>}
                  </span>
                  <Field label="Điểm" className={s.points}>
                    {(id) => <Input id={id} inputMode="decimal" aria-label={`Điểm của câu ${i + 1}`} value={r.points} disabled={!draft} onChange={(e) => setRows(rows.map((x, k) => (k === i ? { ...x, points: e.target.value } : x)))} />}
                  </Field>
                  {draft && (
                    <span className={s.itemActions}>
                      <Button size="sm" variant="ghost" aria-label={`Đưa câu ${r.title} lên`} disabled={i === 0} onClick={() => move(i, -1)}>Lên</Button>
                      <Button size="sm" variant="ghost" aria-label={`Đưa câu ${r.title} xuống`} disabled={i === rows.length - 1} onClick={() => move(i, 1)}>Xuống</Button>
                      <Button size="sm" variant="ghost" aria-label={`Gỡ câu ${r.title}`} onClick={() => setRows(rows.filter((_, k) => k !== i))}>Gỡ</Button>
                    </span>
                  )}
                </li>
              );
            })}
          </ol>
          <p className={s.sum}>Tổng điểm các câu: <strong>{fmt(total)}</strong> · điểm tối đa của bài: <strong>{detail.max_score.replace(".", ",")}</strong> (điểm các câu được quy về thang này)</p>
        </>
      )}
      <p className={s.srOnly} role="status" aria-live="polite">{liveMsg}</p>
      {err !== null && (problems.length > 0 ? (
        <InlineNotice tone="danger" title="Danh sách câu chưa hợp lệ">
          <ul className={s.problemList}>{problems.map((p, i) => <li key={i}>{p.message}</li>)}</ul>
        </InlineNotice>
      ) : err instanceof ApiError && err.code === "VERSION_CONFLICT" ? (
        <InlineNotice tone="warning" title="Có người vừa sửa bài thi này" action={<Button size="sm" onClick={() => onSaved(err.conflict!.current as ExamDetail, "")}>Xem bản mới</Button>}>Danh sách bạn đang sắp xếp vẫn được giữ.</InlineNotice>
      ) : (
        <ApiErrorNotice error={err} showTechnical />
      ))}
      {draft && (
        <div className={s.formActions}>
          <Button icon={<Plus aria-hidden />} onClick={() => setPicking(true)}>Thêm câu từ ngân hàng</Button>
          <Button onClick={() => void save()} loading={pending} disabled={!dirty || rows.length === 0}>Lưu danh sách câu</Button>
          {dirty && <span className={s.draftNote}>Chưa lưu</span>}
        </div>
      )}
      {picking && (
        <Picker
          courseId={courseId}
          taken={new Set(rows.map((r) => r.question_id))}
          onClose={() => setPicking(false)}
          onPick={(qs) => {
            setRows([...rows, ...qs.map((q) => ({ question_id: q.id, title: q.title, type: q.type, topic: q.topic, points: "1,00" }))]);
            setPicking(false);
          }}
        />
      )}
    </div>
  );
}

/** Chọn câu đã duyệt từ ngân hàng (có tìm theo tiêu đề); câu đã có trong bài hiện mờ, không chọn lại được. */
function Picker({ courseId, taken, onClose, onPick }: { courseId: string; taken: Set<string>; onClose: () => void; onPick: (qs: QuestionRow[]) => void }) {
  const [text, setText] = useState("");
  const [q, setQ] = useState("");
  const [chosen, setChosen] = useState<Map<string, QuestionRow>>(new Map());
  useEffect(() => {
    const t = setTimeout(() => setQ(text.trim()), 250);
    return () => clearTimeout(t);
  }, [text]);
  const list = useCursorList<QuestionRow>(qKey(courseId, "picker", q), qPath(courseId), { limit: 50, query: { review_status: "APPROVED", q: q || undefined } });
  const toggle = (r: QuestionRow, on: boolean) =>
    setChosen((prev) => {
      const next = new Map(prev);
      if (on) next.set(r.id, r);
      else next.delete(r.id);
      return next;
    });
  return (
    <Drawer open wide onClose={onClose} title="Chọn câu từ ngân hàng" description="Chỉ hiện câu đã duyệt và chưa lưu trữ."
      footer={<><Button variant="primary" disabled={chosen.size === 0} onClick={() => onPick([...chosen.values()])}>{chosen.size > 0 ? `Thêm ${chosen.size} câu` : "Thêm câu"}</Button><Button variant="text" onClick={onClose}>Đóng</Button></>}>
      <div className={s.picker}>
        <Field label="Tìm câu hỏi">{(id) => <Input id={id} type="search" placeholder="Tiêu đề…" value={text} maxLength={100} onChange={(e) => setText(e.target.value)} />}</Field>
        {list.isPending ? (
          <Skeleton lines={6} />
        ) : list.isError ? (
          <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} showTechnical />
        ) : list.items.length === 0 ? (
          <EmptyState title="Không có câu phù hợp">{q ? "Thử từ khoá khác." : "Chưa có câu nào được duyệt. Duyệt câu ở Ngân hàng câu hỏi trước."}</EmptyState>
        ) : (
          <ul className={s.pickList} aria-label="Câu đã duyệt">
            {list.items.map((r) => (
              <li key={r.id} className={s.pickRow}>
                <Checkbox
                  label={<span className={s.itemText}><span className={s.itemTitle}>{r.title}</span><span className={s.meta}>{TYPE_LABEL[r.type]} · {DIFF_LABEL[r.difficulty]} · {r.topic}{taken.has(r.id) ? " · đã có trong bài" : ""}</span></span>}
                  checked={taken.has(r.id) || chosen.has(r.id)}
                  disabled={taken.has(r.id)}
                  onChange={(e) => toggle(r, e.target.checked)}
                />
              </li>
            ))}
          </ul>
        )}
        {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button>}
      </div>
    </Drawer>
  );
}
