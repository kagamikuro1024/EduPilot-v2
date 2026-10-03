import {
  Activity,
  Building2,
  CalendarDays,
  ChartColumn,
  CircleHelp,
  ClipboardCheck,
  FileText,
  House,
  Inbox,
  Library,
  Lightbulb,
  ListChecks,
  MessageSquare,
  MessagesSquare,
  NotebookPen,
  Plug,
  SlidersHorizontal,
  UserRound,
  Users,
  type LucideIcon,
} from "lucide-react";
import type { Role } from "@/mock/core";

/**
 * `short`: nhãn ngắn cho bottom nav điện thoại khi nhãn đầy đủ quá dài.
 * `badgeKey`: con số do khung app tính từ `mock/derive.ts` (SRS 4.8 N9) — KHÔNG ghi số ở đây.
 */
export type NavItem = { href: string; label: string; short?: string; icon: LucideIcon; badgeKey?: "inbox" | "grading" };
export type NavGroup = { label?: string; items: NavItem[] };

const STUDENT: NavGroup[] = [
  {
    items: [
      { href: "/", label: "Hôm nay", icon: House },
      { href: "/chat", label: "Chat riêng", icon: MessageSquare },
      { href: "/threads", label: "Threads", icon: MessagesSquare },
      { href: "/practice", label: "Luyện đề", icon: CircleHelp },
      { href: "/library", label: "Thư viện", icon: Library },
      { href: "/calendar", label: "Lịch", icon: CalendarDays },
      { href: "/me", label: "Kết quả của tôi", icon: ChartColumn },
    ],
  },
];

function staff(role: "ta" | "teacher"): NavGroup[] {
  const groups: NavGroup[] = [
    {
      label: "Làm việc",
      items: [
        { href: "/", label: "Hôm nay", icon: House },
        { href: "/inbox", label: "Hộp thư hỗ trợ", short: "Hộp thư", icon: Inbox, badgeKey: "inbox" },
        { href: "/students", label: "Sinh viên", icon: Users },
        { href: "/attendance", label: "Điểm danh", icon: ClipboardCheck },
      ],
    },
    {
      label: "Đánh giá",
      items: [
        { href: "/gradebook", label: "Sổ điểm", icon: NotebookPen },
        { href: "/grading", label: "Chấm bài", icon: ListChecks, badgeKey: "grading" },
        { href: "/questions", label: "Ngân hàng câu hỏi", icon: CircleHelp },
      ],
    },
    {
      label: "Nội dung",
      items: [
        { href: "/threads", label: "Threads", icon: MessagesSquare },
        { href: "/documents", label: "Tài liệu", icon: FileText },
        { href: "/calendar", label: "Lịch", icon: CalendarDays },
      ],
    },
    {
      label: "Hiểu lớp học",
      items: [
        { href: "/insights", label: "Insights", icon: Lightbulb },
        { href: "/analytics", label: "Analytics", icon: ChartColumn },
      ],
    },
  ];
  if (role === "teacher") {
    groups.push({
      label: "Hệ thống",
      items: [
        { href: "/observability", label: "Quan sát AI", icon: Activity },
        { href: "/settings/llm", label: "Cấu hình LLM", icon: SlidersHorizontal },
        { href: "/settings/integrations", label: "Tích hợp", icon: Plug },
      ],
    });
  }
  return groups;
}

const ADMIN: NavGroup[] = [
  {
    label: "Vận hành",
    items: [
      { href: "/", label: "Hôm nay", icon: House },
      { href: "/observability", label: "Quan sát AI", icon: Activity },
    ],
  },
  {
    label: "Quản trị",
    items: [
      { href: "/admin/courses", label: "Lớp học", icon: Building2 },
      { href: "/admin/users", label: "Người dùng", icon: UserRound },
    ],
  },
  {
    label: "Cấu hình",
    items: [
      { href: "/settings/llm", label: "Cấu hình LLM", icon: SlidersHorizontal },
      { href: "/settings/integrations", label: "Tích hợp", icon: Plug },
    ],
  },
];

/** SV chưa vào lớp chỉ có `Hôm nay` (SRS 4.3.2) — mọi mục khác cần dữ liệu lớp. */
const STUDENT_NO_COURSE: NavGroup[] = [{ items: [STUDENT[0].items[0]] }];

export function navFor(role: Role, hasCourse = true): NavGroup[] {
  if (role === "student") return hasCourse ? STUDENT : STUDENT_NO_COURSE;
  if (role === "admin") return ADMIN;
  return staff(role);
}

/** Bottom nav điện thoại: tối đa 4 đích + "Thêm" (DESIGN.md §2). */
export const MOBILE_PRIMARY: Record<Role, string[]> = {
  student: ["/", "/chat", "/threads", "/practice"],
  ta: ["/", "/inbox", "/attendance", "/students"],
  teacher: ["/", "/inbox", "/attendance", "/students"],
  admin: ["/", "/observability", "/admin/courses", "/settings/llm"],
};

