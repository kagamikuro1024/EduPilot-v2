"use client";

import { Check, MoreHorizontal } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import s from "./Menu.module.css";

/**
 * Popover neo vào nút kích hoạt: đóng khi bấm ra ngoài hoặc Esc, trả focus về nút.
 * Dùng cho menu tràn, bộ chọn lớp, thông báo.
 */
export function Popover({
  trigger,
  children,
  align = "end",
  width,
  label,
  defaultOpen = false,
}: {
  trigger: (props: { open: boolean; toggle: () => void; "aria-expanded": boolean; "aria-haspopup": "true" }) => ReactNode;
  children: ReactNode | ((close: () => void) => ReactNode);
  align?: "start" | "end";
  width?: number;
  label?: string;
  /** mở sẵn (trang mẫu /dev/ui) */
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (root.current && !root.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setOpen(false);
      (root.current?.querySelector("[aria-haspopup]") as HTMLElement | null)?.focus();
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const close = () => setOpen(false);
  return (
    <div className={s.root} ref={root}>
      {trigger({ open, toggle: () => setOpen((o) => !o), "aria-expanded": open, "aria-haspopup": "true" })}
      {open && (
        <div className={[s.panel, align === "start" ? s.start : s.end].join(" ")} style={{ width }} role="dialog" aria-label={label}>
          {typeof children === "function" ? children(close) : children}
        </div>
      )}
    </div>
  );
}

export type MenuItem = { label: ReactNode; onSelect: () => void; danger?: boolean; icon?: ReactNode; hint?: ReactNode; disabled?: boolean; selected?: boolean };

/** Menu tràn "⋯" cho hành động thứ ba trở đi (DESIGN.md §12). */
export function OverflowMenu({ items, label = "Thêm hành động" }: { items: MenuItem[]; label?: string }) {
  return (
    <Popover
      label={label}
      width={220}
      trigger={(p) => (
        <button type="button" className={s.dots} aria-label={label} title={label} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
          <MoreHorizontal aria-hidden />
        </button>
      )}
    >
      {(close) => <MenuList items={items} onPicked={close} autoFocus />}
    </Popover>
  );
}

/** Danh sách lệnh: mở ra là focus mục đầu; ↑ ↓ đổi mục (vòng), Home / End, Enter chọn (nút gốc). */
export function MenuList({ items, onPicked, autoFocus }: { items: MenuItem[]; onPicked?: () => void; /** focus mục đầu khi mở (menu trong Popover); danh sách tĩnh thì không */ autoFocus?: boolean }) {
  const ref = useRef<HTMLUListElement>(null);
  const enabled = () => Array.from(ref.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]:not(:disabled)') ?? []);
  useEffect(() => {
    if (autoFocus) enabled()[0]?.focus();
  }, [autoFocus]);
  function onKeyDown(e: React.KeyboardEvent) {
    const list = enabled();
    if (list.length === 0) return;
    const i = list.indexOf(document.activeElement as HTMLElement);
    let n = -1;
    if (e.key === "ArrowDown") n = (i + 1) % list.length;
    else if (e.key === "ArrowUp") n = (i - 1 + list.length) % list.length;
    else if (e.key === "Home") n = 0;
    else if (e.key === "End") n = list.length - 1;
    if (n >= 0) {
      e.preventDefault();
      list[n].focus();
    }
  }
  return (
    <ul className={s.menu} role="menu" ref={ref} onKeyDown={onKeyDown}>
      {items.map((it, i) => (
        <li key={i} role="none">
          <button
            type="button"
            role={it.selected === undefined ? "menuitem" : "menuitemradio"}
            aria-checked={it.selected}
            disabled={it.disabled}
            className={[s.item, it.danger ? s.danger : "", it.selected ? s.picked : ""].join(" ")}
            onClick={() => {
              it.onSelect();
              onPicked?.();
            }}
          >
            {it.icon}
            <span className={s.itemLabel}>{it.label}</span>
            {it.hint && <span className={s.hint}>{it.hint}</span>}
            {it.selected && <Check className={s.tick} aria-hidden />}
          </button>
        </li>
      ))}
    </ul>
  );
}

export function MenuDivider() {
  return <hr className={s.divider} />;
}
