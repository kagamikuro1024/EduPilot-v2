"use client";

import { Bell, Check, ChevronDown, KeyRound, LogOut, Menu as MenuIcon, PanelLeftClose, PanelLeft, RotateCcw, Search, Settings, UserPlus, Users } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { DEMO_STUDENT_BLURB, DEMO_STUDENT_IDS, ROLE_LABEL, STAFF, STUDENTS, SUBJECT, type Role } from "@/mock/core";
import { KEYS, type Ticket } from "@/mock/state";
import { ALL_COURSES, clearDemoSession } from "@/shared/session/cookies";
import { useSession } from "@/shared/session/session";
import { resetDemo, useDemoSlice } from "@/shared/state/demo";
import { ButtonLink, Drawer, Kbd, MenuDivider, MenuList, Page, PageHeader, Popover } from "@/shared/ui";
import { CommandPalette } from "@/shared/ui/CommandPalette";
import { MOBILE_PRIMARY, canOpen, navFor, whoCanOpen, type NavItem } from "./nav";
import s from "./AppShell.module.css";

type Note = { title: string; meta: string; href: string; unread?: boolean };

/** Thông báo gốc theo vai; phần phát sinh từ tương tác (ticket) thêm ở `useNotes`. */
const NOTIFICATIONS: Record<Role, Note[]> = {
  student: [
    { title: "Tài liệu mới: Tuần 10 — Quản lý khoá và PKI", meta: "Thư viện · hôm qua", href: "/library" },
    { title: "Bài tập 03 đã nộp, đang chờ chấm", meta: "Bài tập · 12 ngày trước", href: "/assignments/bt03" },
  ],
  ta: [
    { title: "4 bài chấm lệch giữa hai lượt", meta: "Chấm bài · 1 giờ trước", href: "/grading", unread: true },
    { title: "Buổi 10 lớp 761987 đang diễn ra — chưa điểm danh", meta: "Điểm danh · 09:00", href: "/attendance", unread: true },
  ],
  teacher: [
    { title: "Bạn được phân công lớp An ninh mạng – 761988. Mã tham gia: BX4P9TW", meta: "Quản trị viên · hôm qua", href: "/class/members", unread: true },
    { title: "Công thức điểm lớp 761988 chưa được xác nhận", meta: "Sổ điểm · hôm qua", href: "/gradebook/scheme" },
  ],
  admin: [
    { title: "Tỷ lệ dùng model dự phòng tăng lên 1,2%", meta: "Quan sát AI · 40 phút trước", href: "/observability", unread: true },
    { title: "Ngân sách LLM tháng đã dùng 62%", meta: "Cấu hình LLM · sáng nay", href: "/settings/llm" },
  ],
};

function useNotes(role: Role, studentId: string | undefined): Note[] {
  const [tickets] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const out = [...NOTIFICATIONS[role]];
  const d3 = tickets.find((t) => t.id === "tk-d3");
  if (d3 && role === "student" && studentId === d3.studentId && d3.status === "answered" && !d3.closedBySv) {
    out.unshift({ title: "Giảng viên đã trả lời câu hỏi của bạn", meta: "Chat riêng · vừa xong", href: "/chat", unread: true });
  }
  if (d3 && (role === "teacher" || role === "ta") && d3.status === "open") {
    out.unshift({ title: "1 câu hỏi mới cần xử lý", meta: "Hộp thư hỗ trợ · vừa gửi", href: "/inbox", unread: true });
  }
  return out;
}

