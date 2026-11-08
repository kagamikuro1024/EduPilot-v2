"use client";
import { formatWhen } from "@/shared/lib/datetime";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/admin/users*`). Admin chỉ thấy dữ liệu tối thiểu: không MSSV, không mật khẩu.
export type AdminRole = "ADMIN" | "TEACHER" | "TA" | "STUDENT";
export type AdminStatus = "PENDING_VERIFICATION" | "INVITED" | "ACTIVE" | "DISABLED";
export type AdminUser = { id: string; email: string; full_name: string; role: AdminRole; status: AdminStatus; last_login_at: string | null; version: number };

export const ROLE_TEXT: Record<AdminRole, string> = { ADMIN: "Quản trị viên", TEACHER: "Giảng viên", TA: "Trợ giảng", STUDENT: "Sinh viên" };
export const STATUS_TEXT: Record<AdminStatus, { text: string; tone: "green" | "amber" | "red" }> = {
  ACTIVE: { text: "Đang dùng", tone: "green" },
  INVITED: { text: "Chờ nhận lời mời", tone: "amber" },
  PENDING_VERIFICATION: { text: "Chờ xác minh email", tone: "amber" },
  DISABLED: { text: "Đã khoá", tone: "red" },
};

export const USERS_KEY = ["admin", "users"] as const;

/** "Hôm nay, 09:20", "Thứ Hai, 21/09 · 14:00", … hoặc "Chưa đăng nhập". */
export function lastSeen(iso: string | null, now = new Date()): string {
  if (!iso) return "Chưa đăng nhập";
  return formatWhen(iso, now);
}
