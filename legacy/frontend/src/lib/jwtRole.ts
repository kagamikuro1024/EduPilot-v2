import type { UserRole } from "@/types/auth";

function base64UrlToJson(payload: string): unknown {
  const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
  const pad = base64.length % 4;
  const padded = base64 + (pad === 0 ? "" : "=".repeat(4 - pad));
  const json = atob(padded);
  return JSON.parse(json) as unknown;
}

/**
 * Reads `role` from JWT payload (added by Java AuthService). Used when persisted state has no role.
 */
export function parseRoleFromJwt(token: string): UserRole | null {
  try {
    const parts = token.split(".");
    if (parts.length < 2) return null;
    const body = base64UrlToJson(parts[1]) as { role?: string };
    const r = body.role;
    if (r === "STUDENT" || r === "TA" || r === "ADMIN") return r;
  } catch {
    return null;
  }
  return null;
}
