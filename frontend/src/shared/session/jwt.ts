import type { Role } from "@/mock/core";

// Chỉ GIẢI MÃ phần claim của JWT để biết vai / sub / email / hạn dùng cho khung giao diện. KHÔNG xác minh chữ ký:
// gateway mới là nơi xác minh (SRS FEAT-ui-foundation 3.3). Token chỉ nằm trong bộ nhớ (`tokenStore`).
export type Claims = { role: Role; sub: string; email: string; exp: number };
export type TokenCheck = { ok: true; claims: Claims } | { ok: false; reason: "invalid" | "expired" };

const ROLE_BY_CLAIM: Record<string, Role> = { ADMIN: "admin", TEACHER: "teacher", TA: "ta", STUDENT: "student" };

function b64urlJson(part: string): unknown {
  const pad = part.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(part.length / 4) * 4, "=");
  const bytes = Uint8Array.from(atob(pad), (c) => c.charCodeAt(0));
  return JSON.parse(new TextDecoder().decode(bytes));
}

export function checkToken(token: string, nowMs = Date.now()): TokenCheck {
  const parts = token.trim().split(".");
  if (parts.length !== 3 || parts.some((p) => !/^[A-Za-z0-9_-]+$/.test(p))) return { ok: false, reason: "invalid" };
  try {
    const head = b64urlJson(parts[0]) as { alg?: string };
    const c = b64urlJson(parts[1]) as { role?: unknown; sub?: unknown; email?: unknown; exp?: unknown };
    const role = typeof c.role === "string" ? ROLE_BY_CLAIM[c.role] : undefined;
    if (head.alg !== "HS256" || !role || typeof c.sub !== "string" || typeof c.exp !== "number") return { ok: false, reason: "invalid" };
    if (c.exp * 1000 <= nowMs) return { ok: false, reason: "expired" };
    return { ok: true, claims: { role, sub: c.sub, email: typeof c.email === "string" ? c.email : "", exp: c.exp } };
  } catch {
    return { ok: false, reason: "invalid" };
  }
}
