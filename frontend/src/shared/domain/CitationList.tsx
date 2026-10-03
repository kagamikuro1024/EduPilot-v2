"use client";

import { ChevronDown } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import s from "./CitationList.module.css";

export type Citation = { id: string; title: string; page?: number; excerpt: ReactNode };

/**
 * "Nguồn tham khảo (n)": mỗi nguồn là tên tài liệu + số trang; bấm mở đoạn trích TẠI CHỖ
 * (không điều hướng, không tab mới) — `aria-expanded` trên nút. Rỗng → câu giải thích.
 */
export function CitationList({ items, defaultOpenId }: { items: Citation[]; defaultOpenId?: string }) {
  const [open, setOpen] = useState<string | null>(defaultOpenId ?? null);
  const base = useId();
  if (items.length === 0) return <p className={s.empty}>Chưa có nguồn tham khảo cho câu trả lời này.</p>;
  return (
    <section className={s.root} aria-label={`Nguồn tham khảo (${items.length})`}>
      <h3 className={s.title}>{`Nguồn tham khảo (${items.length})`}</h3>
      <ul className={s.list}>
        {items.map((c) => {
          const on = open === c.id;
          const panel = `${base}-${c.id}`;
          return (
            <li key={c.id} className={s.item}>
              <button type="button" className={s.head} aria-expanded={on} aria-controls={panel} onClick={() => setOpen(on ? null : c.id)}>
                <span className={s.name}>{c.title}</span>
                {c.page !== undefined && <span className={s.page}>{`tr. ${c.page}`}</span>}
                <ChevronDown className={s.chev} aria-hidden />
              </button>
              {on && (
                <blockquote id={panel} className={s.excerpt}>
                  {c.excerpt}
                </blockquote>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
