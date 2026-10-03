"use client";

import { Check, ChevronDown, KeyRound, LogOut, Menu as MenuIcon, PanelLeftClose, PanelLeft, Search, Settings, ShieldCheck, UserPlus, Users } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { BT03_SEED } from "@/mock/assess";
import { COURSE_1, ROLE_LABEL, SUBJECT, type Role } from "@/mock/core";
import { reviewPending, ticketStats } from "@/mock/derive";
import { KEYS, type Bt03State, type Ticket } from "@/mock/state";
import { mergeTickets } from "@/mock/support";
import { ALL_COURSES } from "@/shared/session/cookies";
import { useSession } from "@/shared/session/session";
import { useEnsureClock, useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { ButtonLink, Drawer, EmptyState, Kbd, MenuDivider, MenuList, Page, PageHeader, Popover } from "@/shared/ui";
import { CommandPalette } from "@/shared/ui/CommandPalette";
import { useRealNotifications } from "@/shared/session/notifications";
import { MOBILE_PRIMARY, canOpen, mockBackend, navFor, needsCourse, whoCanOpen, type NavItem } from "./nav";
import { NotificationPopover } from "./NotificationPopover";
import s from "./AppShell.module.css";

/** Mốc giả lập của hai thông báo nền cho Quản trị viên (không có sự kiện nào sinh ra chúng). */
/** Tên người hiển thị: GV có học hàm ("TS. Lê Thu Hà"), TA / Admin / SV chỉ có tên. */
function displayName(role: Role, name: string, title?: string) {
  return role === "teacher" && title ? `${title} ${name}` : name;
}

const NARROW = "(max-width: 719px)";
const MID = "(max-width: 1099px)";
function useMedia(query: string) {
  return useSyncExternalStore(
    (cb) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", cb);
      return () => mql.removeEventListener("change", cb);
    },
    () => window.matchMedia(query).matches,
    () => false,
  );
}

// Lựa chọn thu gọn thanh bên (chỉ ≥ 1100 px) nhớ ở `ep:ui:sidebar` = "collapsed" | "expanded" — không phải token (SRS 5.3).
const SIDEBAR_KEY = "ep:ui:sidebar";
const sidebarListeners = new Set<() => void>();
function useSidebarCollapsed(): [boolean, (v: boolean) => void] {
  const v = useSyncExternalStore(
    (cb) => {
      sidebarListeners.add(cb);
      window.addEventListener("storage", cb);
      return () => {
        sidebarListeners.delete(cb);
        window.removeEventListener("storage", cb);
      };
    },
    () => {
      try {
        return localStorage.getItem(SIDEBAR_KEY) === "collapsed";
      } catch {
        return false;
      }
    },
    () => false,
  );
  const set = (next: boolean) => {
    try {
      localStorage.setItem(SIDEBAR_KEY, next ? "collapsed" : "expanded");
    } catch { /* bộ nhớ bị chặn: chỉ mất việc nhớ */ }
    sidebarListeners.forEach((f) => f());
  };
  return [v, set];
}

const MOCK_SCREENS_OFF = process.env.NEXT_PUBLIC_MOCK_SCREENS === "0";

