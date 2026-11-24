"use client";

import { useState } from "react";
import { StatusText } from "../ui";
import s from "./GradeCalculation.module.css";

/** Staff: `expr` + `value`. Sinh viên (P-11-15): `text` là cả câu (không cột giá trị) hoặc `note` + `value`; `detail` mở bằng «Chi tiết». */
export type CalcLine = { label: string; expr?: string; text?: string; note?: string; detail?: string[]; value?: string; quote?: string; page?: number; filled_by_user?: boolean };

const vi = (v: string) => v.replace(".", ",");

/**
 * Giải trình điểm học phần (DESIGN §14.11): mỗi dòng "nhãn · phép tính = giá trị" cho nhân viên, một câu tiếng Việt cho sinh viên; `Căn cứ` mở đoạn trích quy chế
 * TẠI CHỖ (kèm số trang); dòng do giảng viên đặt ghi "do giảng viên đặt". Các dòng do máy chủ (`Compute`) sinh — màn này không tính gì.
 */
export function GradeCalculation({ lines, caption }: { lines: CalcLine[]; caption: string }) {
  const [open, setOpen] = useState<string | null>(null);
  const toggle = (k: string) => setOpen(open === k ? null : k);
  return (
    <ol className={s.list} aria-label={caption} data-part="grade-calculation">
      {lines.map((l, i) => {
        const aside = l.expr || l.note || l.detail?.length || l.quote || l.filled_by_user;
        return (
          <li key={i} className={s.line}>
            <div className={s.main}>
              <span className={s.label}>{l.text ?? l.label}</span>
              {aside && (
                <span className={s.aside}>
                  {l.expr && <span className={s.expr}>{vi(l.expr)}</span>}
                  {l.note && <span className={s.expr}>{l.note}</span>}
                  {l.detail?.length ? (
                    <button type="button" className={s.cite} aria-expanded={open === `d${i}`} onClick={() => toggle(`d${i}`)}>
                      Chi tiết
                    </button>
                  ) : null}
                  {l.quote ? (
                    <button type="button" className={s.cite} aria-expanded={open === `q${i}`} onClick={() => toggle(`q${i}`)}>
                      {`Căn cứ · tr. ${l.page ?? ""}`}
                    </button>
                  ) : l.filled_by_user ? (
                    <StatusText tone="neutral">do giảng viên đặt</StatusText>
                  ) : null}
                </span>
              )}
              {l.text === undefined && l.value !== undefined && <span className={s.value}>{l.expr ? `= ${vi(l.value)}` : vi(l.value)}</span>}
            </div>
            {l.detail?.length && open === `d${i}` ? (
              <ul className={s.detail} aria-label="Chi tiết">
                {l.detail.map((d) => (
                  <li key={d}>{d}</li>
                ))}
              </ul>
            ) : null}
            {l.quote && open === `q${i}` && <blockquote className={s.quote}>{l.quote}</blockquote>}
          </li>
        );
      })}
    </ol>
  );
}
