"use client";

import { Bell, Check, ChevronDown, KeyRound, LogOut, Menu as MenuIcon, PanelLeftClose, PanelLeft, Search, Settings, UserPlus, Users } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ROLE_LABEL, SUBJECT, TERM, type Role } from "@/mock/core";
import { clearDemoSession } from "@/shared/session/cookies";
import { useSession } from "@/shared/session/session";
import { ButtonLink, Drawer, EmptyState, Kbd, MenuDivider, MenuList, Popover } from "@/shared/ui";
import { CommandPalette } from "@/shared/ui/CommandPalette";
import { MOBILE_PRIMARY, canOpen, navFor, type NavItem } from "./nav";
import s from "./AppShell.module.css";

const NOTIFICATIONS: Record<Role, Array<{ title: string; meta: string; href: string; unread?: boolean }>> = {
  student: [
    { title: "Giảng viên đã trả lời câu hỏi về điểm cộng của bạn", meta: "Hộp thư · 12 phút trước", href: "/chat", unread: true },
    { title: "Câu trả lời trong thread “Vì sao ECB lộ mẫu?” đã được xác nhận", meta: "Threads · 1 giờ trước", href: "/threads/t-ecb", unread: true },
    { title: "Tài liệu mới: Tuần 5 — Mật mã khối và chế độ hoạt động", meta: "Thư viện · hôm qua", href: "/library" },
  ],
  ta: [
    { title: "2 câu hỏi mới cần xử lý", meta: "Hộp thư hỗ trợ · 26 phút trước", href: "/inbox", unread: true },
    { title: "4 bài chấm lệch giữa hai lượt", meta: "Chấm bài · 1 giờ trước", href: "/grading", unread: true },
  ],
  teacher: [
    { title: "Bạn được phân công dạy INT1006 2 — mã tham gia K7MQ2RD", meta: "Admin · hôm qua", href: "/class/members", unread: true },
    { title: "2 câu hỏi mới cần xử lý", meta: "Hộp thư hỗ trợ · 26 phút trước", href: "/inbox", unread: true },
    { title: "Quy chế INT1006 2 chưa có công thức điểm", meta: "Sổ điểm · hôm qua", href: "/gradebook/scheme" },
  ],
  admin: [
    { title: "Tỷ lệ dùng model dự phòng tăng lên 4,2%", meta: "Quan sát AI · 40 phút trước", href: "/observability", unread: true },
    { title: "Ngân sách LLM tháng đã dùng 61%", meta: "Cấu hình LLM · sáng nay", href: "/settings/llm" },
  ],
};

