// QC 10.8 mobile pha 1 (qc, cặp 3): tiện ích chung cho qc-mob-pha1.spec.ts.
// Chạy trên VPS (ep-vps-test) trỏ HTTPS thật; Chromium giả thiết bị (không WebKit).
import { appendFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { devices, type Browser, type BrowserContext, type Page } from "@playwright/test";

const cfg = JSON.parse(readFileSync(".qc-mob.json", "utf8")) as Record<string, string>;
export const BASE = cfg.base;
export const PW = cfg.pw;
const PASS = (cfg.pass ?? {}) as Record<string, string>;
/** Mật khẩu riêng của tài khoản QC (sv.qc*); tài khoản seed dùng mật khẩu chung. */
export const pwFor = (user: string) => PASS[user.split("@")[0]] ?? PW;
export const ACCT = { s1: "sv.qc" }; // sv.qc2/sv.qc2b thuộc qc2 — không dùng
export const C = { QCMOB1: cfg.QCMOB1, QCMOB2: cfg.QCMOB2, QCMOB4B: cfg.QCMOB4B, QCMOB5: cfg.QCMOB5, QCMOB6: cfg.QCMOB6 };
export const OUT = process.env.QC_OUT ?? "/tmp/qc-mob";
mkdirSync(OUT, { recursive: true });
const LOG = `${OUT}/run.jsonl`;
const TOKF = `${OUT}/tok.json`;
/** Hạn mức đăng nhập của gateway theo IP: chờ rồi thử lại thay vì đổ cả lô test. */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export function rec(tc: string, st: "PASS" | "FAIL" | "NOTE", note = "") {
  const line = `${st} ${tc} ${note}`;
  console.log(line.slice(0, 320));
  appendFileSync(LOG, JSON.stringify({ tc, st, note }) + "\n");
}
export function check(tc: string, ok: boolean, note = "") {
  rec(tc, ok ? "PASS" : "FAIL", note);
  return ok;
}

/** Thiết bị giả: Pixel 7 (412×915) và iPhone 15 (393×852) trên Chromium. */
const { defaultBrowserType: _p, ...PIXEL } = devices["Pixel 7"] as Record<string, unknown>;
const { defaultBrowserType: _i, ...IPHONE } = devices["iPhone 15"] as Record<string, unknown>;
export const DEV = { pixel: PIXEL, iphone: IPHONE } as Record<string, Record<string, unknown>>;

// ---------- API ----------
const tk: Record<string, { t: string; exp: number }> = existsSync(TOKF) ? JSON.parse(readFileSync(TOKF, "utf8")) : {};
const saveTk = () => writeFileSync(TOKF, JSON.stringify(tk), { mode: 0o600 });
/** Đăng nhập API có chờ khi bị 429 (hạn mức theo IP); token nhớ qua các lần chạy trong 10 phút. */
export async function token(user: string) {
  const email = user.includes("@") ? user : `${user}@edupilot.local`;
  if (tk[email] && tk[email].exp > Date.now() + 60_000) return tk[email].t;
  for (let i = 0; i < 8; i++) {
    const r = await fetch(`${BASE}/api/v1/auth/login`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ email, password: pwFor(email) }),
    });
    if (r.status === 429) {
      await sleep(15_000);
      continue;
    }
    if (!r.ok) throw new Error(`login ${email} → ${r.status}`);
    tk[email] = { t: ((await r.json()) as { access_token: string }).access_token, exp: Date.now() + 10 * 60_000 };
    saveTk();
    return tk[email].t;
  }
  throw new Error(`login ${email} → 429 (hết lượt chờ)`);
}
/** Gọi thẳng gateway (chỉ tạo dữ liệu QC, không đọc PII người khác). */
export async function api(user: string, method: string, path: string, body?: unknown) {
  const h: Record<string, string> = { "content-type": "application/json", authorization: `Bearer ${await token(user)}` };
  if (method === "POST") h["Idempotency-Key"] = crypto.randomUUID();
  const r = await fetch(`${BASE}/api/v1${path}`, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(20_000) });
  const t = await r.text();
  return { s: r.status, j: t ? JSON.parse(t) : null };
}

