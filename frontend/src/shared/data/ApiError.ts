import { userMessageFor } from "@/shared/i18n/vi";

export type ServerErrorCode =
  | "BAD_REQUEST" | "UNAUTHENTICATED" | "TOKEN_EXPIRED" | "TOKEN_INVALID" | "FORBIDDEN" | "NOT_FOUND" | "METHOD_NOT_ALLOWED"
  | "CONFLICT" | "VERSION_CONFLICT" | "IDEMPOTENCY_IN_PROGRESS" | "PAYLOAD_TOO_LARGE" | "UNSUPPORTED_MEDIA_TYPE" | "VALIDATION_FAILED"
  | "INVALID_CURSOR" | "IDEMPOTENCY_KEY_REUSED" | "IDEMPOTENCY_KEY_REQUIRED" | "RATE_LIMITED" | "SSE_LIMIT_REACHED" | "INTERNAL"
  | "SERVICE_UNAVAILABLE" | "NOT_READY" | "DEADLINE_EXCEEDED"
  | "OVERLOADED" | "LLM_NOT_CONFIGURED" | "LLM_UNAVAILABLE" | "PROVIDER_IN_USE" | "MODEL_DIMS_MISMATCH" | "ROUTE_INVALID"
  | "INVALID_CREDENTIALS" | "LOGIN_THROTTLED" | "ACCOUNT_DISABLED" | "SESSION_REVOKED" | "LINK_INVALID" | "EMAIL_NOT_VERIFIED"
  | "JOIN_CODE_INVALID" | "COURSE_FULL" | "COURSE_ARCHIVED"
  | "EXAM_IN_PROGRESS" | "CHAT_BUSY" | "MESSAGE_TOO_LONG" | "CHAT_UNAVAILABLE" | "MESSAGE_NOT_RETRYABLE" | "PII_DETECTED" | "POST_STATE_CONFLICT";
export type ApiErrorCode = ServerErrorCode | "BAD_GATEWAY" | "NETWORK" | "ABORTED" | "PARSE_ERROR" | "BAD_TARGET";

export const AUTH_CODES: readonly string[] = ["TOKEN_EXPIRED", "TOKEN_INVALID", "UNAUTHENTICATED"];

export class ApiError extends Error {
  status: number;
  code: ApiErrorCode;
  userMessage: string;
  traceId?: string;
  details?: unknown;
  retryAfter?: number; // giây
  conflict?: { currentVersion: number; current: unknown };

  constructor(init: { status: number; code: string; message?: string; traceId?: string; details?: unknown; retryAfter?: number }) {
    super(init.message || init.code);
    this.name = "ApiError";
    this.status = init.status;
    this.code = init.code as ApiErrorCode;
    this.traceId = init.traceId;
    this.details = init.details;
    this.retryAfter = init.retryAfter;
    this.userMessage = userMessageFor(init.code, init.retryAfter);
    if (init.code === "VERSION_CONFLICT" && init.details && typeof init.details === "object") {
      const d = init.details as { current_version?: number; current?: unknown };
      if (typeof d.current_version === "number") this.conflict = { currentVersion: d.current_version, current: d.current };
    }
  }
}

/** 422 VALIDATION_FAILED: `details[]` = [{field, code, message}] → { field: message }. Trường lặp giữ lỗi đầu tiên. */
export function fieldErrors(err: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  if (!(err instanceof ApiError) || !Array.isArray(err.details)) return out;
  for (const d of err.details as Array<{ field?: string; message?: string; code?: string }>) {
    if (d && typeof d.field === "string" && !(d.field in out)) out[d.field] = d.message || d.code || "Giá trị chưa hợp lệ.";
  }
  return out;
}
