"use client";

import { useState } from "react";
import { StatusText } from "../ui";
import s from "./GradeCalculation.module.css";

export type CalcLine = { label: string; expr: string; value: string; quote?: string; page?: number; filled_by_user?: boolean };

const vi = (v: string) => v.replace(".", ",");

/**
 * Giải trình điểm học phần dạng phép tính tuyến tính (DESIGN §14.11): mỗi dòng "nhãn · biểu thức = giá trị"; `Căn cứ` mở đoạn trích quy chế TẠI CHỖ
 * (kèm số trang); dòng do giảng viên đặt ghi "do giảng viên đặt". Các dòng do máy chủ (`Compute`) sinh — màn này không tính gì.
 */
export function GradeCalculation({ lines, caption }: { lines: CalcLine[]; caption: string }) {
  const [open, setOpen] = useState<number | null>(null);
  return (
    <ol className={s.list} aria-label={caption} data-part="grade-calculation">
      {lines.map((l, i) => (
        <li key={i} className={s.line}>
          <div className={s.main}>
            <span className={s.label}>{l.label}</span>
            {(l.expr || l.quote || l.filled_by_user) && (
              <span className={s.aside}>
                {l.expr && <span className={s.expr}>{vi(l.expr)}</span>}
                {l.quote ? (
                  <button type="button" className={s.cite} aria-expanded={open === i} onClick={() => setOpen(open === i ? null : i)}>
                    {`Căn cứ · tr. ${l.page ?? ""}`}
                  </button>
                ) : l.filled_by_user ? (
                  <StatusText tone="neutral">do giảng viên đặt</StatusText>
                ) : null}
              </span>
            )}
            <span className={s.value}>{l.expr ? `= ${vi(l.value)}` : vi(l.value)}</span>
          </div>
          {l.quote && open === i && <blockquote className={s.quote}>{l.quote}</blockquote>}
        </li>
      ))}
    </ol>
  );
}