export function AppShell({ children }: { children: React.ReactNode }) {
  const { role, user, course, courses, setRole, setCourse } = useSession();
  const pathname = usePathname();
  const router = useRouter();
  const groups = useMemo(() => navFor(role), [role]);
  const flat = useMemo(() => groups.flatMap((g) => g.items), [groups]);
  const [collapsed, setCollapsed] = useState(false);
  const [palette, setPalette] = useState(false);
  const [more, setMore] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const typing = e.target instanceof HTMLElement && e.target.closest("input, textarea, select, [contenteditable]");
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typing)) {
        e.preventDefault();
        setPalette(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const isActive = (href: string) => (href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`));
  const mobilePrimary = flat.filter((i) => MOBILE_PRIMARY[role].includes(i.href));
  const mobileMore = flat.filter((i) => !MOBILE_PRIMARY[role].includes(i.href));
  const notes = NOTIFICATIONS[role];
  const unread = notes.filter((n) => n.unread).length;

  function switchRole(next: Role) {
    setRole(next);
    if (!canOpen(next, pathname)) router.push("/");
  }

  return (
    <div className={[s.shell, collapsed ? s.collapsed : ""].join(" ")}>
      <a href="#main" className={s.skip}>
        Bỏ qua điều hướng
      </a>

      <aside className={s.sidebar} aria-label="Điều hướng chính">
        <Link href="/" className={s.brand} aria-label="EduPilot — Hôm nay">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={collapsed ? "/brand/logo-edupilot-mark.svg" : "/brand/logo-edupilot.svg"} alt="" height={24} className={s.logo} />
        </Link>

        {role !== "admin" && !collapsed && (
          <div className={s.context}>
            <p className={s.contextName}>{course.name}</p>
            <p className={s.contextMeta}>
              {course.code} · {TERM}
            </p>
          </div>
        )}

        <nav className={s.nav}>
          {groups.map((g, gi) => (
            <div key={gi} className={s.group}>
              {g.label && !collapsed && <p className={s.groupLabel}>{g.label}</p>}
              <ul>
                {g.items.map((item) => (
                  <li key={item.href}>
                    <NavLink item={item} active={isActive(item.href)} collapsed={collapsed} />
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>

        <div className={s.sideFoot}>
          {!collapsed && <p className={s.demo}>Bản mô phỏng · dữ liệu giả</p>}
          <button type="button" className={s.collapse} onClick={() => setCollapsed((c) => !c)} aria-label={collapsed ? "Mở rộng thanh bên" : "Thu gọn thanh bên"}>
            {collapsed ? <PanelLeft aria-hidden /> : <PanelLeftClose aria-hidden />}
            {!collapsed && <span>Thu gọn</span>}
          </button>
        </div>
      </aside>

      <header className={s.topbar}>
        <Link href="/" className={s.mobileBrand} aria-label="EduPilot — Hôm nay">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/brand/logo-edupilot-mark.svg" alt="" width={24} height={24} />
        </Link>

        {role === "admin" ? (
          <p className={s.scope}>Toàn hệ thống · {SUBJECT.code}</p>
        ) : (
          <Popover
            align="start"
            width={300}
            label="Chọn lớp"
            trigger={(p) => (
              <button type="button" className={s.courseBtn} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
                <span className={s.courseCode}>{course.code}</span>
                <span className={s.courseName}>· {course.name}</span>
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                <p className={s.panelLabel}>{role === "student" ? "Lớp của bạn" : "Lớp bạn dạy"}</p>
                <MenuList
                  onPicked={close}
                  items={courses.map((c) => ({
                    label: (
                      <span className={s.courseOpt}>
                        <span>{c.code}</span>
                        <span className={s.courseOptMeta}>
                          {c.schedule}
                          {c.state === "new" ? " · lớp mới nhận" : ""}
                        </span>
                      </span>
                    ),
                    icon: c.id === course.id ? <Check aria-hidden /> : <span className={s.iconGap} />,
                    onSelect: () => setCourse(c.id),
                  }))}
                />
                <MenuDivider />
                <MenuList
                  onPicked={close}
                  items={
                    role === "student"
                      ? [{ label: "Tham gia lớp bằng mã", icon: <KeyRound aria-hidden />, onSelect: () => router.push("/join") }]
                      : role === "teacher"
                        ? [
                            { label: "Quản lý lớp này", icon: <Users aria-hidden />, onSelect: () => router.push("/class/members") },
                            { label: "Mời trợ giảng", icon: <UserPlus aria-hidden />, onSelect: () => router.push("/class/members?tab=staff") },
                          ]
                        : []
                  }
                />
              </div>
            )}
          </Popover>
        )}

        <div className={s.topEnd}>
          <button type="button" className={s.search} onClick={() => setPalette(true)}>
            <Search aria-hidden />
            <span className={s.searchText}>Tìm nhanh hoặc đi đến…</span>
            <span className={s.searchKey}>
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </span>
          </button>

          <Popover
            width={340}
            label="Thông báo"
            trigger={(p) => (
              <button type="button" className={s.iconBtn} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true" aria-label={unread ? `Thông báo, ${unread} chưa đọc` : "Thông báo"}>
                <Bell aria-hidden />
                {unread > 0 && <span className={s.unreadDot} aria-hidden />}
              </button>
            )}
          >
            {(close) => (
              <div className={s.notes}>
                <p className={s.panelLabel}>Thông báo</p>
                <ul>
                  {notes.map((n) => (
                    <li key={n.title}>
                      <Link href={n.href} className={s.note} onClick={close}>
                        <span className={[s.noteDot, n.unread ? s.noteUnread : ""].join(" ")} aria-hidden />
                        <span>
                          <span className={s.noteTitle}>{n.title}</span>
                          <span className={s.noteMeta}>{n.meta}</span>
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </Popover>

          <Popover
            width={260}
            label="Tài khoản"
            trigger={(p) => (
              <button type="button" className={s.profile} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
                <span className={s.initials} aria-hidden>
                  {user.name
                    .split(" ")
                    .slice(-1)[0]
                    .slice(0, 1)}
                </span>
                <span className={s.profileRole}>{ROLE_LABEL[role]}</span>
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                <div className={s.who}>
                  <p className={s.whoName}>
                    {user.title && role !== "admin" ? `${user.title} ` : ""}
                    {user.name}
                  </p>
                  <p className={s.whoMail}>{user.email}</p>
                </div>
                <MenuDivider />
                <p className={s.panelLabel}>Xem bản mô phỏng với vai trò</p>
                <MenuList
                  onPicked={close}
                  items={(["student", "ta", "teacher", "admin"] as Role[]).map((r) => ({
                    label: ROLE_LABEL[r],
                    icon: r === role ? <Check aria-hidden /> : <span className={s.iconGap} />,
                    onSelect: () => switchRole(r),
                  }))}
                />
                <MenuDivider />
                <MenuList
                  onPicked={close}
                  items={[
                    ...(role === "teacher" || role === "admin"
                      ? [{ label: "Cài đặt hệ thống", icon: <Settings aria-hidden />, onSelect: () => router.push("/settings/llm") }]
                      : []),
                    {
                      label: "Đăng xuất",
                      icon: <LogOut aria-hidden />,
                      onSelect: () => {
                        clearDemoSession();
                        router.push("/login");
                      },
                    },
                  ]}
                />
              </div>
            )}
          </Popover>
        </div>
      </header>

      <main id="main" className={s.main} tabIndex={-1}>
        {canOpen(role, pathname) ? (
          children
        ) : (
          <div className={s.blocked}>
            <EmptyState
              title={`Vai trò ${ROLE_LABEL[role]} không mở được trang này`}
              action={<ButtonLink href="/">Về Hôm nay</ButtonLink>}
            >
              Mỗi vai trò chỉ thấy đúng phần việc của mình. Đổi vai trò ở góc trên bên phải để xem trang này trong bản mô phỏng.
            </EmptyState>
          </div>
        )}
      </main>

      <nav className={s.bottomNav} aria-label="Điều hướng chính">
        {mobilePrimary.map((item) => (
          <BottomLink key={item.href} item={item} active={isActive(item.href)} />
        ))}
        <button type="button" className={[s.bottomItem, mobileMore.some((i) => isActive(i.href)) ? s.bottomActive : ""].join(" ")} onClick={() => setMore(true)}>
          <MenuIcon aria-hidden />
          <span>Thêm</span>
        </button>
      </nav>

      <Drawer open={more} onClose={() => setMore(false)} title="Thêm">
        <ul className={s.moreList}>
          {mobileMore.map((item) => (
            <li key={item.href}>
              <Link href={item.href} className={s.moreItem} onClick={() => setMore(false)} aria-current={isActive(item.href) ? "page" : undefined}>
                <item.icon aria-hidden />
                <span>{item.label}</span>
                {item.badge ? <span className={s.badge}>{item.badge}</span> : null}
              </Link>
            </li>
          ))}
        </ul>
      </Drawer>

      <CommandPalette open={palette} onClose={() => setPalette(false)} items={flat} />
    </div>
  );
}

function NavLink({ item, active, collapsed }: { item: NavItem; active: boolean; collapsed: boolean }) {
  const Icon = item.icon;
  return (
    <Link href={item.href} className={s.navItem} aria-current={active ? "page" : undefined} title={collapsed ? item.label : undefined}>
      <Icon aria-hidden />
      <span className={s.navLabel}>{item.label}</span>
      {item.badge ? <span className={s.badge}>{item.badge}</span> : null}
    </Link>
  );
}

function BottomLink({ item, active }: { item: NavItem; active: boolean }) {
  const Icon = item.icon;
  return (
    <Link href={item.href} className={[s.bottomItem, active ? s.bottomActive : ""].join(" ")} aria-current={active ? "page" : undefined}>
      <span className={s.bottomIcon}>
        <Icon aria-hidden />
        {item.badge ? <span className={s.bottomBadge}>{item.badge}</span> : null}
      </span>
      <span>{item.label}</span>
    </Link>
  );
}
