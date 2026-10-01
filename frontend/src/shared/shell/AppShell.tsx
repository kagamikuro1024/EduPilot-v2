"use client";

import { Bell, Check, ChevronDown, KeyRound, LogOut, Menu as MenuIcon, PanelLeftClose, PanelLeft, RotateCcw, Search, Settings, UserPlus, Users } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { BT03_SEED } from "@/mock/assess";
import { COURSE_1, COURSE_2, DEMO_STUDENT_BLURB, DEMO_STUDENT_IDS, ROLE_LABEL, STAFF, STUDENTS, SUBJECT, type Role } from "@/mock/core";
import { ASSIGNED_AT, BT03_SUBMITTED_AT, CH5_UPLOADED_AT, agoLabel, reviewPending, ticketStats } from "@/mock/derive";
import { docById } from "@/mock/docs";
import { markNoteRead, notesFor, viewerKey, type Note } from "@/mock/notes";
import { KEYS, SCHEMES_SEED, type Bt03State, type SchemesState, type Ticket } from "@/mock/state";
import { mergeTickets } from "@/mock/support";
import { ALL_COURSES, clearDemoSession } from "@/shared/session/cookies";
import { useSession } from "@/shared/session/session";
import { useEnsureClock, useSimNow } from "@/shared/state/clock";
import { resetDemo, useDemoSlice } from "@/shared/state/demo";
import { Button, ButtonLink, Drawer, Field, Input, Kbd, MenuDivider, MenuList, Page, PageHeader, Popover, Section } from "@/shared/ui";
import { CommandPalette } from "@/shared/ui/CommandPalette";
import { MOBILE_PRIMARY, canOpen, navFor, needsCourse, whoCanOpen, type NavItem } from "./nav";
import s from "./AppShell.module.css";

/** Mốc giả lập của hai thông báo nền cho Quản trị viên (không có sự kiện nào sinh ra chúng). */
const FALLBACK_RATE_AT = new Date("2026-10-29T08:40:00+07:00");
const BUDGET_AT = new Date("2026-10-29T08:20:00+07:00");

/**
 * Mục "trạng thái" của chuông: TÍNH TỪ DỮ LIỆU (SRS 4.9) nên tự biến khi sự kiện xảy ra;
 * không có chấm chưa đọc (chỉ sự kiện mới có). Tên tài liệu lấy từ bảng N5, mốc từ 4.8 N6.
 */
function statusNotes(role: Role, studentId: string | undefined, hasCourse: boolean, bt03: Bt03State, schemes: SchemesState, viewer: string): Note[] {
  const read = [viewer];
  const out: Note[] = [];
  if (role === "student" && hasCourse) {
    const ch5 = docById("d-ch5")!;
    out.push({ id: "n-doc-ch5", to: { roles: ["student"] }, title: `Tài liệu mới: ${ch5.title}`, meta: "Thư viện", href: "/library", ms: CH5_UPLOADED_AT.getTime(), readBy: read });
    if (studentId === "sv-2" && bt03.status !== "published") {
      out.push({
        id: "n-bt03-waiting",
        to: { studentId: "sv-2" },
        title: "Bài tập 03 đã nộp, đang chờ chấm",
        meta: "Bài tập",
        href: "/assignments/bt03",
        ms: BT03_SUBMITTED_AT.getTime(),
        readBy: read,
      });
    }
  }
  if (role === "teacher" || role === "ta") {
    out.push({
      id: "n-assigned",
      to: { roles: ["teacher", "ta"] },
      title: `Bạn được phân công lớp An ninh mạng – 761988. Mã tham gia: BX4P9TW`,
      meta: "Quản trị viên",
      href: `/class/members?course=${COURSE_2}`,
      ms: ASSIGNED_AT.getTime(),
      readBy: read,
    });
  }
  if (role === "teacher" && schemes[COURSE_2]?.status !== "confirmed") {
    out.push({
      id: "n-scheme-2",
      to: { roles: ["teacher"] },
      title: "Công thức điểm lớp 761988 chưa được xác nhận",
      meta: "Sổ điểm",
      href: `/gradebook/scheme?course=${COURSE_2}`,
      ms: ASSIGNED_AT.getTime(),
      readBy: read,
    });
  }
  if (role === "admin") {
    out.push(
      { id: "n-fallback", to: { roles: ["admin"] }, title: "Tỷ lệ dùng model dự phòng tăng lên 1,2%", meta: "Quan sát AI", href: "/observability", ms: FALLBACK_RATE_AT.getTime(), readBy: read },
      { id: "n-budget", to: { roles: ["admin"] }, title: "Ngân sách LLM tháng đã dùng 62%", meta: "Cấu hình LLM", href: "/settings/llm", ms: BUDGET_AT.getTime(), readBy: read },
    );
  }
  return out;
}

/** Tên người hiển thị: GV có học hàm ("TS. Lê Thu Hà"), TA / Admin / SV chỉ có tên. */
function displayName(role: Role, name: string, title?: string) {
  return role === "teacher" && title ? `${title} ${name}` : name;
}

