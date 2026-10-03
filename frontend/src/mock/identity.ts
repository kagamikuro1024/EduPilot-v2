import type { Role } from "./core";

// Bảng ánh xạ tài khoản seed → người mock (SRS FEAT-account-security 7.4). Màn mock lấy VAI từ JWT và NGƯỜI MOCK từ email đã xác minh
// của phiên; chỉ dùng để chọn bộ dữ liệu giả, không bao giờ để mở dữ liệu thật của ai.
const BY_EMAIL: Record<string, string> = {
  "sv.gioi@edupilot.local": "sv-1",
  "sv.kha@edupilot.local": "sv-2",
  "sv.nguyco@edupilot.local": "sv-3",
  "sv.moi@edupilot.local": "sv-4",
};

const STAFF_SEED = new Set(["admin@edupilot.local", "teacher@edupilot.local", "ta@edupilot.local"]);

/** Tài khoản seed có người mock riêng (tên mock thay tên thật). */
export function isSeedAccount(email: string): boolean {
  const e = email.trim().toLowerCase();
  return e in BY_EMAIL || STAFF_SEED.has(e);
}

/** Mã sinh viên mock (sv-1…) cho một phiên Sinh viên; email lạ → sv-2 (Sinh viên B). */
export function mockStudentFor(email: string): string {
  return BY_EMAIL[email.trim().toLowerCase()] ?? "sv-2";
}

/** Vai nhận biết được trong dữ liệu mock (đề phòng claim lạ). */
export function isMockRole(r: string): r is Role {
  return r === "student" || r === "ta" || r === "teacher" || r === "admin";
}