/** Tên người hiển thị: GV có học hàm ("TS. Lê Thu Hà"), TA / Admin / SV chỉ có tên. */
function displayName(role: Role, name: string, title?: string) {
  return role === "teacher" && title ? `${title} ${name}` : name;
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const { role, user, studentId, course, courses, isAll, hasCourse, switchTo, setCourse } = useSession();
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
  const notes = useNotes(role, studentId);
  const unread = notes.filter((n) => n.unread).length;
  const courseTitle = isAll ? "Tất cả lớp của tôi" : `${course.code} · ${course.name}`;
  const personName = displayName(role, user.name, user.title);

  function switchRole(next: Role, person?: string) {
    switchTo(next, person);
    if (!canOpen(next, pathname)) router.push("/");
    else router.refresh();
  }

  return (
    <div className={[s.shell, collapsed ? s.collapsed : ""].join(" ")}>
      <a href="#main" className={s.skip}>
        Bỏ qua điều hướng
      </a>

      <aside className={s.sidebar} aria-label="Điều hướng chính">
        <Link href="/" className={s.brand} data-part="brand" aria-label="EduPilot — Hôm nay">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={collapsed ? "/brand/logo-edupilot-mark.svg" : "/brand/logo-edupilot.svg"} alt="" height={32} className={collapsed ? s.logoMark : s.logo} />
        </Link>

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
                    <span className={s.courseCode}>{course.code}</span>
                    <span className={s.courseName} title={courseTitle}>
                      · {course.name}
                    </span>
                  </>
                )}
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                <p className={s.courseCurrent}>{role === "student" && !hasCourse ? "Chưa có lớp" : isAll ? "Tất cả lớp của tôi" : course.label}</p>
                <p className={s.panelLabel}>{role === "student" ? "Lớp của bạn" : "Lớp bạn phụ trách"}</p>
                <MenuList
                  onPicked={close}
                  items={[
                    ...courses.map((c) => ({
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
                    })),
                    ...(role === "teacher"
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
          <button type="button" className={s.search} onClick={() => setPalette(true)} aria-label="Tìm nhanh hoặc đi đến">
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
              <button type="button" className={s.profile} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true" aria-label={`Tài khoản: ${personName}`}>
                <span className={s.initials} aria-hidden>
                  {user.name
                    .split(" ")
                    .slice(-1)[0]
                    .slice(0, 1)}
                </span>
                <span className={s.profileRole}>{role === "student" ? user.name : ROLE_LABEL[role]}</span>
                <ChevronDown aria-hidden />
              </button>
            )}
          >
            {(close) => (
              <div className={s.coursePanel}>
                <div className={s.who}>
                  <p className={s.whoName}>{personName}</p>
                  <p className={s.whoRole}>{ROLE_LABEL[role]}</p>
                  <p className={s.whoMail}>{user.email}</p>
                </div>
                <MenuDivider />
                <p className={s.panelLabel}>Đổi vai</p>
                <MenuList
                  onPicked={close}
                  items={[
                    ...DEMO_STUDENT_IDS.map((id, i) => {
                      const st = STUDENTS.find((x) => x.id === id)!;
                      const here = role === "student" && studentId === id;
                      return {
                        label: (
                          <span className={s.courseOpt}>
                            <span>
                              Sinh viên {"ABCD"[i]} · {st.name}
                            </span>
                            <span className={s.courseOptMeta}>{DEMO_STUDENT_BLURB[id]}</span>
                          </span>
                        ),
                        icon: here ? <Check aria-hidden /> : <span className={s.iconGap} />,
                        onSelect: () => switchRole("student", id),
                      };
                    }),
                    ...(["ta", "teacher", "admin"] as const).map((r) => ({
                      label: (
                        <span className={s.courseOpt}>
                          <span>
                            {ROLE_LABEL[r]} · {STAFF[r].name}
                          </span>
                        </span>
                      ),
                      icon: r === role ? <Check aria-hidden /> : <span className={s.iconGap} />,
                      onSelect: () => switchRole(r),
                    })),
                  ]}
                />
                <MenuDivider />
                <MenuList
                  onPicked={close}
                  items={[
                    ...(role === "teacher" || role === "admin"
                      ? [{ label: "Cài đặt hệ thống", icon: <Settings aria-hidden />, onSelect: () => router.push("/settings/llm") }]
                      : []),
                    {
                      label: "Đặt lại dữ liệu demo",
                      icon: <RotateCcw aria-hidden />,
                      onSelect: () => {
                        resetDemo();
                        router.refresh();
                      },
                    },
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

      <p className={s.demoMobile}>Bản mô phỏng · dữ liệu giả</p>

      <main id="main" className={s.main} tabIndex={-1}>
        {canOpen(role, pathname) ? (
          children
        ) : (
          <Page>
            <PageHeader
              title="Bạn không có quyền mở trang này"
              description={whoCanOpen(pathname)}
              actions={<ButtonLink href="/">Về Hôm nay</ButtonLink>}
            />
          </Page>
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
      <span>{item.short ?? item.label}</span>
    </Link>
  );
}
