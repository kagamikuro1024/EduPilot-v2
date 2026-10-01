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

/** `short`: nhãn ngắn cho bottom nav điện thoại khi nhãn đầy đủ quá dài. */
export type NavItem = { href: string; label: string; short?: string; icon: LucideIcon; badge?: number };
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
        { href: "/inbox", label: "Hộp thư hỗ trợ", short: "Hộp thư", icon: Inbox, badge: 5 },
        { href: "/students", label: "Sinh viên", icon: Users },
        { href: "/attendance", label: "Điểm danh", icon: ClipboardCheck },
      ],
    },
    {
      label: "Đánh giá",
      items: [
        { href: "/gradebook", label: "Sổ điểm", icon: NotebookPen },
        { href: "/grading", label: "Chấm bài", icon: ListChecks, badge: 4 },
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

export function navFor(role: Role): NavGroup[] {
  if (role === "student") return STUDENT;
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
  ["/settings", ["teacher", "admin"]],
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

const NOUN: Record<Role, string> = { teacher: "giảng viên", ta: "trợ giảng", student: "sinh viên", admin: "quản trị viên" };

/** "Trang này dành cho giảng viên và trợ giảng." — câu lý do ở màn chặn quyền (FR-X4). */
export function whoCanOpen(pathname: string) {
  const rule = ruleFor(pathname);
  const roles = (rule?.[1] ?? []).slice().sort((a, b) => Object.keys(NOUN).indexOf(a) - Object.keys(NOUN).indexOf(b));
  const names = roles.map((r) => NOUN[r]);
  return `Trang này dành cho ${names.length > 1 ? `${names.slice(0, -1).join(", ")} và ${names[names.length - 1]}` : names[0]}.`;
}
