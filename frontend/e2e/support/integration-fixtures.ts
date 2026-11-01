import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Gateway giả cho `/settings/integrations` (US-P7-05, US-P7-11) theo HỢP ĐỒNG THẬT `backend-go/api/openapi.yaml`.
// Máy chủ KHÔNG BAO GIỜ trả mật khẩu IMAP hay client secret của Teams: các hàm dựng view ở đây cũng vậy.
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const e = (status: number, code: string, extra: Record<string, unknown> = {}) => j({ code, message: code, trace_id: "t", ...extra }, status);

export const COURSE = "00000000-0000-7000-8000-00000000c001";
export const COURSE2 = "00000000-0000-7000-8000-00000000c002";
export const JOB = "00000000-0000-7000-8000-000000009001";

export type View = Record<string, unknown>;

export const imapView = (over: View = {}): View => ({
  kind: "IMAP",
  configured: true,
  enabled: true,
  status: "CONNECTED",
  config: { host: "imap.ptit.edu.vn", port: 993, security: "TLS", username: "nopbai@ptit.edu.vn", mailbox: "INBOX", poll_seconds: 60 },
  read_only: false,
  last_ok_at: "2026-10-23T01:10:00.000Z",
  last_sync_at: "2026-10-23T01:10:00.000Z",
  last_error: null,
  error_since: null,
  version: 3,
  ...over,
});

export const smtpView = (over: View = {}): View => ({
  kind: "SMTP",
  configured: true,
  enabled: true,
  status: "CONNECTED",
  config: { host: "smtp.ptit.edu.vn", port: 587, security: "STARTTLS", from: "no-reply@ptit.edu.vn" },
  read_only: true,
  last_ok_at: "2026-10-23T01:00:00.000Z",
  last_sync_at: null,
  last_error: null,
  error_since: null,
  version: 1,
  ...over,
});

export const teamsView = (over: View = {}): View => ({
  kind: "TEAMS",
  configured: true,
  enabled: true,
  status: "CONNECTED",
  config: { tenant_id: "ptit.edu.vn", client_id: "11111111-2222-3333-4444-555555555555" },
  read_only: false,
  last_ok_at: "2026-10-23T01:00:00.000Z",
  last_sync_at: null,
  last_error: null,
  error_since: null,
  sandbox: false,
  available: true,
  version: 2,
  ...over,
});

const course = (id: string, code: string, name: string) => ({
  id, class_code: code, subject_code: "INT1006", name, semester: "2026-2027-HK1", status: "ACTIVE",
  teacher: { id: "u1", full_name: "Lê Thu Hà" }, assistants_count: 1, students_active: 30, students_pending: 0, capacity: 40, version: 1,
});

export type Call = { method: string; path: string; body: string; headers: Record<string, string> };
type Handler = (route: Route, dflt: (r: Route) => Promise<void> | void) => Promise<void> | void;

export type Opts = {
  imap?: View;
  smtp?: View;
  teams?: View;
  /** kết quả `POST …/imap/test` */
  imapTest?: { ok: boolean; kind?: string };
  teamsTest?: { ok: boolean; kind?: string };
  /** kết quả việc đồng bộ (`GET /jobs/{id}`) */
  job?: { status?: string; progress?: number; result?: unknown };
  h?: Record<string, Handler>;
};

/**
 * Gateway giả có trạng thái: `PUT imap|teams` kiểm `version`, tăng version, GIỮ mật khẩu đã nhận nhưng không trả lại;
 * `POST sync-now` trả 202 {job_id}; `GET /jobs/{id}` trả kết quả. `h` thay hành vi mặc định (429, 409, 422…).
 */
