"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { markRedThread } from "@/shared/motion/redThread";
import s from "./ActionList.module.css";

export type Tone = "red" | "amber" | "green" | "blue" | "neutral";

/** Danh sách việc: hàng ngăn bằng đường kẻ, không card (DESIGN.md §10.3). */
export function ActionList({ children, label }: { children: ReactNode; label?: string }) {
  return (
    <ul className={s.list} aria-label={label}>
      {children}
    </ul>
  );
}

/**
 * Một việc: chấm trạng thái, tiêu đề, một dòng bối cảnh, siêu dữ liệu, một hành động.
 * `href` làm cả hàng bấm được; `redThread` bật chuyển động sợi chỉ đỏ khi chọn.
 */
export function ActionRow({
  tone = "neutral",
  title,
  context,
  meta,
  action,
  href,
  redThread,
  selected,
  onSelect,
  lead,
  data,
}: {
  tone?: Tone;
  title: ReactNode;
  context?: ReactNode;
  meta?: ReactNode;
  action?: ReactNode;
  href?: string;
  redThread?: boolean;
  selected?: boolean;
  onSelect?: () => void;
  lead?: ReactNode;
  /** thuộc tính `data-*` cho hàng (móc đo / kiểm), ví dụ `{ "data-part": "thread-row" }` */
  data?: Record<`data-${string}`, string>;
}) {
  const body = (
    <>
      {lead ?? <span className={[s.dot, s[tone]].join(" ")} aria-hidden />}
      <span className={s.text}>
        <span className={s.title}>{title}</span>
        {context && <span className={s.context}>{context}</span>}
        {meta && <span className={s.meta}>{meta}</span>}
      </span>
    </>
  );
  return (
    <li className={[s.row, selected ? s.selected : ""].join(" ")} {...data}>
      {href ? (
        <Link href={href} className={s.main} onClick={(e) => redThread && markRedThread(e.currentTarget)}>
          {body}
        </Link>
      ) : onSelect ? (
        <button type="button" className={s.main} onClick={onSelect} aria-pressed={selected}>
          {body}
        </button>
      ) : (
        <div className={s.main}>{body}</div>
      )}
      {action && <span className={s.action}>{action}</span>}
    </li>
  );
}