export function AppShell({ children }: { children: React.ReactNode }) {
  const { role, user, course, courses, isAll, hasCourse, realCourses, realCourseId, setCourse, identity, logout } = useSession();
  const pathname = usePathname();
  const router = useRouter();
  const groups = useMemo(() => navFor(role, hasCourse), [role, hasCourse]);
  const flat = useMemo(() => groups.flatMap((g) => g.items), [groups]);
  const [prefCollapsed, setPrefCollapsed] = useSidebarCollapsed();
  const mid = useMedia(MID);
  // 720–1099 px luôn thu gọn (72 px); ≥ 1100 px theo lựa chọn đã nhớ (216 px mặc định)
  const collapsed = mid || prefCollapsed;
  const [palette, setPalette] = useState(false);
  // mỗi lần MỞ tăng `paletteTick` để hộp thoại gốc mở lại kể cả khi sự kiện `close` của lần đóng trước chưa kịp cập nhật `palette`
  const [paletteTick, setPaletteTick] = useState(0);
  const openPalette = () => {
    setPalette(true);
    setPaletteTick((n) => n + 1);
  };
  const [more, setMore] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const typing = e.target instanceof HTMLElement && e.target.closest("input, textarea, select, [contenteditable]");
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typing)) {
        e.preventDefault();
        openPalette();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const isActive = (href: string) => (href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`));
  const mobilePrimary = flat.filter((i) => MOBILE_PRIMARY[role].includes(i.href));
  const mobileMore = flat.filter((i) => !MOBILE_PRIMARY[role].includes(i.href));
  // Phiên thật: bộ chọn liệt kê lớp THẬT (`/me/courses`); phiên mô phỏng giữ danh sách mô phỏng.
  const realPicked = realCourses?.find((c) => c.id === realCourseId);
  const courseTitle = !hasCourse ? "Chưa có lớp" : isAll ? "Tất cả lớp của tôi" : realPicked ? `${realPicked.class_code} · ${realPicked.name}` : `${course.code} · ${course.name}`;
  const canAll = realCourses ? role !== "student" && realCourses.length > 1 : role === "teacher";
  const personName = displayName(role, user.name, user.title);
  const scope = useMemo(() => (isAll ? courses.map((c) => c.id) : hasCourse ? [course.id] : []), [isAll, courses, hasCourse, course.id]);

  // Đồng hồ giả lập: khung app lưu mốc t0 một lần, mọi chuỗi thời gian tương đối đọc `now` này (SRS 4.8 N6).
  useEnsureClock();
  const now = useSimNow();

  // N9: badge tính từ dữ liệu, giảm ngay khi hành động xảy ra; không bao giờ ghi số ở nav.ts.
  const [storedTickets] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [approvedIds] = useDemoSlice<string[]>("grading.approved", []);
  const badges = {
    inbox: ticketStats(mergeTickets(storedTickets), scope, now).open,
    grading: scope.includes(COURSE_1) ? reviewPending(bt03.status, approvedIds).length : 0,
  };

  // Chuông đọc GET /notifications (làm mới 30 s / khi tab lấy lại focus).
  const real = useRealNotifications(true);

  // Thanh bên và thanh dưới loại trừ nhau: chỉ gắn một bộ badge vào DOM ở mỗi bề rộng.
  const narrow = useMedia(NARROW);

  return (
    <div className={[s.shell, collapsed ? s.collapsed : ""].join(" ")}>
      <a href="#main" className={s.skip}>
        Bỏ qua điều hướng
      </a>

      <header className={s.topbar} data-part="topbar">
        <Link href="/" className={s.mobileBrand} data-part="brand" aria-label="EduPilot — Hôm nay">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/brand/logo-edupilot-mark.svg" alt="" width={28} height={28} />
        </Link>

        {role === "admin" ? (
          <p className={s.scope}>Toàn hệ thống · {SUBJECT.code}</p>
        ) : (
          <Popover
            align="start"
            width={300}
            label="Chọn lớp"
            trigger={(p) => (
              <button
                type="button"
                className={s.courseBtn}
                onClick={p.toggle}
                aria-expanded={p["aria-expanded"]}
                aria-haspopup="true"
                aria-label={`Chọn lớp, đang xem ${courseTitle}`}
                title={courseTitle}
              >
                {role === "student" && !hasCourse ? (
                  <span className={s.courseCode}>Chưa có lớp</span>
                ) : isAll ? (
                  <span className={s.courseCode}>Tất cả lớp của tôi</span>
                ) : (
                  <>
                    <span className={s.courseCode}>{realPicked?.class_code ?? course.code}</span>
                    <span className={s.courseName} title={courseTitle}>
                      · {realPicked?.name ?? course.name}
                    </span>
                  </>
                )}
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                {narrow && <p className={s.courseCurrent}>{role === "student" && !hasCourse ? "Chưa có lớp" : isAll ? "Tất cả lớp của tôi" : courseTitle}</p>}
                <p className={s.panelLabel}>{role === "student" ? "Lớp của bạn" : "Lớp bạn phụ trách"}</p>
                <MenuList
                  onPicked={close}
                  items={[
                    ...(realCourses
                      ? realCourses.map((c) => ({
                          label: (
                            <span className={s.courseOpt} title={`${c.class_code} · ${c.name}`}>
                              <span>{c.class_code} · {c.name}</span>
                              <span className={s.courseOptMeta}>{c.semester}</span>
                            </span>
                          ),
                          icon: !isAll && c.id === realCourseId ? <Check aria-hidden /> : <span className={s.iconGap} />,
                          onSelect: () => setCourse(c.id),
                        }))
                      : courses.map((c) => ({
                          label: (
                            <span className={s.courseOpt}>
                              <span>{c.label}</span>
                              <span className={s.courseOptMeta}>
                                {c.schedule}
                                {c.state === "new" ? " · lớp mới nhận" : ""}
                              </span>
                            </span>
                          ),
                          icon: !isAll && c.id === course.id ? <Check aria-hidden /> : <span className={s.iconGap} />,
                          onSelect: () => setCourse(c.id),
                        }))),
                    ...(canAll
                      ? [
                          {
                            label: "Tất cả lớp của tôi",
                            icon: isAll ? <Check aria-hidden /> : <span className={s.iconGap} />,
                            onSelect: () => setCourse(ALL_COURSES),
                          },
                        ]
                      : []),
                  ]}
                />
                <MenuDivider />
                <MenuList
                  onPicked={close}
                  items={
                    role === "student"
                      ? [{ label: "Tham gia lớp bằng mã", icon: <KeyRound aria-hidden />, onSelect: () => router.push("/join") }]
                      : [
                          { label: "Quản lý lớp này", icon: <Users aria-hidden />, onSelect: () => router.push("/class/members") },
                          ...(role === "teacher"
                            ? [{ label: "Mời trợ giảng", icon: <UserPlus aria-hidden />, onSelect: () => router.push("/class/members?tab=staff") }]
                            : []),
                        ]
                  }
                />
              </div>
            )}
          </Popover>
        )}

        <div className={s.topEnd}>
          <button type="button" className={s.search} onClick={openPalette} aria-label="Tìm nhanh hoặc đi đến">
            <Search aria-hidden />
            <span className={s.searchText}>Tìm nhanh hoặc đi đến…</span>
            <span className={s.searchKey}>
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </span>
          </button>

          <NotificationPopover
            items={real.items}
            unread={real.unread}
            failed={real.failed}
            onRead={(id) => real.markRead(id)}
          />

          <Popover
            width={260}
            label="Tài khoản"
            trigger={(p) => (
              <button type="button" className={s.profile} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true" aria-label={`Tài khoản: ${personName}`}>
                <span className={s.initials} aria-hidden>
                  {user.name
                    .split(" ")
                    .slice(-1)[0]
                    .slice(0, 1)}
                </span>
                <span className={s.profileRole}>{personName}</span>
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                <div className={s.who}>
                  <p className={s.whoName}>{personName}</p>
                  <p className={s.whoRole}>{ROLE_LABEL[role]}</p>
                  <p className={s.whoMail}>{identity ? identity.email || identity.sub : user.email}</p>
                </div>
                <MenuDivider />
                <MenuList
                  onPicked={close}
                  items={[
                    { label: "Tài khoản và bảo mật", icon: <ShieldCheck aria-hidden />, onSelect: () => router.push("/settings") },
                    ...(role === "teacher" || role === "admin"
                      ? [{ label: "Cài đặt hệ thống", icon: <Settings aria-hidden />, onSelect: () => router.push("/settings/llm") }]
                      : []),
                    {
                      label: "Đăng xuất",
                      icon: <LogOut aria-hidden />,
                      onSelect: () => logout(), // thu hồi phiên ở máy chủ, xoá bộ nhớ, về /login
                    },
                  ]}
                />
              </div>
            )}
          </Popover>
        </div>
      </header>

      {!narrow && (
      <aside className={s.sidebar} data-part="sidebar" data-collapsed={collapsed || undefined}>
        <Link href="/" className={s.brand} data-part="brand" aria-label="EduPilot — Hôm nay">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={collapsed ? "/brand/logo-edupilot-mark.svg" : "/brand/logo-edupilot.svg"} alt="" height={32} className={collapsed ? s.logoMark : s.logo} />
        </Link>

        <nav className={s.nav} id="sidebar-nav" aria-label="Điều hướng chính">
          {groups.map((g, gi) => (
            <div key={gi} className={s.group}>
              {g.label && !collapsed && <p className={s.groupLabel}>{g.label}</p>}
              <ul>
                {g.items.map((item) => (
                  <li key={item.href}>
                    <NavLink item={item} active={isActive(item.href)} collapsed={collapsed} badge={item.badgeKey ? badges[item.badgeKey] : 0} />
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>

        <div className={s.sideFoot}>
          {!collapsed && <p className={s.demo}>Bản mô phỏng · dữ liệu giả</p>}
          {!mid && (
            <button type="button" className={s.collapse} onClick={() => setPrefCollapsed(!prefCollapsed)} aria-expanded={!prefCollapsed} aria-controls="sidebar-nav" aria-label={collapsed ? "Mở rộng thanh bên" : "Thu gọn thanh bên"}>
              {collapsed ? <PanelLeft aria-hidden /> : <PanelLeftClose aria-hidden />}
              {!collapsed && <span>Thu gọn</span>}
            </button>
          )}
        </div>
      </aside>
      )}

      <p className={s.demoMobile}>Bản mô phỏng · dữ liệu giả</p>

      <main id="main" className={s.main} tabIndex={-1}>
        {!canOpen(role, pathname) ? (
          <Page>
            <PageHeader
              title="Bạn không có quyền xem màn này"
              description={whoCanOpen(pathname)}
              actions={<ButtonLink href="/">Về Hôm nay</ButtonLink>}
            />
          </Page>
        ) : !hasCourse && needsCourse(role, pathname) ? (
          <NoCourse />
        ) : MOCK_SCREENS_OFF && mockBackend(pathname) ? (
          <NoBackend phase={mockBackend(pathname)!} />
        ) : (
          children
        )}
      </main>

      {narrow && (
        <>
          <nav className={s.bottomNav} data-part="bottom-nav" aria-label="Điều hướng chính">
            {mobilePrimary.map((item) => (
              <BottomLink key={item.href} item={item} active={isActive(item.href)} badge={item.badgeKey ? badges[item.badgeKey] : 0} />
            ))}
            {mobileMore.length > 0 && (
              <button type="button" className={[s.bottomItem, mobileMore.some((i) => isActive(i.href)) ? s.bottomActive : ""].join(" ")} onClick={() => setMore(true)}>
                <span className={s.bottomIcon}>
                  <MenuIcon aria-hidden />
                  {/* badge của mục nằm trong "Thêm" vẫn phải thấy được ở thanh dưới (SRS 4.8 N9) */}
                  {mobileMore.map((i) =>
                    i.badgeKey && badges[i.badgeKey] > 0 ? (
                      <span key={i.href} className={s.bottomBadge} data-part={`nav-badge-${i.badgeKey}`} title={i.label}>
                        {badges[i.badgeKey]}
                      </span>
                    ) : null,
                  )}
                </span>
                <span>Thêm</span>
              </button>
            )}
          </nav>

          <Drawer open={more} onClose={() => setMore(false)} title="Thêm" sheet>
            <ul className={s.moreList}>
              {mobileMore.map((item) => (
                <li key={item.href}>
                  <Link href={item.href} className={s.moreItem} onClick={() => setMore(false)} aria-current={isActive(item.href) ? "page" : undefined}>
                    <item.icon aria-hidden />
                    <span>{item.label}</span>
                    {item.badgeKey && badges[item.badgeKey] > 0 ? <span className={s.badge}>{badges[item.badgeKey]}</span> : null}
                  </Link>
                </li>
              ))}
            </ul>
          </Drawer>
        </>
      )}

      <CommandPalette open={palette} tick={paletteTick} onClose={() => setPalette(false)} items={flat} />
    </div>
  );
}

function NavLink({ item, active, collapsed, badge }: { item: NavItem; active: boolean; collapsed: boolean; badge: number }) {
  const Icon = item.icon;
  return (
    <Link href={item.href} className={s.navItem} aria-current={active ? "page" : undefined} aria-label={collapsed ? item.label : undefined} title={collapsed ? item.label : undefined}>
      <Icon aria-hidden />
      <span className={s.navLabel}>{item.label}</span>
      {item.badgeKey && badge > 0 ? (
        <span className={s.badge} data-part={`nav-badge-${item.badgeKey}`}>
          {badge}
        </span>
      ) : null}
    </Link>
  );
}

function BottomLink({ item, active, badge }: { item: NavItem; active: boolean; badge: number }) {
  const Icon = item.icon;
  return (
    <Link href={item.href} className={[s.bottomItem, active ? s.bottomActive : ""].join(" ")} aria-current={active ? "page" : undefined}>
      <span className={s.bottomIcon}>
        <Icon aria-hidden />
        {item.badgeKey && badge > 0 ? (
          <span className={s.bottomBadge} data-part={`nav-badge-${item.badgeKey}`}>
            {badge}
          </span>
        ) : null}
      </span>
      <span>{item.short ?? item.label}</span>
    </Link>
  );
}

/**
 * SV chưa vào lớp mở route cần lớp (SRS 4.3.2): một màn duy nhất trong khung app, URL giữ nguyên,
 * KHÔNG phải màn chặn quyền và không có dòng dữ liệu nào của lớp.
 */
function NoCourse() {
  return (
    <Page>
      <PageHeader
        title="Bạn chưa vào lớp nào"
        description="Nhập mã tham gia do giảng viên cung cấp để dùng tính năng này."
        actions={<ButtonLink href="/join" variant="primary">Tham gia lớp bằng mã</ButtonLink>}
      />
    </Page>
  );
}

/** Route chưa có backend (build `MOCK_SCREENS=0`): nói thật tiến độ, một nút kế tiếp; mục nav vẫn còn (US-PU-04 AC11). */
function NoBackend({ phase }: { phase: { phase: string; name: string } }) {
  return (
    <Page>
      <div data-part="empty-no-backend">
        <EmptyState title="Màn này chưa sẵn sàng" action={<ButtonLink href="/">Về Hôm nay</ButtonLink>}>
          Tính năng này đang được xây ở giai đoạn {phase.phase} — {phase.name}. Khi xong, bạn sẽ dùng nó ngay tại đây.
        </EmptyState>
      </div>
    </Page>
  );
}
