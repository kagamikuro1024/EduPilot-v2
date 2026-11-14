"use client";

import Link from "next/link";
import { ArrowLeft, ChevronDown } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useRef, type ReactNode } from "react";
import { consumeRedThread } from "@/shared/motion/redThread";
import { Button, ButtonLink, type ButtonVariant } from "./Button";
import { MenuList, OverflowMenu, Popover, type MenuItem } from "./Menu";
import { Panel } from "./Panel";
import s from "./Layout.module.css";

/**
 * Khung nội dung của một route (DESIGN.md §23.1): đúng hai kiểu. `work` = hết vùng nội dung (bảng, lưới, chia đôi), `read` = một cột 800 px căn giữa.
 * Route chỉ chọn kiểu; độ rộng, đệm, căn lề do primitive quyết. Không có `className`, không có prop độ rộng.
 */
export function Page({ kind, children }: { kind: "work" | "read"; children: ReactNode; className?: never }) {
  return (
    <div className={[s.page, s[kind]].join(" ")} data-ep-page={kind}>
      {children}
    </div>
  );
}

/** Một hành động ở đầu trang, truyền như DỮ LIỆU (không phải JSX) để primitive gom nút phụ vào menu «Thêm» khi hẹp (DESIGN.md §23.2). */
export type HeaderAction = {
  label: string;
  icon?: ReactNode;
  onClick?: () => void;
  href?: string;
  disabled?: boolean;
  /** lý do khoá: hiện trên chính nút khi `disabled` */
  reason?: string;
  busy?: boolean;
  /** nút mở menu (vd. «Tạo câu hỏi» ▾): các lệnh trong menu, hoặc hàm dựng menu nhiều cấp */
  menu?: MenuItem[] | ((close: () => void) => ReactNode);
  /** nút bật / tắt một khối ngay dưới (aria-expanded) */
  expanded?: boolean;
  /** lệnh nguy hiểm trong menu «Thêm» / ⋯ */
  danger?: boolean;
  /** chỉ hiện từ 720 px (màn đã có thanh hành động dính riêng ở điện thoại) */
  desktopOnly?: boolean;
};

function ActionButton({ a, variant }: { a: HeaderAction; variant: ButtonVariant }) {
  if (a.href && !a.disabled) {
    return (
      <ButtonLink href={a.href} variant={variant} icon={a.icon}>
        {a.label}
      </ButtonLink>
    );
  }
  if (a.menu) {
    const items = a.menu;
    return (
      <Popover
        label={a.label}
        width={240}
        trigger={(p) => (
          <Button variant={variant} icon={a.icon} iconEnd={<ChevronDown aria-hidden />} disabled={a.disabled} disabledReason={a.reason} loading={a.busy} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
            {a.label}
          </Button>
        )}
      >
        {(close) => (typeof items === "function" ? items(close) : <MenuList items={items} onPicked={close} autoFocus />)}
      </Popover>
    );
  }
  return (
    <Button variant={variant} icon={a.icon} disabled={a.disabled} disabledReason={a.reason} loading={a.busy} onClick={a.onClick} aria-expanded={a.expanded}>
      {a.label}
    </Button>
  );
}

/**
 * Tiêu đề route (DESIGN.md §23.2): `h1` duy nhất, tối đa một dòng meta (dữ liệu, không câu, không lặp lớp đang chọn), hành động bên phải CÙNG hàng tiêu đề:
 * một `primary` + tối đa hai `secondary` + `more` (menu ⋯). Dưới 720 px hai nút phụ gom vào một nút «Thêm». Không có mô tả dưới tiêu đề.
 * Nơi đến của Red Thread Transition: nếu người dùng vừa chọn việc ở "Hôm nay", một vạch đỏ chạy từ hàng đã chọn tới dưới tiêu đề này.
 */
