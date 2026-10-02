"use client";

import { MoreHorizontal } from "lucide-react";
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
}: {
  trigger: (props: { open: boolean; toggle: () => void; "aria-expanded": boolean; "aria-haspopup": "true" }) => ReactNode;
  children: ReactNode | ((close: () => void) => ReactNode);
  align?: "start" | "end";
  width?: number;
  label?: string;
}) {
  const [open, setOpen] = useState(false);
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

export type MenuItem = { label: ReactNode; onSelect: () => void; danger?: boolean; icon?: ReactNode; hint?: ReactNode };

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
      {(close) => <MenuList items={items} onPicked={close} />}
    </Popover>
  );
}

export function MenuList({ items, onPicked }: { items: MenuItem[]; onPicked?: () => void }) {
  return (
    <ul className={s.menu} role="menu">
      {items.map((it, i) => (
        <li key={i} role="none">
          <button
            type="button"
            role="menuitem"
            className={[s.item, it.danger ? s.danger : ""].join(" ")}
            onClick={() => {
              it.onSelect();
              onPicked?.();
            }}
          >
            {it.icon}
            <span className={s.itemLabel}>{it.label}</span>
            {it.hint && <span className={s.hint}>{it.hint}</span>}
          </button>
        </li>
      ))}
    </ul>
  );
}

export function MenuDivider() {
  return <hr className={s.divider} />;
}
