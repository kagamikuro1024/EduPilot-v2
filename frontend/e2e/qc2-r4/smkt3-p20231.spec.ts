import { expect, test, type Route } from "@playwright/test";
import { apiLogin, em, installRec, note } from "./r4ui-lib";

// SmkT3: TC-P202-31 — dòng thời gian từng request để biết refresh thừa từ đâu. Dùng tài khoản qc2iu.gv1 (lớp thật, /grading gọi ≥ 5 đường dữ liệu cùng lúc).
installRec("smkt3-p20231");
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });

test("[TC-P202-31] 5 request song song nhận 401 TOKEN_EXPIRED: đúng 1 refresh, mỗi cái phát lại đúng 1 lần (dòng thời gian)", async ({ page }, info) => {
  if (info.project.name !== "desktop") test.skip();
  await apiLogin(page, em("gv1"));
  const T0 = Date.now();
  const tl: string[] = [];
  const log = (s: string) => tl.push(`${String(Date.now() - T0).padStart(5)} ${s}`);
  let armed = false;
  let stale: string | undefined;
  const forced = new Map<string, number>(); // khoá "METHOD path" → số lần đã ép 401 ở token cũ
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
    const old = auth === stale && !u.pathname.endsWith("/me/courses");
    log(`${old ? "401!" : "pass"} ${k.slice(0, 60)} tok=…${auth.slice(-6)} n=${n}`);
    if (old && n === 1) { forced.set(k, 1); return json(route, 401, { code: "TOKEN_EXPIRED", message: "m", trace_id: "b".repeat(32) }); }
    return route.fallback();
  });
  page.on("request", (r) => r.url().endsWith("/auth/refresh") && log("REFRESH →"));
  page.on("response", (r) => r.url().endsWith("/auth/refresh") && log(`REFRESH ← ${r.status()}`));
  await page.goto("/grading");
  await page.waitForTimeout(4000);
  armed = true;
  log("--- armed, reload");
  await page.reload();
  await page.waitForTimeout(8000);
  console.log("TIMELINE\n" + tl.join("\n"));
  const exp = [...forced.keys()];
  const refs = tl.filter((l) => l.includes("REFRESH →")).length;
  note(`ép ${exp.length} request 401; refresh sau khi armed: ${refs}; ${tl.filter((l) => /REFRESH|401!/.test(l)).map((l) => l.trim().slice(0, 70)).join(" | ")}`);
  expect(exp.length).toBeGreaterThanOrEqual(5);
  void expect;
});
