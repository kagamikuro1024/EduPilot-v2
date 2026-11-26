"use client";

import { useState } from "react";
import { Button, Field, Input } from "@/shared/ui";
import type { Route } from "./api";
import s from "./llm.module.css";

const PARAMS = [
  { key: "temperature", label: "Nhiệt độ", unit: "0–2", lo: 0, hi: 2, int: false },
  { key: "max_tokens", label: "Số token tối đa mỗi câu trả lời", unit: "1–32.768", lo: 1, hi: 32768, int: true },
  { key: "timeout_s", label: "Thời gian chờ", unit: "giây, 1–300", lo: 1, hi: 300, int: true },
  { key: "retries", label: "Số lần thử lại", unit: "0–5", lo: 0, hi: 5, int: true },
] as const;

export function Advanced({ route, draft, setDraft, onSave, readOnly }: { route: Route; draft: Record<string, string>; setDraft: (d: Record<string, string>) => void; onSave: (p: Record<string, number>) => void; readOnly: boolean }) {
  const [errs, setErrs] = useState<Record<string, string>>({});
  function save() {
    const e: Record<string, string> = {};
    const out: Record<string, number> = {};
    for (const p of PARAMS) {
      const raw = (draft[p.key] ?? "").trim().replace(",", ".");
      if (raw === "") continue;
      const n = Number(raw);
      if (!Number.isFinite(n) || n < p.lo || n > p.hi || (p.int && !Number.isInteger(n))) e[p.key] = `${p.label}: nhập ${p.int ? "số nguyên" : "số"} trong khoảng ${p.unit}.`;
      else out[p.key] = n;
    }
    setErrs(e);
    if (Object.keys(e).length === 0) onSave(out);
  }
  return (
    <div className={s.advanced} data-part="route-advanced">
      <div className={s.pair}>
        {PARAMS.map((p) => (
          <Field key={p.key} label={p.label} helper={`Đơn vị: ${p.unit}. Để trống = mặc định.`} error={errs[p.key]}>
            {(id, d) => (
              <Input id={id} aria-describedby={d} invalid={Boolean(errs[p.key])} inputMode="decimal" readOnly={readOnly} value={draft[p.key] ?? ""} onChange={(e) => setDraft({ ...draft, [p.key]: e.target.value })} />
            )}
          </Field>
        ))}
      </div>
      {!readOnly && (
        <div>
          <Button variant="primary" size="sm" onClick={save}>
            Lưu
          </Button>
        </div>
      )}
      <span className="ep-sr-only">{route.task}</span>
    </div>
  );
}
