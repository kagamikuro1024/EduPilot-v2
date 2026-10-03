"use client";

import { CornerDownLeft, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import type { NavItem } from "@/shared/shell/nav";
import dlg from "./Dialog.module.css";
import { useScrollLock } from "./useScrollLock";
import s from "./CommandPalette.module.css";

// Bỏ dấu để gõ "diem" vẫn ra "Điểm danh".
const fold = (v: string) => v.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/đ/g, "d");

/** "Tìm nhanh hoặc đi đến…" (⌘K, /): lọc theo tên route của vai trò hiện tại, Enter để đi. */
export function CommandPalette({ open, tick = 0, onClose, items, initialQuery = "", loading }: { open: boolean; /** tăng mỗi lần yêu cầu mở (mở lại khi `open` không đổi giá trị) */ tick?: number; onClose: () => void; items: NavItem[]; initialQuery?: string; loading?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  const router = useRouter();
  const [q, setQ] = useState("");
  const [active, setActive] = useState(0);
  const uid = useId();
  useScrollLock(open);

  const results = useMemo(() => {
    const needle = fold(q.trim());
    return needle ? items.filter((i) => fold(i.label).includes(needle)) : items;
  }, [q, items]);

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- đồng bộ DOM <dialog> với prop `open`
      if (initialQuery) setQ(initialQuery);
      d.showModal();
    }
    if (!open && d.open) {
      d.close();
      setQ(""); // đặt lại khi ĐÓNG: mở lại không ghi đè chữ người dùng vừa gõ
      setActive(0);
    }
  }, [open, tick, initialQuery]);

  function go(item: NavItem | undefined) {
    if (!item) return;
    onClose();
    router.push(item.href);
  }

  return (
    <dialog ref={ref} className={[dlg.overlay, dlg.dialog, s.palette].join(" ")} onCancel={(e) => { e.preventDefault(); onClose(); }} /* Esc: đóng qua state (không dựa sự kiện close đến muộn) */ onClick={(e) => e.target === ref.current && onClose()} aria-label="Tìm nhanh">
      <div className={s.search}>
        <Search aria-hidden />
        <input
          autoFocus
          value={q}
          placeholder="Tìm nhanh hoặc đi đến…"
          aria-label="Tìm nhanh hoặc đi đến"
          role="combobox"
          aria-expanded="true"
          aria-controls={`${uid}-list`}
          aria-autocomplete="list"
          aria-activedescendant={results[active] && !loading ? `${uid}-opt-${active}` : undefined}
          onChange={(e) => {
            setQ(e.target.value);
            setActive(0);
          }}
          onKeyDown={(e) => {
            if (e.key === "ArrowDown") {
              e.preventDefault();
              setActive((a) => Math.min(results.length - 1, a + 1));
            }
            if (e.key === "ArrowUp") {
              e.preventDefault();
              setActive((a) => Math.max(0, a - 1));
            }
            if (e.key === "Enter") go(results[active]);
          }}
        />
      </div>
      <ul id={`${uid}-list`} className={s.list} role="listbox" aria-label="Kết quả" aria-busy={loading || undefined}>
        {loading && (
          <li className={s.none}>
            <span className="ep-sr-only">Đang tải</span>
            <span className={s.sk} />
          </li>
        )}
        {!loading && results.map((it, i) => {
          const Icon = it.icon;
          return (
            <li key={it.href} id={`${uid}-opt-${i}`} role="option" aria-selected={i === active}>
              <button type="button" className={s.item} onMouseEnter={() => setActive(i)} onClick={() => go(it)}>
                <Icon aria-hidden />
                <span>{it.label}</span>
                {i === active && <CornerDownLeft className={s.enter} aria-hidden />}
              </button>
            </li>
          );
        })}
        {!loading && results.length === 0 && <li className={s.none}>Không thấy mục nào khớp. Thử từ khác, ví dụ &ldquo;điểm&rdquo;.</li>}
      </ul>
    </dialog>
  );
}
