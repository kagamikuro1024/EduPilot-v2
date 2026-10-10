import { navFor } from "../../src/shared/shell/nav";
import type { DemoRole } from "./session";
import { TID } from "./thread-fixtures";

// Danh sách route để quét (axe, một-nút-chính): mọi đích điều hướng của vai (khớp ma trận quyền SRS 7.5) + vài route chi tiết đại diện.
// Route thuộc PU không cần vai: /login, /dev/ui, /dev/data.
const DETAIL: Record<DemoRole, string[]> = {
  student: [`/threads/${TID}`, "/practice/at-symmetric", "/practice/history", "/assignments/bt03", "/join"],
  ta: ["/class/members"],
  teacher: ["/class/members", "/gradebook/scheme"],
  admin: [],
};

export const ROLES: DemoRole[] = ["student", "ta", "teacher", "admin"];
export const routesFor = (role: DemoRole) => [...new Set([...navFor(role, true).flatMap((g) => g.items.map((i) => i.href)), ...DETAIL[role]])];
export const PU_ROUTES = ["/login", "/dev/ui", "/dev/data"];
