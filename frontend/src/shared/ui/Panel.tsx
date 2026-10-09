import type { ReactNode } from "react";
import s from "./Panel.module.css";

// Server Component thuần CSS (0 byte JS): không đánh dấu client, không context. Chặn `Panel` lồng `Panel` bằng phép 22 của scripts/ui-antipatterns.sh
// (cùng tệp) và kiểm DOM `[data-ep-panel] [data-ep-panel]` = 0 ở e2e/panels.spec.ts (qua ranh giới thành phần). `className` KHÔNG cho truyền: không ai tự chế khung.
type PanelProps = {
  as?: "div" | "section";
  padding?: "md" | "lg" | "none";
  "aria-label"?: string;
  children: ReactNode;
  className?: never;
};

/** Một vùng làm việc = một Panel trên nền `--ep-canvas`. Tiêu đề vùng (`h2`) nằm NGOÀI panel; nhóm bên trong tách bằng `PanelSection`. */
export function Panel({ as: Tag = "div", padding = "md", children, ...rest }: PanelProps) {
  return (
    <Tag className={[s.panel, padding === "lg" ? s.lg : padding === "none" ? s.none : ""].join(" ").trim()} data-ep-panel aria-label={rest["aria-label"]}>
      {children}
    </Tag>
  );
}

type PanelSectionProps = {
  title?: ReactNode;
  action?: ReactNode;
  tone?: "default" | "strong";
  children: ReactNode;
};

/**
 * Nhóm trong panel: khoảng trắng + một đường kẻ 1 px giữa hai nhóm liền nhau, không viền / bóng / bo góc riêng.
 * `tone="strong"` là ô nhấn (≤ 3 mỗi panel; chỉ cho số liệu / trạng thái quan trọng). Không chứa `Panel`.
 */
export function PanelSection({ title, action, tone = "default", children }: PanelSectionProps) {
  return (
    <div className={[s.section, tone === "strong" ? s.strong : ""].join(" ").trim()} data-ep-panel-section data-tone={tone === "strong" ? "strong" : undefined}>
      {(title || action) && (
        <div className={s.head}>
          {title && <h3 className="ep-item-title">{title}</h3>}
          {action && <div className={s.action}>{action}</div>}
        </div>
      )}
      {children}
    </div>
  );
}
