"use client";

import { Check, Tag, X } from "lucide-react";
import { useRef, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, newIdempotencyKey } from "@/shared/data";
import { userMessageFor } from "@/shared/i18n/vi";
import { Button, InlineNotice, Popover, Select, Field, Input } from "@/shared/ui";
import { useGuidanceFlags } from "@/features/assignments/guidanceApi";
import { DIFF_LABEL, qPath, type Difficulty } from "./questionsApi";
import s from "./Questions.module.css";

type Result = { id: string; ok: boolean; code?: string };
type Undo = { push: (m: React.ReactNode, onUndo?: () => void) => void; clear: () => void };
const MAX = 50;
const REJECT_DELAY_MS = 5000;
const REASONS: Array<{ value: string; label: string }> = [
  { value: "MO_HO", label: "Mơ hồ" }, { value: "SAI_DAP_AN", label: "Sai đáp án" }, { value: "TRUNG_LAP", label: "Trùng lặp" },
  { value: "LAC_DE", label: "Lạc đề" }, { value: "KHO_DE_BAT_THUONG", label: "Độ khó lạ" }, { value: "SAI_DINH_DANG", label: "Sai định dạng" },
];

/** Lý do loại chọn nhanh, bỏ qua được (M6): nằm cạnh dòng «Hoàn tác»; chọn lại cùng nhãn thì bỏ chọn. */
function ReasonChips({ onPick }: { onPick: (r: string | null) => void }) {
  const [v, setV] = useState<string | null>(null);
  return (
    <span role="group" aria-label="Lý do loại" className={s.reasons}>
      {REASONS.map((o) => (
        <button
          key={o.value}
          type="button"
          aria-pressed={v === o.value}
          className={s.reasonChip}
          onClick={() => {
            const next = v === o.value ? null : o.value;
            setV(next);
            onPick(next);
          }}
        >
          {o.label}
        </button>
      ))}
    </span>
  );
}
/** Lệnh `Loại` đang hoãn 5 s: rời trang / đóng tab (`pagehide`) thì gửi NGAY bằng `fetch keepalive`, không để mất (P-10-8 Q-S10-31). Ở mức module vì BulkBar bị gỡ ngay khi vùng chọn rỗng. */
const pendingRejects = new Set<(keepalive: boolean) => void>();
if (typeof window !== "undefined") window.addEventListener("pagehide", () => [...pendingRejects].forEach((run) => run(true)));

/**
 * Thanh hành động khi chọn hàng (US-P9-04 AC3): `Duyệt`, `Loại`, gắn chủ đề / độ khó. Kết quả TỪNG id: id lỗi ở lại trong vùng chọn, hàng lỗi hiện lý do tại chỗ.
 * `Loại` đảo ngược được: hàng ẩn ngay, dòng `Hoàn tác` 5 s, hết 5 s mới gửi (hoàn tác = không gửi gì). Mất mạng: "Chưa gửi được" + thử lại cùng Idempotency-Key.
 */
