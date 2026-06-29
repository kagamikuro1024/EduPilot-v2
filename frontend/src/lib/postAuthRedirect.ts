import type { UserRole } from "@/types/auth";

/**
 * Default landing after successful authentication.
 * TA/ADMIN → analytics dashboard; STUDENT (or unknown) → assignments hub.
 */
export function getPostAuthPath(role: UserRole | null): "/analytics" | "/assignments" {
  if (role === "TA" || role === "ADMIN") return "/analytics";
  return "/assignments";
}
