"use client";

import { useId, useLayoutEffect, useMemo, useRef, useState, type ChangeEvent, type KeyboardEvent } from "react";
import s from "./CodeEditor.module.css";

export const MAX_SOURCE_BYTES = 65536;
const enc = new TextEncoder();
const bytes = (v: string) => enc.encode(v).length;
const fmt = (n: number) => n.toLocaleString("vi-VN");

/** Cắt chuỗi về ≤ `max` byte UTF-8 mà không cắt giữa một ký tự. */
function clampBytes(v: string, max: number): string {
  if (bytes(v) <= max) return v;
  let lo = 0;
  let hi = v.length;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (bytes(v.slice(0, mid)) <= max) lo = mid;
    else hi = mid - 1;
  }
  let out = v.slice(0, lo);
  if (out.length > 0 && /[\uD800-\uDBFF]$/.test(out)) out = out.slice(0, -1); // không để lại nửa cặp surrogate
  return out;
}

const INDENT = "    ";

/**
 * Trình soạn mã bằng `<textarea>` (không thư viện — US-PE-06 AC3): cột số dòng cuộn đồng bộ, `Tab` thụt 4 dấu cách / `Shift+Tab` bỏ thụt,
 * `Esc` rồi `Tab` để rời ô (không bẫy bàn phím), đếm byte `{x}/65.536` và chặn gõ quá giới hạn kèm lời. Dòng dài cuộn ngang TRONG ô.
 */
export function CodeEditor({ value, onChange, readOnly, label, onKeyEscape }: { value: string; onChange: (v: string) => void; readOnly?: boolean; label: string; onKeyEscape?: () => void }) {
  const area = useRef<HTMLTextAreaElement>(null);
  const gutter = useRef<HTMLPreElement>(null);
  const escaped = useRef(false); // vừa bấm Esc: Tab kế tiếp rời ô thay vì thụt
  const hintId = useId();
  const [full, setFull] = useState(false);
  const lines = useMemo(() => {
    const n = value.split("\n").length;
    return Array.from({ length: n }, (_, i) => i + 1).join("\n");
  }, [value]);
  const size = useMemo(() => bytes(value), [value]);

  // vị trí con trỏ sau khi sửa bằng bàn phím: đặt ngay sau lần vẽ lại (trước khi người dùng gõ tiếp)
  const sel = useRef<[number, number] | null>(null);
  useLayoutEffect(() => {
    if (sel.current && area.current) area.current.setSelectionRange(sel.current[0], sel.current[1]);
    sel.current = null;
  }, [value]);

  const apply = (next: string, selStart: number, selEnd: number) => {
    const clamped = clampBytes(next, MAX_SOURCE_BYTES);
    setFull(clamped.length < next.length);
    sel.current = [Math.min(selStart, clamped.length), Math.min(selEnd, clamped.length)];
    onChange(clamped);
  };

  function onInput(e: ChangeEvent<HTMLTextAreaElement>) {
    const v = e.target.value;
    if (bytes(v) > MAX_SOURCE_BYTES) {
      const caret = e.target.selectionStart;
      apply(v, caret, caret); // chặn: giữ phần đầu vừa đủ giới hạn
      return;
    }
    setFull(false);
    onChange(v);
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Escape") {
      escaped.current = true;
      onKeyEscape?.();
      return;
    }
    if (e.key !== "Tab") {
      escaped.current = false;
      return;
    }
    if (escaped.current) {
      escaped.current = false;
      return; // để trình duyệt chuyển focus
    }
    e.preventDefault();
    if (readOnly) return;
    const el = e.currentTarget;
    const { selectionStart: a, selectionEnd: b } = el;
    if (!e.shiftKey && a === b) {
      apply(value.slice(0, a) + INDENT + value.slice(b), a + INDENT.length, a + INDENT.length);
      return;
    }
    // nhiều dòng (hoặc Shift+Tab): thụt / bỏ thụt từng dòng đang chọn
    const start = value.lastIndexOf("\n", a - 1) + 1;
    const endIdx = value.indexOf("\n", b);
    const end = endIdx === -1 ? value.length : endIdx;
    const block = value.slice(start, end).split("\n");
    let firstDelta = 0;
    let total = 0;
    const out = block.map((ln, i) => {
      if (!e.shiftKey) {
        if (i === 0) firstDelta = INDENT.length;
        total += INDENT.length;
        return INDENT + ln;
      }
      const m = /^ {1,4}/.exec(ln)?.[0].length ?? 0;
      if (i === 0) firstDelta = -m;
      total -= m;
      return ln.slice(m);
    });
    apply(value.slice(0, start) + out.join("\n") + value.slice(end), Math.max(start, a + firstDelta), b + total);
  }

  return (
    <div className={s.wrap}>
      <div className={s.frame}>
        <pre ref={gutter} className={s.gutter} aria-hidden>{lines}</pre>
        <textarea
          ref={area}
          className={s.area}
          value={value}
          onChange={onInput}
          onKeyDown={onKeyDown}
          onBlur={() => (escaped.current = false)}
          onScroll={(e) => {
            if (gutter.current) gutter.current.scrollTop = e.currentTarget.scrollTop;
          }}
          readOnly={readOnly}
          spellCheck={false}
          autoCapitalize="off"
          autoCorrect="off"
          autoComplete="off"
          wrap="off"
          aria-label={label}
          aria-describedby={hintId}
          data-part="code-editor"
        />
      </div>
      <p id={hintId} className={s.hint}>
        <span>Nhấn Esc rồi Tab để rời khỏi ô soạn mã.</span>
        <span className={full ? s.full : undefined} role={full ? "status" : undefined}>
          {full ? `Đã đủ ${fmt(MAX_SOURCE_BYTES)} byte — phần gõ thêm không được nhận. ` : ""}{fmt(size)}/{fmt(MAX_SOURCE_BYTES)}
        </span>
      </p>
    </div>
  );
}
