"use client";

import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";
import { consumeRedThread } from "@/shared/motion/redThread";
import { Panel } from "./Panel";
import s from "./Layout.module.css";

/** Khung nội dung của một route. `reading` 960px cho đọc/form; `wide` cho bảng; `full` cho màn chia đôi. */
export function Page({ width = "reading", children, className }: { width?: "reading" | "wide" | "full"; children: ReactNode; className?: string }) {
  return <div className={[s.page, s[width], className ?? ""].join(" ")}>{children}</div>;
}

/**
 * Tiêu đề route: một tiêu đề, tối đa một câu phụ, hành động chính bên phải (DESIGN.md §10.2).
 * Nơi đến của Red Thread Transition: nếu người dùng vừa chọn việc ở "Hôm nay", một vạch đỏ
 * chạy từ hàng đã chọn tới dưới tiêu đề này.
 */
export function PageHeader({
  title,
  description,
  actions,
  back,
  meta,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  back?: { href: string; label: string };
  meta?: ReactNode;
}) {
  const titleRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (titleRef.current) consumeRedThread(titleRef.current);
  }, []);
  return (
    <div className={s.header}>
      {back && (
        <Link href={back.href} className={s.back}>
          <ArrowLeft aria-hidden />
          {back.label}
        </Link>
      )}
      <div className={s.headRow}>
        <div className={s.headText}>
          <h1 ref={titleRef} className="ep-page-title" data-part="page-title">
            {title}
          </h1>
          {description && <p className={s.desc}>{description}</p>}
          {meta && <div className={s.meta}>{meta}</div>}
        </div>
        {actions && <div className={s.actions}>{actions}</div>}
      </div>
    </div>
  );
}

/** Nhóm nội dung lớn: tiêu đề + câu phụ + một hành động dạng chữ ở phải. Không có khung bao. */
export function Section({
  title,
  description,
  action,
  children,
  className,
  id,
  part,
  panel,
}: {
  title?: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
  id?: string;
  /** D59: đặt `children` trong MỘT `Panel` (tiêu đề vùng vẫn ngoài panel). `true` = padding md; `"none"` cho bảng sát mép. */
  panel?: boolean | "lg" | "none";
  /** móc đo dev: `data-part` của phần (vd. `settings-section`) */
  part?: string;
}) {
  return (
    <section className={[s.section, className ?? ""].join(" ")} id={id} data-part={part} aria-labelledby={title && id ? `${id}-title` : undefined}>
      {(title || action) && (
        <div className={s.sectionHead}>
          <div>
            {title && (
              <h2 className="ep-section-title" id={id ? `${id}-title` : undefined}>
                {title}
              </h2>
            )}
            {description && <p className={s.sectionDesc}>{description}</p>}
          </div>
          {action && <div className={s.sectionAction}>{action}</div>}
        </div>
      )}
      {panel ? <Panel padding={panel === true ? "md" : panel}>{children}</Panel> : children}
    </section>
  );
}

/** Thanh công cụ gọn phía trên bảng/danh sách: tìm kiếm, bộ lọc, hành động phụ. */
export function Toolbar({ children, end }: { children: ReactNode; end?: ReactNode }) {
  return (
    <div className={s.toolbar}>
      <div className={s.toolbarStart}>{children}</div>
      {end && <div className={s.toolbarEnd}>{end}</div>}
    </div>
  );
}

/** Hai cột: chính + bối cảnh (≥ 1100px), xếp chồng khi hẹp. */
export function Split({ main, aside, ratio = "wide-main" }: { main: ReactNode; aside: ReactNode; ratio?: "wide-main" | "half" }) {
  return (
    <div className={[s.split, ratio === "half" ? s.half : ""].join(" ")}>
      <div className={s.splitMain}>{main}</div>
      <aside className={s.splitAside}>{aside}</aside>
    </div>
  );
}

/** Danh sách cặp nhãn–giá trị dạng dòng, cho thông tin tĩnh. */
export function DefinitionList({ items }: { items: Array<{ term: ReactNode; value: ReactNode }> }) {
  return (
    <dl className={s.dl}>
      {items.map((it, i) => (
        <div key={i} className={s.dlRow}>
          <dt>{it.term}</dt>
          <dd>{it.value}</dd>
        </div>
      ))}
    </dl>
  );
}