// Ai được mở route nào (bản mô phỏng của CourseAccessGuard / RBAC) — ma trận SRS prototype mục 2.
// Khớp theo tiền tố dài nhất. Quyền "chỉ đọc" của TA ở sổ điểm / cấu hình do từng trang tự xử lý.
const ACCESS: Array<[string, Role[]]> = [
  ["/inbox", ["ta", "teacher"]],
  ["/students", ["ta", "teacher"]],
  ["/attendance", ["ta", "teacher"]],
  ["/gradebook", ["ta", "teacher"]],
  ["/grading", ["ta", "teacher"]],
  ["/questions", ["ta", "teacher"]],
  ["/documents", ["ta", "teacher"]],
  ["/insights", ["ta", "teacher"]],
  ["/analytics", ["ta", "teacher"]],
  ["/class", ["ta", "teacher"]],
  ["/observability", ["teacher", "admin"]],
  ["/settings/llm", ["teacher", "admin"]],
  ["/settings/integrations", ["teacher", "admin"]],
  ["/settings", ["student", "ta", "teacher", "admin"]], // Tài khoản và bảo mật: mọi vai, chỉ tác động chính mình
  ["/admin", ["admin"]],
  ["/chat", ["student"]],
  ["/practice", ["student"]],
  ["/library", ["student"]],
  ["/me", ["student"]],
  ["/assignments", ["student"]],
  ["/join", ["student"]],
  ["/threads", ["student", "ta", "teacher"]],
  ["/calendar", ["student", "ta", "teacher"]],
];

function ruleFor(pathname: string) {
  return ACCESS.filter(([p]) => pathname === p || pathname.startsWith(`${p}/`)).sort((a, b) => b[0].length - a[0].length)[0];
}

export function canOpen(role: Role, pathname: string) {
  const rule = ruleFor(pathname);
  return !rule || rule[1].includes(role);
}

/** Route cần dữ liệu lớp: mọi route của SV trừ `Hôm nay` và `/join` (SRS 4.3.2). */
export function needsCourse(role: Role, pathname: string) {
  return role === "student" && pathname !== "/" && pathname !== "/join" && !pathname.startsWith("/join/");
}

const NOUN: Record<Role, string> = { teacher: "giảng viên", ta: "trợ giảng", student: "sinh viên", admin: "quản trị viên" };

/** "Trang này dành cho giảng viên và trợ giảng." — câu lý do ở màn chặn quyền (FR-X4). */
export function whoCanOpen(pathname: string) {
  const rule = ruleFor(pathname);
  const roles = (rule?.[1] ?? []).slice().sort((a, b) => Object.keys(NOUN).indexOf(a) - Object.keys(NOUN).indexOf(b));
  const names = roles.map((r) => NOUN[r]);
  return `Trang này dành cho ${names.length > 1 ? `${names.slice(0, -1).join(", ")} và ${names[names.length - 1]}` : names[0]}.`;
}

// ---- Route → backend → phase (SRS FEAT-ui-foundation 7.6): dùng cho EmptyState ở build `NEXT_PUBLIC_MOCK_SCREENS=0` ----
export type Backend = { phase: string; name: string };
const MOCK_BACKEND: Array<[string, Backend]> = [
  ["/admin", { phase: "P2", name: "Lớp học" }],
  ["/class", { phase: "P2", name: "Lớp học" }],
  ["/join", { phase: "P2", name: "Lớp học" }],
  ["/chat", { phase: "P3", name: "Hỏi đáp" }],
  ["/threads", { phase: "P3", name: "Hỏi đáp" }],
  ["/inbox", { phase: "P4", name: "Hộp thư hỗ trợ" }],
  ["/students", { phase: "P5", name: "Sinh viên và điểm danh" }],
  ["/attendance", { phase: "P5", name: "Sinh viên và điểm danh" }],
  ["/gradebook", { phase: "P6", name: "Sổ điểm" }],
  ["/me", { phase: "P6", name: "Sổ điểm" }],
  ["/grading", { phase: "P7", name: "Chấm bài" }],
  ["/assignments", { phase: "P7", name: "Bài tập" }],
  ["/settings/integrations", { phase: "P7", name: "Tích hợp" }],
  ["/documents", { phase: "P8", name: "Tài liệu, thư viện, lịch" }],
  ["/library", { phase: "P8", name: "Tài liệu, thư viện, lịch" }],
  ["/calendar", { phase: "P8", name: "Tài liệu, thư viện, lịch" }],
  ["/practice", { phase: "P9", name: "Luyện đề" }],
  ["/questions", { phase: "P9", name: "Luyện đề" }],
  ["/insights", { phase: "P10", name: "Hiểu lớp học" }],
  ["/analytics", { phase: "P10", name: "Hiểu lớp học" }],
  ["/observability", { phase: "P10", name: "Hiểu lớp học" }],
];

/** Route chưa có backend thật → phase sẽ dựng nó; `null` = backend thật (`/settings/llm`) hoặc route PU (`/dev/*`). `/` là "Hôm nay" (P2). */
export function mockBackend(pathname: string): Backend | null {
  if (pathname === "/") return { phase: "P2", name: "Lớp học" };
  const hit = MOCK_BACKEND.filter(([p]) => pathname === p || pathname.startsWith(`${p}/`)).sort((a, b) => b[0].length - a[0].length)[0];
  return hit ? hit[1] : null;
}

/** Route có backend thật cần JWT (US-PU-04 AC9). */
export const needsToken = (pathname: string) => pathname === "/settings/llm" || pathname.startsWith("/settings/llm/");