export function BulkBar({ courseId, selected, topics, onSelected, onFailed, onHide, onChanged, undo }: {
  courseId: string;
  selected: Set<string>;
  topics: string[];
  onSelected: (s: Set<string>) => void;
  onFailed: (m: Map<string, string>) => void;
  onHide: (fn: (prev: Set<string>) => Set<string>) => void;
  onChanged: () => void;
  undo: Undo;
}) {
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const [tagTopic, setTagTopic] = useState("");
  const [tagDiff, setTagDiff] = useState("");
  const key = useRef<string | null>(null);
  const timer = useRef<number | null>(null);
  const last = useRef<{ body: Record<string, unknown>; ids: string[] } | null>(null);
  const flags = useGuidanceFlags(courseId, true);
  const hints = flags.data?.some((f) => f.feature === "QUESTION_HINTS" && f.enabled) ?? false; // chỉ giảng viên đọc được cờ: TA không thấy nhãn
  const tooMany = selected.size > MAX;
  const ids = [...selected].slice(0, MAX);

  async function send(body: Record<string, unknown>, sendIds: string[], keepalive = false) {
    key.current ??= newIdempotencyKey();
    last.current = { body, ids: sendIds };
    setBusy(true);
    setErr(null);
    try {
      const r = await apiClient.post<{ results: Result[] }>(`${qPath(courseId)}/bulk`, { ...body, ids: sendIds }, { idempotencyKey: key.current, keepalive });
      key.current = null;
      const bad = new Map(r.data.results.filter((x) => !x.ok).map((x) => [x.id, userMessageFor(x.code ?? "INTERNAL")]));
      onFailed(bad);
      onSelected(new Set(bad.keys()));
      onHide((p) => new Set([...p].filter((id) => bad.has(id))));
      onChanged();
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }

  function reject() {
    const sendIds = ids;
    let reason: string | null = null;
    onHide((p) => new Set([...p, ...sendIds]));
    onSelected(new Set());
    const run = (keepalive: boolean) => {
      pendingRejects.delete(run);
      if (timer.current) window.clearTimeout(timer.current);
      void send({ action: "REJECT", ...(reason ? { reason } : {}) }, sendIds, keepalive);
    };
    pendingRejects.add(run);
    timer.current = window.setTimeout(() => run(false), REJECT_DELAY_MS);
    undo.push(hints ? <>{`Đã loại ${sendIds.length} câu`}<ReasonChips onPick={(r) => { reason = r; }} /></> : `Đã loại ${sendIds.length} câu`, () => {
      pendingRejects.delete(run);
      if (timer.current) window.clearTimeout(timer.current);
      onHide((p) => new Set([...p].filter((id) => !sendIds.includes(id))));
      onSelected(new Set(sendIds));
    });
  }

  return (
    <div className={s.bulk} role="toolbar" aria-label="Thao tác với câu đã chọn">
      <span className={s.bulkCount}>Đã chọn {selected.size}</span>
      <Button variant="primary" size="sm" icon={<Check aria-hidden />} loading={busy} disabled={tooMany} title={tooMany ? `Chọn tối đa ${MAX} câu` : undefined} onClick={() => void send({ action: "APPROVE" }, ids)}>Duyệt</Button>
      <Button size="sm" icon={<X aria-hidden />} disabled={tooMany || busy} title={tooMany ? `Chọn tối đa ${MAX} câu` : undefined} onClick={reject}>Loại</Button>
      <Popover
        label="Gắn nhãn"
        width={280}
        trigger={(p) => <Button size="sm" icon={<Tag aria-hidden />} disabled={tooMany || busy} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">Gắn nhãn</Button>}
      >
        {(close) => (
          <form className={s.bulkTags} onSubmit={(e) => { e.preventDefault(); void send({ action: "SET_TAGS", ...(tagTopic.trim() ? { topic: tagTopic.trim() } : {}), ...(tagDiff ? { difficulty: tagDiff } : {}) }, ids); close(); }}>
            <Field label="Chủ đề">{(id) => <Input id={id} list="bulk-topics" value={tagTopic} maxLength={80} onChange={(e) => setTagTopic(e.target.value)} />}</Field>
            <datalist id="bulk-topics">{topics.map((t) => <option key={t} value={t} />)}</datalist>
            <Field label="Độ khó">{(id) => <Select id={id} value={tagDiff} onChange={(e) => setTagDiff(e.target.value)}><option value="">Giữ nguyên</option>{(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}</Select>}</Field>
            <Button type="submit" variant="primary" size="sm" disabled={!tagTopic.trim() && !tagDiff}>Gắn nhãn</Button>
          </form>
        )}
      </Popover>
      <Button variant="text" size="sm" onClick={() => { onSelected(new Set()); onFailed(new Map()); }}>Bỏ chọn</Button>
      {err && (err.status === 0 ? <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => last.current && void send(last.current.body, last.current.ids)}>Thử lại</Button>}>Chưa gửi được.</InlineNotice> : <ApiErrorNotice error={err} />)}
    </div>
  );
}
