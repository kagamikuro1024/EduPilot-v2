"use client";

import { Plus, X } from "lucide-react";
import { useMemo, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useAutosaveDraft } from "@/shared/data";
import { Button, Checkbox, Field, IconButton, InlineNotice, Input, SegmentedControl, Select, Textarea } from "@/shared/ui";
import { DIFF_LABEL, qPath, TYPE_LABEL, type Difficulty, type QType, type QuestionDetail } from "./questionsApi";
import s from "./Questions.module.css";

type OptRow = { body: string; correct: boolean; pinned: boolean };
type Draft = { type: QType; title: string; topic: string; difficulty: Difficulty; stem: string; explanation: string; options: OptRow[]; value: boolean | null };

const blank = (type: QType): Draft => ({
  type,
  title: "",
  topic: "",
  difficulty: "MEDIUM",
  stem: "",
  explanation: "",
  options: type === "MCQ_SINGLE" || type === "MCQ_MULTI" ? [1, 2].map(() => ({ body: "", correct: false, pinned: false })) : [],
  value: null,
});

function fromDetail(d: QuestionDetail): Draft {
  const right = new Set(d.answer_key?.option_ids ?? []);
  return {
    type: d.type,
    title: d.title,
    topic: d.topic,
    difficulty: d.difficulty,
    stem: d.stem,
    explanation: d.explanation ?? "",
    options: d.options.map((o) => ({ body: o.body, correct: right.has(o.id), pinned: o.pinned_last })),
    value: d.answer_key?.value ?? null,
  };
}

// Mã lỗi của máy chủ → câu tiếng Việt đặt ngay ở trường (details[].field là tên trường JSON; `options[i].body` và `correct` có dạng riêng).
const FIELD_TEXT: Record<string, string> = {
  options: "Câu trắc nghiệm có từ 2 đến 8 đáp án.",
  correct: "Chọn đúng số đáp án đúng cho loại câu này.",
};

/**
 * Soạn / sửa MỘT câu hỏi (trắc nghiệm, đúng–sai hoặc phần chung của bài lập trình). Sửa tại chỗ, tự lưu nháp theo `question:<id>` (UX.md); không bao giờ mất chữ đã gõ.
 * `detail` = câu đang sửa (kèm `version` cho khoá lạc quan); không có = tạo mới ở loại `createType`.
 */