export async function integrationsApi(p: Page, opts: Opts = {}) {
  const calls: Call[] = [];
  let imap = { ...imapView(), ...(opts.imap ?? {}) };
  let teams = { ...teamsView(), ...(opts.teams ?? {}) };
  const smtp = { ...smtpView(), ...(opts.smtp ?? {}) };
  /** mật khẩu đã lưu — chỉ để khẳng định "trống = giữ cũ", không bao giờ vào phản hồi */
  const secrets: Record<string, string> = {};

  const handle = async (route: Route, skipCustom = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const url = new URL(req.url());
    const path = url.pathname.replace("/api/v1", "");
    const body = req.postData() ?? "";
    const tail = path.replace(/^\/courses\/[^/]+\//, "").replace(/^\/jobs\/[^/]+$/, "jobs/:id");
    const key = `${req.method()} ${tail}`;
    if (!skipCustom) calls.push({ method: req.method(), path: path + url.search, body, headers: req.headers() });
    const custom = opts.h?.[key];
    if (custom && !skipCustom) return custom(route, (r) => handle(r, true));

    if (key === "GET /admin/courses") return route.fulfill(j({ items: [course(COURSE, "761987", "An ninh mạng"), course(COURSE2, "761988", "Mật mã học")], next_cursor: null }));
    if (key === "GET integrations/smtp") return route.fulfill(j(smtp));
    if (key === "POST integrations/smtp/test") return route.fulfill(j({ queued: true }, 202));
    if (key === "GET integrations/imap") return route.fulfill(j(imap));
    if (key === "GET integrations/teams") return route.fulfill(j(teams));

    if (key === "PUT integrations/imap") {
      const b = JSON.parse(body) as { version?: number; password?: string; enabled: boolean; host: string; port: number; security: string; username: string; mailbox?: string; poll_seconds?: number };
      if (b.version !== undefined && b.version !== imap.version) return route.fulfill(e(409, "VERSION_CONFLICT", { details: { current_version: imap.version } }));
      if (!b.password && !secrets.imap) {
        return route.fulfill(e(422, "VALIDATION_FAILED", { details: [{ field: "password", code: "REQUIRED", message: "Lần đầu cần nhập mật khẩu." }] }));
      }
      if (b.password) secrets.imap = b.password;
      imap = {
        ...imap,
        configured: true,
        enabled: b.enabled,
        status: b.enabled ? "CONNECTED" : "NOT_CONNECTED",
        config: { host: b.host, port: b.port, security: b.security, username: b.username, mailbox: b.mailbox || "INBOX", poll_seconds: b.poll_seconds ?? 60 },
        last_error: null,
        error_since: null,
        version: (imap.version as number) + 1,
      };
      return route.fulfill(j(imap));
    }
    if (key === "POST integrations/imap/test") {
      const out = opts.imapTest ?? { ok: true };
      return route.fulfill(j(out));
    }
    if (key === "POST integrations/imap/sync-now") return route.fulfill(j({ job_id: JOB }, 202));

    if (key === "PUT integrations/teams") {
      const b = JSON.parse(body) as { version?: number; client_secret?: string; enabled: boolean; tenant_id: string; client_id: string };
      if (b.version !== undefined && b.version !== teams.version) return route.fulfill(e(409, "VERSION_CONFLICT", { details: { current_version: teams.version } }));
      if (!b.client_secret && !secrets.teams) {
        return route.fulfill(e(422, "VALIDATION_FAILED", { details: [{ field: "client_secret", code: "REQUIRED", message: "Lần đầu cần nhập client secret." }] }));
      }
      if (b.client_secret) secrets.teams = b.client_secret;
      teams = { ...teams, configured: true, enabled: b.enabled, status: b.enabled ? "CONNECTED" : "NOT_CONNECTED", config: { tenant_id: b.tenant_id, client_id: b.client_id }, version: (teams.version as number) + 1 };
      return route.fulfill(j(teams));
    }
    if (key === "POST integrations/teams/test") return route.fulfill(j(opts.teamsTest ?? { ok: true }));

    if (key === "GET jobs/:id") {
      const job = opts.job ?? { status: "SUCCEEDED", progress: 100, result: { received: 5, created: 3, matched: 2, unmatched: 1, ignored: 1, failed: 0 } };
      return route.fulfill(j({ id: JOB, status: job.status ?? "SUCCEEDED", progress: job.progress ?? 100, result: job.result }));
    }
    return route.fulfill(e(404, "NOT_FOUND"));
  };

  await p.route(/\/api\/v1\/(courses\/[^/]+\/integrations|admin\/courses|jobs\/)/, (r) => handle(r));
  return { calls, secrets, imap: () => imap, teams: () => teams, of: (m: string, suffix: string) => calls.filter((c) => c.method === m && c.path.endsWith(suffix)) };
}