const NARROW = "(max-width: 719px)";
function subscribeNarrow(cb: () => void) {
  const mql = window.matchMedia(NARROW);
  mql.addEventListener("change", cb);
  return () => mql.removeEventListener("change", cb);
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const { role, user, studentId, course, courses, isAll, hasCourse, switchTo, setCourse } = useSession();
  const pathname = usePathname();
  const router = useRouter();
  const groups = useMemo(() => navFor(role, hasCourse), [role, hasCourse]);
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
  const courseTitle = !hasCourse ? "Chưa có lớp" : isAll ? "Tất cả lớp của tôi" : `${course.code} · ${course.name}`;
  const personName = displayName(role, user.name, user.title);
  const viewer = viewerKey(role, studentId);
  const scope = useMemo(() => (isAll ? courses.map((c) => c.id) : hasCourse ? [course.id] : []), [isAll, courses, hasCourse, course.id]);

  // Đồng hồ giả lập: khung app lưu mốc t0 một lần, mọi chuỗi thời gian tương đối đọc `now` này (SRS 4.8 N6).
  useEnsureClock();
  const now = useSimNow();

  // N9: badge tính từ dữ liệu, giảm ngay khi hành động xảy ra; không bao giờ ghi số ở nav.ts.
  const [storedTickets] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const badges = {
    inbox: ticketStats(mergeTickets(storedTickets), scope, now).open,
    grading: scope.includes(COURSE_1) ? reviewPending(bt03.status).length : 0,
  };

  const [storedNotes] = useDemoSlice<Note[]>(KEYS.notes, []);
  const notes = useMemo(
    () => notesFor([...storedNotes, ...statusNotes(role, studentId, hasCourse, bt03, schemes, viewer)], role, studentId, scope),
    [storedNotes, role, studentId, hasCourse, bt03, schemes, viewer, scope],
  );
  const unread = notes.filter((n) => !n.readBy.includes(viewer)).length;

  // Thanh bên và thanh dưới loại trừ nhau: chỉ gắn một bộ badge vào DOM ở mỗi bề rộng.
  const narrow = useSyncExternalStore(subscribeNarrow, () => window.matchMedia(NARROW).matches, () => false);

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

      {!narrow && (
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
                    <NavLink item={item} active={isActive(item.href)} collapsed={collapsed} badge={item.badgeKey ? badges[item.badgeKey] : 0} />
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
      )}

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
                {unread > 0 && <span className={s.unreadDot} data-part="bell-dot" aria-hidden />}
              </button>
            )}
          >
            {(close) => (
              <div className={s.notes}>
                <p className={s.panelLabel}>Thông báo</p>
                {notes.length === 0 ? (
                  <p className={s.noteEmpty}>Chưa có thông báo nào.</p>
                ) : (
                  <ul>
                    {notes.map((n) => {
                      const ago = agoLabel(n.ms, now);
                      return (
                        <li key={n.id}>
                          <Link
                            href={n.href}
                            className={s.note}
                            onClick={() => {
                              markNoteRead(n.id, viewer);
                              close();
                            }}
                          >
                            <span className={[s.noteDot, n.readBy.includes(viewer) ? "" : s.noteUnread].join(" ")} aria-hidden />
                            <span>
                              <span className={s.noteTitle}>{n.title}</span>
                              <span className={s.noteMeta}>
                                {n.meta} · {n.just && ago === "vừa xong" ? n.just : ago}
                              </span>
                            </span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                )}
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
        {!canOpen(role, pathname) ? (
          <Page>
            <PageHeader
              title="Bạn không có quyền mở trang này"
              description={whoCanOpen(pathname)}
              actions={<ButtonLink href="/">Về Hôm nay</ButtonLink>}
            />
          </Page>
        ) : !hasCourse && needsCourse(role, pathname) ? (
          <NoCourse />
        ) : (
          children
        )}
      </main>

      {narrow && (
        <>
          <nav className={s.bottomNav} aria-label="Điều hướng chính">
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

          <Drawer open={more} onClose={() => setMore(false)} title="Thêm">
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

      <CommandPalette open={palette} onClose={() => setPalette(false)} items={flat} />
    </div>
  );
}

function NavLink({ item, active, collapsed, badge }: { item: NavItem; active: boolean; collapsed: boolean; badge: number }) {
  const Icon = item.icon;
  return (
    <Link href={item.href} className={s.navItem} aria-current={active ? "page" : undefined} title={collapsed ? item.label : undefined}>
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
  const router = useRouter();
  const [code, setCode] = useState("");
  return (
    <Page>
      <PageHeader
        title="Bạn chưa vào lớp nào"
        description="Nhập mã tham gia do giảng viên cung cấp để dùng Chat riêng, Threads, Luyện đề, Thư viện và Lịch."
      />
      <Section>
        <form
          className={s.joinForm}
          onSubmit={(e) => {
            e.preventDefault();
            if (code.trim()) router.push(`/join/${code.trim().toUpperCase()}`);
          }}
        >
          <Field label="Mã tham gia" helper="Mã gồm 7 ký tự, không phân biệt chữ hoa chữ thường.">
            {(id, describedBy) => (
              <Input
                id={id}
                aria-describedby={describedBy}
                value={code}
                autoComplete="off"
                spellCheck={false}
                maxLength={12}
                placeholder="Nhập mã tham gia"
                onChange={(e) => setCode(e.target.value)}
              />
            )}
          </Field>
          <Button type="submit" variant="primary" disabled={!code.trim()}>
            Tiếp tục
          </Button>
        </form>
      </Section>
    </Page>
  );
}
