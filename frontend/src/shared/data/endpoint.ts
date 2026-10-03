// Gốc API: một nguồn duy nhất cho apiClient và lớp phiên. Để trống NEXT_PUBLIC_API_URL = cùng origin (Caddy chuyển /api/* tới gateway).
export const API_ORIGIN = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(/\/$/, "");
export const API_BASE = `${API_ORIGIN}/api/v1`;
