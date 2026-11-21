import { expect, test, type Page, type Route } from "@playwright/test";
import { apiLogin, em, installRec, note } from "./r4ui-lib";

// SmkT3: TC-P202-31 (AC10): ≥ 5 request song song nhận 401 TOKEN_EXPIRED → ĐÚNG 1 POST /auth/refresh (tính từ lúc 401 đầu tiên; lần làm mới sớm lúc tải trang là việc khác),
// mỗi request phát lại đúng 1 lần; POST phát lại mang CÙNG Idempotency-Key. Ép 401 bằng page.route lên mọi request mang access token của lần làm mới lúc tải trang.
installRec("smkt3-p20231");
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
const ROUTES = ["/", "/documents", "/library", "/exams", "/assignments", "/attendance", "/calendar", "/threads", "/observability", "/settings/llm", "/questions", "/class/members", "/gradebook", "/inbox", "/insights", "/grading", "/analytics"];
const ADMIN_ROUTES = ["/admin/courses", "/admin/users", "/admin/audit", "/observability", "/settings/llm", "/settings/integrations", "/"];

type Hit = { k: string; t: number; idem?: string; forced: boolean };
async function scan(page: Page, path: string, armWith: (arm: () => void) => Promise<void>, armEarly = false) {
  const T0 = Date.now();
  const tl: string[] = [];
  const log = (s: string) => tl.push(`${String(Date.now() - T0).padStart(5)} ${s}`);
  const hits: Hit[] = [];
  const refresh: number[] = [];
  let armed = armEarly; // ép 401 ngay từ yêu cầu đầu sau lần làm mới lúc tải: các truy vấn của trang phát song song bằng cùng access token
  let stale: string | undefined;
  const seen = new Map<string, number>();
  await page.route(/\/api\/v1\/(?!auth\/)/, async (route) => {
    const r = route.request();
    const u = new URL(r.url());
    const k = `${r.method()} ${u.pathname}${u.search}`;
    if (!armed || u.pathname.endsWith("/events")) return route.fallback();
    const auth = r.headers()["authorization"] ?? "";
    if (stale === undefined && auth) stale = auth;
    const n = (seen.get(k) ?? 0) + 1;
    seen.set(k, n);
    // /me/courses cho qua: các request theo lớp chỉ phát đi sau khi có danh sách lớp, cùng lúc với request bị ép 401
    const force = n === 1 && auth === stale && !u.pathname.endsWith("/me/courses");
    log(`${force ? "401!" : "pass"} ${k.slice(0, 64)} n=${n}`);
    hits.push({ k, t: Date.now(), idem: r.headers()["idempotency-key"], forced: force });
    if (force) return json(route, 401, { code: "TOKEN_EXPIRED", message: "m", trace_id: "b".repeat(32) });
    return route.fallback();
  });
  const onReq = (r: { url(): string }) => r.url().endsWith("/auth/refresh") && (refresh.push(Date.now()), log("REFRESH →"));
  page.on("request", onReq);
  await page.goto(path);
  await page.waitForTimeout(3500);
  await armWith(() => { armed = true; log("--- armed"); });
  // làm mới đồng loạt: sự kiện focus / visibilitychange khiến mọi truy vấn đang mở của trang refetch cùng lúc (React Query refetchOnWindowFocus)
  await page.evaluate(() => { window.dispatchEvent(new Event("focus")); document.dispatchEvent(new Event("visibilitychange")); });
  await page.waitForTimeout(6000);
  page.off("request", onReq);
  await page.unroute(/\/api\/v1\/(?!auth\/)/);
  const forced = hits.filter((h) => h.forced);
  const t401 = forced.length ? Math.min(...forced.map((h) => h.t)) : Infinity;
  const afterFirst = refresh.filter((t) => t > t401).length;
  const replays = forced.map((h) => seen.get(h.k));
  return { tl, forced, afterFirst, refreshAll: refresh.length, replays, seen };
}

test("[TC-P202-31] ≥ 5 request song song 401 TOKEN_EXPIRED → đúng 1 refresh, mỗi cái phát lại đúng 1 lần, POST cùng Idempotency-Key", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(420_000);
  let best: Awaited<ReturnType<typeof scan>> | null = null;
  let route = "";
  outer: for (const [who, routes] of [["gv1", ROUTES], ["adm", ADMIN_ROUTES]] as const) {
    await apiLogin(page, em(who));
    for (const path of routes) {
      const r = await scan(page, path, async (arm) => arm(), true);
      console.log(`ROUTE ${who} ${path}: ép ${r.forced.length}, refresh sau 401 đầu ${r.afterFirst}, tổng refresh ${r.refreshAll}\n${r.tl.join("\n")}`);
      if (!best || r.forced.length > best.forced.length) { best = r; route = `${who} ${path}`; }
      if (r.forced.length >= 5) break outer;
    }
  }
  const b = best!;
  note(`${route}: ép 401 TOKEN_EXPIRED ${b.forced.length} request song song (${b.forced.map((h) => h.k.replace("/api/v1", "").slice(0, 30)).join("; ")}); POST /auth/refresh sau 401 đầu = ${b.afterFirst} (tổng kể cả lần làm mới sớm lúc tải trang = ${b.refreshAll}); số lần mỗi request tới mạng (1 bị ép + 1 phát lại = 2) = [${b.replays.join(",")}]`);
  expect(b.forced.length).toBeGreaterThanOrEqual(5);
  expect(b.afterFirst).toBe(1);
  for (const n of b.replays) expect(n).toBe(2);
});

test("[TC-P202-31] POST bị 401 một lần rồi thành công: phát lại mang cùng Idempotency-Key, đúng 1 bản ghi", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  await apiLogin(page, em("adm"));
  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Người dùng", level: 1 })).toBeVisible({ timeout: 30_000 });
  const posts: Array<{ idem?: string; forced: boolean }> = [];
  let stale: string | undefined;
  const refresh: number[] = [];
  page.on("request", (r) => r.url().endsWith("/auth/refresh") && refresh.push(Date.now()));
  await page.route("**/api/v1/admin/users", async (route) => {
    const r = route.request();
    if (r.method() !== "POST") return route.fallback();
    const auth = r.headers()["authorization"];
    if (stale === undefined) stale = auth;
    const forced = posts.length === 0;
    posts.push({ idem: r.headers()["idempotency-key"], forced });
    if (forced) return json(route, 401, { code: "TOKEN_EXPIRED", message: "m", trace_id: "b".repeat(32) });
    return route.fallback();
  });
  await page.getByRole("button", { name: "Mời giảng viên" }).click();
  const email = `qc2iu.inv${Date.now()}@edupilot.local`;
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Họ và tên").fill("QC2IU Mời Thử");
  await page.getByRole("button", { name: "Gửi lời mời" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã gửi link mời, hạn 72 giờ." })).toBeVisible({ timeout: 20_000 });
  note(`POST /admin/users: ${posts.length} lần (lần 1 bị ép 401), khoá ${posts.map((p) => p.idem?.slice(0, 12)).join(" / ")}; POST /auth/refresh = ${refresh.length}`);
  expect(posts.length).toBe(2);
  expect(posts[0].idem).toBeTruthy();
  expect(posts[0].idem).toBe(posts[1].idem);
  expect(refresh.length).toBe(1);
});
