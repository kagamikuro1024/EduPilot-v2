// QC @real (P-11-29): thư viện dùng chung cho `qc-real-*.spec.ts`. Chạy trên stack ep-qc thật (không gateway giả).
// Chạy: QC_REAL=1 QC_BASE=https://localhost:<cổng HTTPS ep-qc> QC_PW=<SEED_DEFAULT_PASSWORD> pnpm exec playwright test -c playwright.qc-real-flow.config.ts
import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { randomUUID } from "node:crypto";
import { setTimeout as sleep } from "node:timers/promises";
import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

export const PW = process.env.QC_PW ?? "";
export const OUT = process.env.QC_OUT ?? "/tmp/qc-real-shots";
mkdirSync(OUT, { recursive: true });
const MAILPIT = process.env.QC_MAILPIT ?? "http://localhost:8525";
const PG = process.env.QC_PG ?? "ep-qc-postgres-1";
const PG_DB = process.env.QC_PG_DB ?? "edupilot";
/** Chờ tick của worker (reminder 5 phút, SLA 15 phút mặc định; hạ bằng REMINDER_TICK / TICKET_SLA_TICK cho gate nhanh). */
export const TICK_WAIT_MS = Number(process.env.QC_TICK_WAIT_MS ?? 17 * 60_000);

export function needReal() {
  test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: QC_REAL=1 playwright test -c playwright.qc-real-flow.config.ts");
  if (process.env.QC_REAL && !PW) throw new Error("đặt QC_PW (SEED_DEFAULT_PASSWORD)");
}

export const email = (u: string) => (u.includes("@") ? u : `${u}@edupilot.local`);
export const uniq = () => `${Date.now().toString(36)}${Math.floor(Math.random() * 1296).toString(36)}`;

/** psql trong container Postgres của ep-qc; trả các dòng, cột cách bằng `|`. */
export function sql(q: string): string[][] {
  const out = execFileSync("docker", ["exec", "-i", PG, "psql", "-U", "edupilot", "-d", PG_DB, "-At", "-F", "|", "-v", "ON_ERROR_STOP=1", "-c", q], { encoding: "utf8" });
  return out.trim() ? out.trim().split("\n").map((l) => l.split("|")) : [];
}

/** Xoá bộ đếm hạn mức đăng ký / quên mật khẩu / đăng nhập theo IP (5 lần / giờ) — fixture QC, không đụng khoá thu hồi phiên `ep:auth:rev`. */
export function resetAuthLimits() {
  const R = process.env.QC_REDIS ?? "ep-qc-redis-1";
  execFileSync("docker", ["exec", R, "sh", "-c", "redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:auth:forgot:*' | xargs -r redis-cli del"], { encoding: "utf8" });
}

export type Json = Record<string, unknown>;
export interface Reply { status: number; json: Json; text: string }

/** Gọi API thật. `token` rỗng = ẩn danh. POST/PUT/PATCH/DELETE tự gắn Idempotency-Key. */
export async function call(request: APIRequestContext, token: string | null, method: string, path: string, body?: unknown, headers: Record<string, string> = {}): Promise<Reply> {
  const h: Record<string, string> = { ...headers };
  if (token) h.authorization = `Bearer ${token}`;
  if (method !== "GET" && !h["Idempotency-Key"]) h["Idempotency-Key"] = randomUUID();
  const r = await request.fetch(`/api/v1${path}`, { method, headers: h, data: body === undefined ? undefined : body });
  const text = await r.text();
  let json: Json = {};
  try { json = JSON.parse(text) as Json; } catch { /* không phải JSON */ }
  return { status: r.status(), json, text };
}

const tokens = new Map<string, string>();
export async function login(request: APIRequestContext, user: string, pw = PW): Promise<string> {
  const key = `${user}|${pw}`;
  const hit = tokens.get(key);
  if (hit) return hit;
  const r = await call(request, null, "POST", "/auth/login", { email: email(user), password: pw });
  expect(r.status, `đăng nhập ${user}: ${r.text.slice(0, 120)}`).toBe(200);
  const t = String(r.json.access_token);
  tokens.set(key, t);
  return t;
}

export async function courseId(request: APIRequestContext, user: string, code = "761987"): Promise<string> {
  const r = await call(request, await login(request, user), "GET", "/me/courses");
  const items = (r.json.items ?? []) as { course: { id: string; class_code: string } }[];
  const hit = items.find((i) => i.course.class_code === code);
  expect(hit, `${user} không thấy lớp ${code}`).toBeTruthy();
  return hit!.course.id;
}

// ---- Mailpit ----
interface MailMeta { ID: string; Subject: string; Created: string; Snippet?: string; To: { Address: string }[] }
export async function waitMail(to: string, subject: RegExp, sinceMs: number, timeoutMs = 60_000): Promise<{ id: string; text: string; subject: string }> {
  const end = Date.now() + timeoutMs;
  for (;;) {
    const j = (await (await fetch(`${MAILPIT}/api/v1/messages?limit=100`)).json()) as { messages?: MailMeta[] };
    const m = (j.messages ?? []).find((x) => x.To.some((a) => a.Address === to) && subject.test(x.Subject) && Date.parse(x.Created) >= sinceMs - 2000);
    if (m) {
      const full = (await (await fetch(`${MAILPIT}/api/v1/message/${m.ID}`)).json()) as { Text: string };
      return { id: m.ID, text: full.Text, subject: m.Subject };
    }
    if (Date.now() > end) throw new Error(`không có thư "${subject}" tới ${to} trong ${timeoutMs / 1000} s`);
    await sleep(1500);
  }
}
export async function mailCount(to: string, subject: RegExp, bodyHas?: string): Promise<number> {
  const j = (await (await fetch(`${MAILPIT}/api/v1/messages?limit=200`)).json()) as { messages?: MailMeta[] };
  return (j.messages ?? []).filter((x) => x.To.some((a) => a.Address === to) && subject.test(x.Subject) && (!bodyHas || (x.Snippet ?? "").includes(bodyHas))).length;
}
/** Liên kết đầu tiên trong thư có `needle` (verify / reset / invite) — chỉ giữ đường dẫn + query để chạy trên mọi origin. */
export function linkIn(text: string, needle: RegExp): string {
  const m = [...text.matchAll(/https?:\/\/[^\s<>")]+/g)].map((x) => x[0]).find((u) => needle.test(u));
  if (!m) throw new Error(`thư không có liên kết khớp ${needle}: ${text.slice(0, 200)}`);
  const u = new URL(m);
  return u.pathname + u.search;
}

// ---- Giao diện ----
export async function uiLogin(page: Page, user: string, pw = PW) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email(user));
  await page.locator("input[type=password]").fill(pw);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}
export async function shot(page: Page, name: string) {
  await page.screenshot({ path: `${OUT}/${name}.png` });
}
/** Không tràn ngang; trả chuỗi ghi chú (cho test.info().annotations). */
export async function noOverflow(page: Page, soft = false) {
  const m = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth }));
  (soft ? expect.soft : expect)(m.sw, `tràn ngang ${m.sw}/${m.cw}`).toBeLessThanOrEqual(m.cw);
}
export function poll<T>(fn: () => Promise<T> | T, ok: (v: T) => boolean, timeoutMs: number, everyMs = 5000): Promise<T> {
  return (async () => {
    const end = Date.now() + timeoutMs;
    for (;;) {
      const v = await fn();
      if (ok(v) || Date.now() > end) return v;
      await sleep(everyMs);
    }
  })();
}