export function PageHeader({
  title,
  back,
  meta,
  primary,
  secondary,
  more,
}: {
  title: ReactNode;
  back?: { href: string; label: string };
  meta?: readonly ReactNode[];
  primary?: HeaderAction;
  secondary?: readonly [HeaderAction?, HeaderAction?];
  more?: readonly HeaderAction[];
}) {
  const titleRef = useRef<HTMLHeadingElement>(null);
  const router = useRouter();
  useEffect(() => {
    if (titleRef.current) consumeRedThread(titleRef.current);
  }, []);
  const second = (secondary ?? []).filter((a): a is HeaderAction => !!a);
  const extra = more ?? [];
  const toItem = (a: HeaderAction): MenuItem => ({ label: a.label, icon: a.icon, danger: a.danger, disabled: a.disabled, onSelect: () => (a.href ? router.push(a.href) : a.onClick?.()) });
  // dưới 720 px: nút chính đứng đầu hàng; không có nút chính thì nút phụ đầu tiên giữ nguyên (không giấu hành động duy nhất của trang), còn lại gom vào «Thêm»
  const lead = primary ? null : (second[0] ?? null);
  const folded = [...second.filter((a) => a !== lead), ...extra].map(toItem);
  const metaParts = (meta ?? []).filter((m) => m !== null && m !== undefined && m !== false && m !== "");
  const hasActions = !!primary || second.length > 0 || extra.length > 0;
  return (
    <div className={s.header} data-ep-page-header>
      {back && (
        <Link href={back.href} className={s.back}>
          <ArrowLeft aria-hidden />
          {back.label}
        </Link>
      )}
      <div className={s.headRow}>
        <h1 ref={titleRef} className="ep-page-title" data-part="page-title">
          {title}
        </h1>
        {hasActions && (
          <div className={s.actions} data-ep-header-actions>
            {second.map((a) => (
              <span key={a.label} className={a === lead ? s.lead : s.wide}>
                <ActionButton a={a} variant="secondary" />
              </span>
            ))}
            {extra.length > 0 && (
              <span className={s.wide}>
                <OverflowMenu items={extra.map(toItem)} />
              </span>
            )}
            {folded.length > 0 && (
              <span className={s.narrow}>
                <Popover
                  label="Thêm"
                  width={240}
                  trigger={(p) => (
                    <Button variant="secondary" iconEnd={<ChevronDown aria-hidden />} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
                      Thêm
                    </Button>
                  )}
                >
                  {(close) => <MenuList items={folded} onPicked={close} autoFocus />}
                </Popover>
              </span>
            )}
            {primary && (
              <span className={[s.primary, primary.desktopOnly ? s.wide : ""].join(" ")}>
                <ActionButton a={primary} variant="primary" />
              </span>
            )}
          </div>
        )}
      </div>
      {metaParts.length > 0 && (
        <div className={s.meta} data-ep-header-meta>
          {metaParts.map((m, i) => (
            <span key={i}>
              {i > 0 && <span aria-hidden className={s.dot}> · </span>}
              {m}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Nhóm nội dung (DESIGN.md §23.3): tiêu đề `h2` NGOÀI panel (một dòng, không dòng phụ) + tối đa một hành động dạng chữ ở phải;
 * nội dung luôn nằm trong MỘT `Panel`. Trang chỉ có một vùng thì không dùng `Section` (`h1` đã gọi tên vùng).
 */
export function Section({
  title,
  action,
  children,
  id,
  part,
}: {
  title?: ReactNode;
  action?: ReactNode;
  children: ReactNode;
  id?: string;
  /** móc đo dev: `data-part` của phần (vd. `settings-section`) */
  part?: string;
}) {
  return (
    <section className={s.section} id={id} data-part={part} data-ep-section aria-labelledby={title && id ? `${id}-title` : undefined}>
      {(title || action) && (
        <div className={s.sectionHead} data-ep-section-head>
          {title && (
            <h2 className="ep-section-title" id={id ? `${id}-title` : undefined} data-ep-section-title>
              {title}
            </h2>
          )}
          {action && <div className={s.sectionAction}>{action}</div>}
        </div>
      )}
      {children ? <Panel>{children}</Panel> : null}
    </section>
  );
}

/** Thanh công cụ gọn phía trên bảng/danh sách: tìm kiếm, bộ lọc, hành động phụ. */
export function Toolbar({ children, end, alignEnd }: { children: ReactNode; end?: ReactNode; /** căn đáy: ô chọn có nhãn và chip lọc nằm cùng một đường đáy */ alignEnd?: boolean }) {
  return (
    <div className={s.toolbar} data-ep-toolbar>
      <div className={s.toolbarStart} data-align={alignEnd ? "end" : undefined}>{children}</div>
      {end && <div className={s.toolbarEnd}>{end}</div>}
    </div>
  );
}

/**
 * Hai vùng cạnh nhau — chỗ DUY NHẤT trong app được đặt chiều cao theo màn (DESIGN.md §23.3).
 * `main-aside`: chính + bối cảnh 340 px, hai panel thẳng đỉnh, xếp chồng dưới 1100 px.
 * `half`: hai cột bằng nhau, hai panel.
 * `list-detail`: MỘT panel sát mép, danh sách 380 px + chi tiết; `fill` = cao đúng phần còn lại của màn, hai ngăn cuộn riêng (chỉ khi `filled`: danh sách có dòng;
 * rỗng / lỗi thì panel thấp theo nội dung). Dưới 1100 px chỉ hiện một ngăn: danh sách, hoặc chi tiết khi `detailOpen`.
 */
export function Split({
  main,
  aside,
  ratio = "main-aside",
  fill = false,
  detailOpen = false,
  listCollapsed = false,
  only,
}: {
  main: ReactNode;
  aside: ReactNode;
  ratio?: "main-aside" | "half" | "list-detail";
  /** `list-detail`: cao hết màn (chỉ khi có dòng) */
  fill?: boolean;
  /** `list-detail` dưới 1100 px: hiện ngăn chi tiết thay cho danh sách */
  detailOpen?: boolean;
  /** `list-detail`: người dùng thu gọn danh sách (chat); chi tiết chiếm cả bề rộng */
  listCollapsed?: boolean;
  /** `main-aside` / `half` dưới 1100 px: chỉ hiện một ngăn (màn có tab «Bài làm / Chấm») */
  only?: "main" | "aside";
}) {
  if (ratio === "list-detail") {
    return (
      <Panel>
        <div className={s.listDetail} data-ep-split="list-detail" data-ep-split-fill={fill ? "" : undefined} data-detail-open={detailOpen ? "" : undefined} data-list-collapsed={listCollapsed ? "" : undefined}>
          <div className={s.ldList}>{main}</div>
          <div className={s.ldDetail}>{aside}</div>
        </div>
      </Panel>
    );
  }
  return (
    <div className={[s.split, ratio === "half" ? s.half : ""].join(" ")} data-ep-split={ratio} data-ep-split-fill={fill ? "" : undefined} data-only={only}>
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
