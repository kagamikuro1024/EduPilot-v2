"use client";

import type { ReactNode } from "react";
import s from "./Tabs.module.css";

type Option<V extends string> = { value: V; label: ReactNode; count?: number };

/** Chuyển giữa các phần lớn của MỘT đối tượng (DESIGN.md §10.8). */
export function Tabs<V extends string>({ value, onChange, options, label }: { value: V; onChange: (v: V) => void; options: Option<V>[]; label: string }) {
  return (
    <div role="tablist" aria-label={label} className={s.tabs}>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          role="tab"
          aria-selected={o.value === value}
          className={s.tab}
          onClick={() => onChange(o.value)}
          onKeyDown={(e) => {
            const i = options.findIndex((x) => x.value === value);
            if (e.key === "ArrowRight") onChange(options[(i + 1) % options.length].value);
            if (e.key === "ArrowLeft") onChange(options[(i - 1 + options.length) % options.length].value);
          }}
        >
          {o.label}
          {o.count !== undefined && <span className={s.count}>{o.count}</span>}
        </button>
      ))}
    </div>
  );
}

/** 2–4 chế độ loại trừ nhau đổi khung nhìn cục bộ (DESIGN.md §10.7). */
export function SegmentedControl<V extends string>({ value, onChange, options, label }: { value: V; onChange: (v: V) => void; options: Option<V>[]; label: string }) {
  return (
    <div role="radiogroup" aria-label={label} className={s.segmented}>
      {options.map((o) => (
        <button key={o.value} type="button" role="radio" aria-checked={o.value === value} className={s.segment} onClick={() => onChange(o.value)}>
          {o.label}
          {o.count !== undefined && <span className={s.count}>{o.count}</span>}
        </button>
      ))}
    </div>
  );
}

/** Bộ lọc nhanh dạng chip bật/tắt (nhiều lựa chọn). */
export function FilterChips<V extends string>({ value, onChange, options, label }: { value: V[]; onChange: (v: V[]) => void; options: Option<V>[]; label: string }) {
  return (
    <div role="group" aria-label={label} className={s.chips}>
      {options.map((o) => {
        const on = value.includes(o.value);
        return (
          <button
            key={o.value}
            type="button"
            aria-pressed={on}
            className={s.chip}
            onClick={() => onChange(on ? value.filter((v) => v !== o.value) : [...value, o.value])}
          >
            {o.label}
            {o.count !== undefined && <span className={s.count}>{o.count}</span>}
          </button>
        );
      })}
    </div>
  );
}
