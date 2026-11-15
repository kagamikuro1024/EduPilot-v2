"use client";

import { Check, X } from "lucide-react";
import { Input, Textarea } from "@/shared/ui";
import type { Ans, PFeedback, PQuestion } from "./practiceApi";
import s from "./Practice.module.css";

export const MAX_ESSAY = 4000;
const LETTERS = "ABCDEFGH";

/** Một đáp án hợp lệ (đủ để gửi): có chọn / có chữ. */
export function filled(q: PQuestion, v: Ans | undefined): boolean {
  if (!v) return false;
  if ("option_ids" in v) return v.option_ids.length > 0;
  if ("value" in v) return typeof v.value === "boolean";
  return v.text.trim().length > 0 && (q.type !== "ESSAY" || v.text.length <= MAX_ESSAY);
}

/**
 * Ô trả lời một câu (DESIGN §14.19). `fb` có mặt ⇒ đã chấm: tô phương án đúng / sai. MCQ chọn một: bấm là đổi; chọn nhiều: bấm để bật / tắt;
 * đúng–sai là hai phương án; trả lời ngắn một dòng; tự luận nhiều dòng, tối đa 4.000 ký tự.
 */
export function QuestionInput({ q, value, onChange, disabled, fb }: { q: PQuestion; value: Ans | undefined; onChange: (v: Ans) => void; disabled?: boolean; fb?: PFeedback | null }) {
  const keyIds = new Set((fb?.key.option_ids as string[] | undefined) ?? []);
  const keyVal = fb && typeof fb.key.value === "boolean" ? (fb.key.value as boolean) : undefined;
  const picked = new Set(value && "option_ids" in value ? value.option_ids : []);

  if (q.type === "SHORT") {
    const text = value && "text" in value ? value.text : "";
    return <Input aria-label="Câu trả lời ngắn" value={text} maxLength={200} disabled={disabled} placeholder="Trả lời ngắn gọn" onChange={(e) => onChange({ text: e.target.value })} />;
  }
  if (q.type === "ESSAY") {
    const text = value && "text" in value ? value.text : "";
    return (
      <div className={s.essay}>
        <Textarea aria-label="Bài viết" value={text} rows={8} maxLength={MAX_ESSAY} disabled={disabled} onChange={(e) => onChange({ text: e.target.value })} />
        <span className={s.count} aria-live="off">{text.length.toLocaleString("vi-VN")}/{MAX_ESSAY.toLocaleString("vi-VN")}</span>
      </div>
    );
  }
  if (q.type === "TRUE_FALSE") {
    const cur = value && "value" in value ? value.value : undefined;
    return (
      <ul className={s.options} aria-label="Đúng hoặc sai">
        {[true, false].map((v) => {
          const state = keyVal === undefined ? "" : v === keyVal ? s.optRight : v === cur ? s.optWrong : "";
          return (
            <li key={String(v)}>
              <button type="button" className={[s.option, cur === v ? s.optPicked : "", state].join(" ")} aria-pressed={cur === v} disabled={disabled} onClick={() => onChange({ value: v })}>
                <span className={s.optMark} aria-hidden>{v ? "Đ" : "S"}</span>
                {v ? "Đúng" : "Sai"}
                <Mark right={keyVal !== undefined && v === keyVal} wrong={keyVal !== undefined && v === cur && v !== keyVal} />
              </button>
            </li>
          );
        })}
      </ul>
    );
  }
  const multi = q.type === "MCQ_MULTI";
  return (
    <ul className={s.options} aria-label="Các đáp án">
      {q.options.map((o, i) => {
        const on = picked.has(o.id);
        const state = !fb ? "" : keyIds.has(o.id) ? s.optRight : on ? s.optWrong : "";
        return (
          <li key={o.id}>
            <button
              type="button"
              className={[s.option, on ? s.optPicked : "", state].join(" ")}
              aria-pressed={on}
              disabled={disabled}
              onClick={() => {
                const next = new Set(multi ? picked : []);
                if (multi && on) next.delete(o.id);
                else next.add(o.id);
                onChange({ option_ids: q.options.filter((x) => next.has(x.id)).map((x) => x.id) });
              }}
            >
              <span className={s.optMark} aria-hidden>{LETTERS[i] ?? i + 1}</span>
              {o.body}
              <Mark right={Boolean(fb) && keyIds.has(o.id)} wrong={Boolean(fb) && on && !keyIds.has(o.id)} />
            </button>
          </li>
        );
      })}
    </ul>
  );
}

function Mark({ right, wrong }: { right: boolean; wrong: boolean }) {
  if (right) return <Check className={s.optIcon} aria-label="Đáp án đúng" />;
  if (wrong) return <X className={s.optIcon} aria-label="Bạn chọn" />;
  return null;
}
