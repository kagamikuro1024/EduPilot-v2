"use client";

import { useMemo, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useAutosaveDraft } from "@/shared/data";
import { Button, Checkbox, Field, InlineNotice, Input, PanelSection, Select, Textarea } from "@/shared/ui";
import { ePath, fromInput, toInput, type ExamDetail } from "./examApi";
import s from "./Exam.module.css";

type Form = {
  title: string;
  instructions: string;
  opens: string;
  closes: string;
  duration: string;
  shuffleQ: boolean;
  shuffleO: boolean;
  maxScore: string;
  step: string;
  multi: "PARTIAL" | "ALL_OR_NOTHING";
  reveal: boolean;
  appealDays: string;
};

const comma = (v: string) => v.replace(".", ",");
const dot = (v: string) => v.trim().replace(",", ".");

function fromDetail(d: ExamDetail): Form {
  return {
    title: d.title,
    instructions: d.instructions ?? "",
    opens: toInput(d.opens_at),
    closes: toInput(d.closes_at),
    duration: d.duration_minutes ? String(d.duration_minutes) : "",
    shuffleQ: d.shuffle_questions,
    shuffleO: d.shuffle_options,
    maxScore: comma(d.max_score),
    step: d.rounding_step,
    multi: d.multi_scoring,
    reveal: d.reveal_answers,
    appealDays: String(d.appeal_days),
  };
}

// Mã lỗi của máy chủ → trường trên form (details[].field là tên trường JSON).
const FIELD_OF: Record<string, keyof Form> = { title: "title", instructions: "instructions", opens_at: "opens", closes_at: "closes", duration_minutes: "duration", max_score: "maxScore", rounding_step: "step", multi_scoring: "multi", appeal_days: "appealDays" };

/**
 * Tab `Thông tin`: giờ mở / đóng, thời lượng, xáo trộn, thang điểm, hiện đáp án, hạn phúc khảo (US-PE-04 AC12). Sửa tại chỗ, tự lưu nháp theo `exam:<id>` (không mất chữ khi mạng xấu);
 * ngoài DRAFT chỉ tiêu đề, hướng dẫn, hiện đáp án và hạn phúc khảo sửa được — phần còn lại khoá kèm lý do.
 */