export function QuestionForm({ courseId, userId, detail, createType, onSaved, onCancel }: {
  courseId: string;
  userId?: string;
  detail?: QuestionDetail;
  createType?: QType;
  onSaved: (q: QuestionDetail) => void;
  onCancel: () => void;
}) {
  const initial = useMemo(() => (detail ? fromDetail(detail) : blank(createType ?? "MCQ_SINGLE")), [detail, createType]);
  const draftKey = `question:${detail?.id ?? `new-${createType ?? "MCQ_SINGLE"}`}`;
  const saved = useAutosaveDraft(draftKey, { userId });
  const restored = useMemo(() => {
    try {
      return saved.value ? (JSON.parse(saved.value) as Draft) : null;
    } catch {
      return null;
    }
    // chỉ lúc mở: bản nháp đã khôi phục được dùng làm giá trị đầu
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const [d, setD] = useState<Draft>(restored && restored.type === initial.type ? restored : initial);
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const fe = fieldErrors(err);
  const locked = err instanceof ApiError && (err.code as string) === "QUESTION_IN_USE";
  const isMcq = d.type === "MCQ_SINGLE" || d.type === "MCQ_MULTI";

  function patch(next: Partial<Draft>) {
    setD((prev) => {
      const merged = { ...prev, ...next };
      saved.setValue(JSON.stringify(merged));
      return merged;
    });
  }
  const setOpt = (i: number, p: Partial<OptRow>) => patch({ options: d.options.map((o, k) => (k === i ? { ...o, ...p } : o)) });
  const setCorrect = (i: number, on: boolean) =>
    patch({ options: d.options.map((o, k) => (d.type === "MCQ_SINGLE" ? { ...o, correct: k === i && on } : k === i ? { ...o, correct: on } : o)) });

  async function submit(e?: React.FormEvent) {
    e?.preventDefault();
    setPending(true);
    setErr(null);
    const body: Record<string, unknown> = { type: d.type, title: d.title, topic: d.topic, difficulty: d.difficulty, stem: d.stem, explanation: d.explanation === "" ? null : d.explanation };
    if (isMcq) {
      body.options = d.options.map((o) => ({ body: o.body, pinned_last: o.pinned }));
      body.correct = d.options.flatMap((o, i) => (o.correct ? [i] : []));
    }
    if (d.type === "TRUE_FALSE") body.value = d.value;
    try {
      const r = detail ? await apiClient.put<QuestionDetail>(`${qPath(courseId)}/${detail.id}`, { ...body, version: detail.version }) : await apiClient.post<QuestionDetail>(qPath(courseId), body);
      saved.clear();
      onSaved(r.data);
    } catch (x) {
      setErr(x);
    } finally {
      setPending(false);
    }
  }

  return (
    <form className={s.form} onSubmit={submit} noValidate>
      {restored && !detail && <InlineNotice compact>Đã khôi phục bản nháp bạn đang soạn.</InlineNotice>}
      {restored && detail && <InlineNotice compact>Đã khôi phục phần bạn đang sửa dở.</InlineNotice>}
      {!detail && (
        <Field label="Loại câu hỏi">
          {() => (
            <SegmentedControl
              label="Loại câu hỏi"
              value={d.type}
              onChange={(type) => setD(blank(type as QType))}
              options={(["MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "CODE"] as QType[]).filter((t) => createType === "CODE" ? t === "CODE" : t !== "CODE").map((t) => ({ value: t, label: TYPE_LABEL[t] }))}
            />
          )}
        </Field>
      )}
      <Field label="Tiêu đề" required error={fe.title}>
        {(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.title)} value={d.title} maxLength={120} onChange={(e) => patch({ title: e.target.value })} />}
      </Field>
      <div className={s.formRow}>
        <Field label="Chủ đề" required error={fe.topic} className={s.grow}>
          {(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.topic)} value={d.topic} maxLength={80} onChange={(e) => patch({ topic: e.target.value })} />}
        </Field>
        <Field label="Độ khó">
          {(id) => (
            <Select id={id} value={d.difficulty} onChange={(e) => patch({ difficulty: e.target.value as Difficulty })}>
              {(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}
            </Select>
          )}
        </Field>
      </div>
      <Field label="Đề bài" required helper="Hỗ trợ Markdown cơ bản: đoạn, **đậm**, `mã`, danh sách, bảng. Thẻ HTML hiện thành chữ." error={fe.stem}>
        {(id, by) => <Textarea id={id} aria-describedby={by} invalid={Boolean(fe.stem)} rows={5} value={d.stem} onChange={(e) => patch({ stem: e.target.value })} />}
      </Field>

      {isMcq && (
        <fieldset className={s.options}>
          <legend className={s.legend}>Đáp án — {d.type === "MCQ_SINGLE" ? "chọn một đáp án đúng" : "chọn một hoặc nhiều đáp án đúng (ít nhất một đáp án sai)"}</legend>
          {d.options.map((o, i) => (
            <div key={i} className={s.optionRow}>
              <Checkbox label="Đúng" aria-label={`Đáp án ${i + 1} là đáp án đúng`} checked={o.correct} onChange={(e) => setCorrect(i, e.target.checked)} />
              <Input aria-label={`Nội dung đáp án ${i + 1}`} invalid={Boolean(fe[`options[${i}].body`])} value={o.body} maxLength={1000} onChange={(e) => setOpt(i, { body: e.target.value })} />
              <Checkbox label="Ghim cuối" aria-label={`Ghim đáp án ${i + 1} ở cuối khi xáo trộn`} checked={o.pinned} onChange={(e) => setOpt(i, { pinned: e.target.checked })} />
              <IconButton label={`Bỏ đáp án ${i + 1}`} size="sm" type="button" disabled={d.options.length <= 2} onClick={() => patch({ options: d.options.filter((_, k) => k !== i) })}>
                <X aria-hidden />
              </IconButton>
              {fe[`options[${i}].body`] && <p role="alert" className={s.fieldNote}>{fe[`options[${i}].body`]}</p>}
            </div>
          ))}
          {(fe.options || fe.correct) && <p role="alert" className={s.fieldNote}>{FIELD_TEXT[fe.options ? "options" : "correct"]}</p>}
          <Button type="button" size="sm" icon={<Plus aria-hidden />} disabled={d.options.length >= 8} onClick={() => patch({ options: [...d.options, { body: "", correct: false, pinned: false }] })}>
            Thêm đáp án
          </Button>
        </fieldset>
      )}
      {d.type === "TRUE_FALSE" && (
        <Field label="Đáp án đúng" required error={fe.value}>
          {() => (
            <SegmentedControl label="Đáp án đúng" value={d.value === null ? "" : d.value ? "true" : "false"} onChange={(v) => patch({ value: v === "true" })} options={[{ value: "true", label: "Đúng" }, { value: "false", label: "Sai" }]} />
          )}
        </Field>
      )}
      {d.type !== "CODE" && (
        <Field label="Giải thích" helper="Hiện cho sinh viên sau khi công bố kết quả (nếu bài thi bật xem đáp án)." error={fe.explanation}>
          {(id, by) => <Textarea id={id} aria-describedby={by} rows={3} value={d.explanation} onChange={(e) => patch({ explanation: e.target.value })} />}
        </Field>
      )}

      {locked && <InlineNotice tone="warning" title="Câu hỏi đang được dùng trong bài thi">Chỉ sửa được chủ đề, độ khó và giải thích. Nhân bản câu hỏi để sửa nội dung.</InlineNotice>}
      {err !== null && !locked && Object.keys(fe).length === 0 && <ApiErrorNotice error={err} showTechnical onRetry={() => void submit()} />}
      <div className={s.formActions}>
        <Button type="submit" variant="primary" loading={pending}>{detail ? "Lưu thay đổi" : "Tạo câu hỏi"}</Button>
        <Button type="button" variant="text" onClick={onCancel}>Huỷ</Button>
        <span className={s.draftNote} aria-live="polite">{saved.status === "saved" ? "Đã lưu nháp trên máy này" : saved.status === "saving" ? "Đang lưu nháp…" : ""}</span>
      </div>
    </form>
  );
}