export type Win = { sid: string; wid: string; qr: string; code: string; round: number };
async function tryOpen(course: string, sid: string, minutes: number): Promise<Win | null> {
  let o = await api("teacher", "POST", `/courses/${course}/sessions/${sid}/checkin-windows`, { minutes });
  if (o.s === 409 && o.j?.code === "WINDOW_ALREADY_OPEN") {
    const l = await api("teacher", "GET", `/courses/${course}/sessions/${sid}/checkin-windows`);
    for (const w of (l.j?.items ?? []) as { id: string; closes_at?: string; closed_at?: string | null; status?: string }[]) {
      if (!w.closed_at && w.status !== "CLOSED") await api("teacher", "POST", `/courses/${course}/checkin-windows/${w.id}/close`);
    }
    o = await api("teacher", "POST", `/courses/${course}/sessions/${sid}/checkin-windows`, { minutes });
  }
  if (o.s !== 201) return null; // ROUND_LIMIT (3 vòng/buổi) → thử buổi khác
  const wid = o.j.id as string;
  const d = await api("teacher", "GET", `/courses/${course}/sessions/${sid}/checkin-windows/${wid}/display`);
  return { sid, wid, qr: (d.j.qr_url as string).split("#")[1], code: (d.j.short_code as string).replace(/\s/g, ""), round: o.j.round as number };
}
/** Mở cửa sổ ở buổi `only` (nếu có), không thì buổi đang diễn ra rồi tới buổi có vòng trống. Mỗi buổi tối đa 3 vòng. */
export async function openWindow(course: string, minutes = 10, only?: string): Promise<Win> {
  if (only) {
    const w = await tryOpen(course, only, minutes);
    if (w) return w;
    throw new Error(`buổi ${only} hết vòng (ROUND_LIMIT)`);
  }
  const ss = await api("teacher", "GET", `/courses/${course}/sessions?limit=100`);
  const items = ((ss.j?.items ?? []) as { id: string; starts_at: string; ends_at: string }[]).sort(
    (a, b) => Date.parse(a.starts_at) - Date.parse(b.starts_at),
  );
  const now = Date.now();
  const inside = items.find((s) => Date.parse(s.starts_at) <= now && now <= Date.parse(s.ends_at));
  const order = [...(inside ? [inside] : []), ...items.filter((s) => s !== inside)];
  for (const s of order) {
    const w = await tryOpen(course, s.id, minutes);
    if (w) return w;
  }
  throw new Error(`hết vòng điểm danh ở mọi buổi của ${course}`);
}
export const closeWindow = (course: string, wid: string) => api("teacher", "POST", `/courses/${course}/checkin-windows/${wid}/close`);

/** Số lượt điểm danh của một SV trong một cửa sổ (SQL qua API GV). */
export async function rows(course: string, sid: string): Promise<{ uid: string; at: string; channel?: string; flags?: string[] }[]> {
  const r = await api("teacher", "GET", `/courses/${course}/sessions/${sid}/attendance`);
  const arr = (r.j?.items ?? r.j?.students ?? []) as Record<string, unknown>[];
  return arr.map((x) => ({
    uid: String(x.user_id ?? x.id ?? ""),
    at: String(x.checked_in_at ?? x.status ?? ""),
    channel: x.channel as string | undefined,
    flags: (x.flags ?? x.signals) as string[] | undefined,
  }));
}

// ---------- Trình duyệt ----------
/** Đăng nhập bằng giao diện (đo thời gian, kiểm phiên) — tốn một lượt hạn mức đăng nhập. */
export async function login(page: Page, user: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(pwFor(user));
  await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
}

/** Context đã có phiên: đăng nhập bằng API rồi cấy cookie làm mới `ep_rt` (không tốn hạn mức đăng nhập giao diện). */
const pool = new Map<string, { ctx: BrowserContext; page: Page }>();
/** Một context/một phiên cho mỗi tài khoản trong cả tệp (cookie làm mới xoay vòng: dùng lại storageState sẽ 401). */
export async function authed(b: Browser, dev: "pixel" | "iphone", user: string, opts: Record<string, unknown> = {}) {
  const hit = pool.get(user);
  if (hit) return hit;
  const c = await b.newContext({ ...DEV[dev], baseURL: BASE, locale: "vi-VN", timezoneId: "Asia/Ho_Chi_Minh", ignoreHTTPSErrors: true, ...opts } as never);
  const page = await c.newPage();
  for (let i = 0; i < 6; i++) {
    try {
      await login(page, user);
      break;
    } catch (e) {
      if (i === 5) throw e;
      await sleep(20_000); // 429 hạn mức đăng nhập theo IP
    }
  }
  const v = { ctx: c, page };
  pool.set(user, v);
  return v;
}
/** Đăng nhập lại từ đầu cho một tài khoản (phiên cũ bị thu hồi khi đăng nhập ở context khác). */
export async function relogin(b: Browser, dev: "pixel" | "iphone", user: string) {
  const hit = pool.get(user);
  if (hit) {
    pool.delete(user);
    await hit.ctx.close().catch(() => undefined);
  }
  return authed(b, dev, user);
}

/** Đóng mọi context cuối tệp. */
export async function closeAll() {
  for (const { ctx } of pool.values()) await ctx.close().catch(() => undefined);
  pool.clear();
}

/** Mở /checkin?course=<id> và CHỜ deep link bỏ tham số (router.replace làm mất chữ đã gõ nếu gõ sớm). */
export async function gotoEntry(p: Page, course: string, classCode?: string) {
  await p.goto(`/checkin?course=${course}`);
  await p.waitForFunction(() => !new URLSearchParams(location.search).has("course"), null, { timeout: 30_000 });
  await p.getByLabel("Mã điểm danh").waitFor({ timeout: 20_000 });
  if (classCode) await p.getByText(`Lớp ${classCode}`, { exact: false }).waitFor({ timeout: 10_000 });
}

export const noOverflow = (p: Page) => p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1);