export function ExamInfo({ courseId, userId, detail, onSaved }: { courseId: string; userId?: string; detail: ExamDetail; onSaved: (d: ExamDetail, note: string) => void }) {
  const draft = detail.status === "DRAFT";
  const saved = useAutosaveDraft(`exam:${detail.id}`, { userId });
  const restored = useMemo(() => {
    try {
      return saved.value ? (JSON.parse(saved.value) as Form) : null;
    } catch {
      return null;
    }
    // chỉ lúc mở: bản nháp đã khôi phục được dùng làm giá trị đầu
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const [f, setF] = useState<Form>(restored ?? fromDetail(detail));
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const server = fieldErrors(err);
  const fe: Partial<Record<keyof Form, string>> = {};
  for (const [k, v] of Object.entries(server)) if (FIELD_OF[k]) fe[FIELD_OF[k]] = v;
  const conflict = err instanceof ApiError && err.code === "VERSION_CONFLICT";
  const dirty = JSON.stringify(f) !== JSON.stringify(fromDetail(detail));

  function patch(next: Partial<Form>) {
    setF((prev) => {
      const merged = { ...prev, ...next };
      saved.setValue(JSON.stringify(merged));
      return merged;
    });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setPending(true);
    setErr(null);
    const body: Record<string, unknown> = { title: f.title.trim(), instructions: f.instructions, reveal_answers: f.reveal, appeal_days: Number(f.appealDays), version: detail.version };
    if (draft) {
      const opens = fromInput(f.opens);
      const closes = fromInput(f.closes);
      if (opens) body.opens_at = opens;
      if (closes) body.closes_at = closes;
      if (f.duration !== "") body.duration_minutes = Number(f.duration);
      Object.assign(body, { shuffle_questions: f.shuffleQ, shuffle_options: f.shuffleO, max_score: dot(f.maxScore), rounding_step: f.step, multi_scoring: f.multi });
    }
    try {
      const r = await apiClient.put<ExamDetail>(`${ePath(courseId)}/${detail.id}`, body);
      saved.clear();
      setF(fromDetail(r.data));
      onSaved(r.data, "Đã lưu thông tin bài thi");
    } catch (x) {
      setErr(x);
    } finally {
      setPending(false);
    }
  }

  const lockNote = draft ? undefined : "Bài đã lên lịch nên phần này không sửa được. Đổi giờ đóng bằng Gia hạn.";
  return (
    <form className={s.form} onSubmit={submit} noValidate aria-label="Thông tin bài thi">
      <PanelSection>
      {restored && <InlineNotice compact>Đã khôi phục phần bạn đang sửa dở.</InlineNotice>}
      {!draft && <InlineNotice compact>Bài đã lên lịch: chỉ tiêu đề, hướng dẫn, hiện đáp án và hạn xem lại điểm còn sửa được.</InlineNotice>}
      <Field label="Tiêu đề" required error={fe.title}>
        {(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.title)} value={f.title} maxLength={120} onChange={(e) => patch({ title: e.target.value })} />}
      </Field>
      <Field label="Hướng dẫn cho sinh viên" helper="Hỗ trợ Markdown cơ bản. Sinh viên đọc ở màn giới thiệu trước khi bấm Bắt đầu làm bài." error={fe.instructions}>
        {(id, by) => <Textarea id={id} aria-describedby={by} invalid={Boolean(fe.instructions)} rows={4} value={f.instructions} maxLength={4000} onChange={(e) => patch({ instructions: e.target.value })} />}
      </Field>
      </PanelSection>
      <PanelSection title="Thời gian">
      <div className={s.formRow}>
        <Field label="Mở lúc" helper={lockNote ?? "Giờ Việt Nam. Phải sau bây giờ ít nhất 1 phút khi lên lịch."} error={fe.opens} className={s.grow}>
          {(id, by) => <Input id={id} aria-describedby={by} type="datetime-local" invalid={Boolean(fe.opens)} disabled={!draft} value={f.opens} onChange={(e) => patch({ opens: e.target.value })} />}
        </Field>
        <Field label="Đóng lúc" helper={draft ? "Sau giờ mở." : undefined} error={fe.closes} className={s.grow}>
          {(id, by) => <Input id={id} aria-describedby={by} type="datetime-local" invalid={Boolean(fe.closes)} disabled={!draft} value={f.closes} onChange={(e) => patch({ closes: e.target.value })} />}
        </Field>
        <Field label="Thời lượng (phút)" helper="5–300, không dài hơn khung giờ." error={fe.duration}>
          {(id, by) => <Input id={id} aria-describedby={by} type="number" inputMode="numeric" min={5} max={300} invalid={Boolean(fe.duration)} disabled={!draft} value={f.duration} onChange={(e) => patch({ duration: e.target.value })} />}
        </Field>
      </div>
      </PanelSection>
      <PanelSection>
      <fieldset className={s.group} disabled={!draft}>
        <legend className={s.legend}>Xáo trộn</legend>
        <Checkbox label="Xáo thứ tự câu hỏi cho từng sinh viên" checked={f.shuffleQ} onChange={(e) => patch({ shuffleQ: e.target.checked })} />
        <Checkbox label="Xáo thứ tự đáp án" checked={f.shuffleO} onChange={(e) => patch({ shuffleO: e.target.checked })} />
      </fieldset>
      </PanelSection>
      <PanelSection title="Chấm điểm">
      <div className={s.formRow}>
        <Field label="Điểm tối đa" error={fe.maxScore} helper="Điểm các câu được quy về thang này.">
          {(id, by) => <Input id={id} aria-describedby={by} inputMode="decimal" invalid={Boolean(fe.maxScore)} disabled={!draft} value={f.maxScore} onChange={(e) => patch({ maxScore: e.target.value })} />}
        </Field>
        <Field label="Làm tròn điểm tới" error={fe.step}>
          {(id) => (
            <Select id={id} disabled={!draft} value={f.step} onChange={(e) => patch({ step: e.target.value })}>
              {["0.01", "0.10", "0.25", "0.50", "1.00"].map((v) => <option key={v} value={v}>{comma(v)} điểm</option>)}
            </Select>
          )}
        </Field>
        <Field label="Câu nhiều đáp án" error={fe.multi}>
          {(id) => (
            <Select id={id} disabled={!draft} value={f.multi} onChange={(e) => patch({ multi: e.target.value as Form["multi"] })}>
              <option value="PARTIAL">Chọn sai bị trừ điểm của câu</option>
              <option value="ALL_OR_NOTHING">Chỉ được điểm khi chọn đủ và đúng</option>
            </Select>
          )}
        </Field>
      </div>
      </PanelSection>
      <PanelSection>
      <fieldset className={s.group}>
        <legend className={s.legend}>Sau khi công bố điểm</legend>
        <Checkbox label="Cho sinh viên xem đáp án đúng và giải thích" checked={f.reveal} onChange={(e) => patch({ reveal: e.target.checked })} />
        <Field label="Hạn gửi yêu cầu xem lại điểm (ngày)" helper="0 = không nhận yêu cầu xem lại." error={fe.appealDays} className={s.narrow}>
          {(id, by) => <Input id={id} aria-describedby={by} type="number" inputMode="numeric" min={0} max={30} invalid={Boolean(fe.appealDays)} value={f.appealDays} onChange={(e) => patch({ appealDays: e.target.value })} />}
        </Field>
      </fieldset>
      </PanelSection>
      <PanelSection>
      {conflict ? (
        <InlineNotice tone="warning" title="Có người vừa sửa bài thi này" action={<Button size="sm" onClick={() => onSaved((err as ApiError).conflict!.current as ExamDetail, "")}>Xem bản mới</Button>}>
          Chữ bạn đã gõ vẫn được giữ. Xem bản mới nhất rồi lưu lại.
        </InlineNotice>
      ) : (
        err !== null && !Object.keys(server).length && <ApiErrorNotice error={err} showTechnical />
      )}
      <div className={s.formActions}>
        <Button type="submit" loading={pending} disabled={!dirty}>Lưu thông tin</Button>
        {saved.status === "saved" && dirty && <span className={s.draftNote}>Bản nháp đã lưu trên máy này</span>}
      </div>
      </PanelSection>
    </form>
  );
}
