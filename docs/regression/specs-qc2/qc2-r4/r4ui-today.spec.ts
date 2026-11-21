import { execFileSync } from "node:child_process";
import { expect, test, type Page, type Route } from "@playwright/test";
import { S, SEED_PW, apiLogin, call, clearRL, em, freshStudent, installRec, joinInfo, layout, newCourse, note, psql, techWords, tokenOf } from "./r4ui-lib";

// QC2-I r4ui: US-P2-11 (trang Hôm nay thật), US-P2-12 (7 tài khoản seed). Seed chỉ đọc; tài khoản qc2iu.* cho phần ghi.
installRec("r4ui-today");
const only = (info: { project: { name: string } }, p: "desktop" | "mobile") => test.skip(info.project.name !== p, `chỉ ${p}`);
const main = (page: Page) => page.locator("main");
const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
const seed = (u: string) => `${u}@edupilot.local`;
const loginSeed = (page: Page, u: string) => apiLogin(page, seed(u), SEED_PW);
const ready = async (page: Page) => {
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 40_000 });
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible({ timeout: 40_000 });
};

test("[TC-P211-32] / sinh viên (375): D có ô nhập mã 7 ký tự → /join/<mã>; B/A ≤ 1 button.primary, lời chào + ngày, một khuyến nghị, timeline, không thẻ số liệu", async ({ page }, info) => {
  only(info, "mobile");
  // D = sv.moi (seed, chưa có lớp)
  await loginSeed(page, "sv.moi");
  const previews: string[] = [];
  page.on("request", (r) => r.url().includes("/join/preview") && previews.push(r.postData() ?? ""));
  await page.goto("/");
  await expect(page.getByRole("heading", { name: /^Chào / })).toBeVisible({ timeout: 40_000 });
  const input = page.getByLabel("Mã tham gia lớp");
  await expect(input).toBeVisible();
  await expect(page.getByRole("button", { name: "Tiếp tục" })).toBeDisabled();
  await input.fill("an7 k2mq");
  await expect(page.getByRole("button", { name: "Tiếp tục" })).toBeEnabled();
  await page.getByRole("button", { name: "Tiếp tục" }).click();
  await expect.poll(() => previews.join("|"), { timeout: 20_000 }).toContain("AN7K2MQ");
  await expect(page).toHaveURL(/\/join(\/AN7K2MQ)?$/);
  note(`D (sv.moi): ô nhập mã có; nhập "an7 k2mq" → preview ${previews[0]}; URL ${new URL(page.url()).pathname}`);
  // B = sv.kha, A = sv.gioi
  for (const u of ["sv.kha", "sv.gioi"]) {
    await page.context().clearCookies();
    await loginSeed(page, u);
    await page.goto("/");
    await ready(page);
    await expect(page.getByRole("heading", { name: /^Chào / })).toBeVisible();
    await page.waitForTimeout(1500);
    const prim = await main(page).locator('[data-variant="primary"]:visible').count();
    const txt = (await main(page).innerText()).replace(/\s+/g, " ");
    const hasBuoi10 = /Buổi 10/.test(txt);
    const noUrgent = /Hôm nay bạn không có việc gấp\./.test(txt);
    const hasContinue = (await page.getByRole("heading", { name: "Tiếp tục học" }).count()) > 0;
    const kpi = await main(page).locator("[data-part*=kpi], [data-part*=stat], svg[role=img], canvas").count();
    const reco = /· \d+ phút|Khoảng \d+ phút|\d+ phút/.test(txt);
    note(`${u}: primary=${prim}; 'Buổi 10'=${hasBuoi10}; 'không có việc gấp'=${noUrgent}; Tiếp tục học=${hasContinue}; thẻ số liệu/biểu đồ=${kpi}; có thời lượng=${reco}; lời chào="${(await page.getByRole("heading", { name: /^Chào / }).innerText()).slice(0, 40)}"; ${txt.slice(0, 220)}`);
    expect(prim).toBeLessThanOrEqual(1);
    expect(kpi).toBe(0);
    expect(hasBuoi10, `${u} timeline có "Buổi 10"`).toBe(true);
  }
});

test("[TC-P211-33] / : khung xương aria-busy, lỗi chuẩn + Thử lại, không từ kỹ thuật, 375 AUDIT/TOUCH, hành động chính ≈ 3 s", async ({ page }, info) => {
  await apiLogin(page, S.gv1.email);
  if (info.project.name === "desktop") {
    // khung xương khi tải chậm
    await page.route(/\/api\/v1\/(me|courses\/[^/]+)\/today/, async (r) => {
      await new Promise((res) => setTimeout(res, 2500));
      return r.fallback();
    });
    await page.goto("/");
    await expect(page.locator("[aria-busy=true]").first()).toBeAttached({ timeout: 20_000 });
    note("khung xương: có phần tử [aria-busy=true] khi tải chậm 2,5 s");
    await page.unrouteAll({ behavior: "ignoreErrors" });
    // lỗi 500
    let n = 0;
    await page.route(/\/api\/v1\/(me|courses\/[^/]+)\/today/, (r) => (n++ < 2 ? json(r, 500, { code: "INTERNAL", message: "m", trace_id: "c".repeat(32) }) : r.fallback()));
    await page.goto("/");
    await expect(page.getByRole("alert").filter({ hasText: "Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." })).toBeVisible({ timeout: 40_000 });
    await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await page.getByRole("button", { name: "Thử lại" }).click();
    await expect(page.getByRole("alert").filter({ hasText: "Chưa tải được việc hôm nay" })).toHaveCount(0, { timeout: 30_000 });
    // thời gian tới hành động chính
    const t0 = Date.now();
    await page.goto("/");
    await main(page).locator('a[data-variant="primary"], button[data-variant="primary"]').first().waitFor({ state: "visible", timeout: 30_000 });
    const dt = Date.now() - t0;
    const lcp = await page.evaluate(() => new Promise<number>((res) => { new PerformanceObserver((l) => res(l.getEntries().at(-1)!.startTime)).observe({ type: "largest-contentful-paint", buffered: true }); setTimeout(() => res(-1), 1500); }));
    note(`hành động chính hiện sau ${dt} ms (goto→visible, máy tải nặng); LCP=${Math.round(lcp)} ms`);
    expect(await main(page).getByText(/RAG|PII|trace|provider/i).count()).toBe(0);
    expect(dt).toBeLessThan(10_000);
  } else {
    await page.goto("/");
    await ready(page);
    await main(page).locator("li[data-kind], [data-variant=primary]").first().waitFor({ state: "visible", timeout: 30_000 }).catch(() => undefined);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
    const l = await layout(page);
    note(`375 (GV): ox=${l.ox} cut=${l.cut.length} touch=${l.touch.length}`);
    expect(l.ox).toBeLessThanOrEqual(0);
    expect(l.cut).toEqual([]);
    expect(l.touch).toEqual([]);
    await page.context().clearCookies();
    await apiLogin(page, em("sv1"));
    await page.goto("/");
    await ready(page);
    const l2 = await layout(page);
    note(`375 (SV): ox=${l2.ox} cut=${l2.cut.length} touch=${l2.touch.length}`);
    expect(l2.ox).toBeLessThanOrEqual(0);
    expect(l2.touch).toEqual([]);
    expect(await main(page).getByText(/RAG|PII|trace|provider/i).count()).toBe(0);
  }
});

test("[TC-P211-35] / GV, TA, Admin: tiêu đề đếm việc, ActionList, COURSE_SETUP mở dần tại chỗ, Sắp tới, Lớp cần chú ý ẩn khi rỗng, Tất cả lớp; TA không thấy COURSE_SETUP/EMAIL_MISMATCH; Admin việc hệ thống", async ({ page }, info) => {
  only(info, "desktop");
  await newCourse(S.gv1.id, "T"); // bảo đảm có lớp mới chưa thiết lập → COURSE_SETUP
  await apiLogin(page, S.gv1.email);
  await page.goto("/");
  await ready(page);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/^(\d+ việc cần xử lý hôm nay|Không có việc cần xử lý hôm nay\.)$/, { timeout: 30_000 });
  await page.getByRole("button", { name: /^Chọn lớp, đang xem/ }).click();
  await page.getByRole("menuitem", { name: "Tất cả lớp của tôi" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(/^\d+ việc cần xử lý hôm nay$/, { timeout: 30_000 });
  const rows = main(page).locator("li[data-kind]");
  await expect(rows.first()).toBeVisible();
  const kinds = await rows.evaluateAll((els) => els.map((e) => e.getAttribute("data-kind")));
  const withClass = await rows.filter({ hasText: /lớp QC2IU/i }).count();
  note(`GV tất cả lớp: ${kinds.length} hàng, kind=${[...new Set(kinds)].join(",")}; hàng ghi 'lớp QC2IU…': ${withClass}; tiêu đề "${await page.getByRole("heading", { level: 1 }).innerText()}"`);
  expect(withClass).toBeGreaterThanOrEqual(1);
  const firstRow = (await rows.first().innerText()).split("\n").filter(Boolean);
  expect(firstRow.length).toBeGreaterThanOrEqual(2); // tiêu đề + lý do (+ hành động)
  const setup = main(page).locator('li[data-kind="COURSE_SETUP"]').first();
  await expect(setup).toBeVisible();
  await expect(setup.locator("ol li")).toHaveCount(0);
  await setup.getByRole("button", { name: /^Xem \d bước/ }).click();
  await expect(setup.locator("ol li")).toHaveCount(4);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  const attention = await main(page).getByRole("heading", { name: "Lớp cần chú ý" }).count();
  const upcoming = await main(page).getByRole("heading", { name: "Sắp tới" }).count();
  const t = await call("GET", `/courses/${S.courseA.id}/today`, await tokenOf(S.gv1.email));
  note(`'Lớp cần chú ý' hiện=${attention} (attention API=${(t.body.attention ?? []).length}); 'Sắp tới' hiện=${upcoming} (upcoming API=${(t.body.upcoming ?? []).length})`);
  if (!(t.body.attention ?? []).length) expect(attention).toBe(0);
  expect(await main(page).locator("[data-part*=kpi], [data-part*=hero]").count()).toBe(0);
  // TA
  await page.context().clearCookies();
  await apiLogin(page, S.ta.email);
  await page.goto("/");
  await ready(page);
  const taK = await main(page).locator("li[data-kind]").evaluateAll((els) => els.map((e) => e.getAttribute("data-kind")));
  note(`TA: kind=${[...new Set(taK)].join(",")}; h1="${await page.getByRole("heading", { level: 1 }).innerText()}"`);
  expect(taK).not.toContain("COURSE_SETUP");
  expect(taK).not.toContain("EMAIL_MISMATCH");
  const taApi = await call("GET", `/courses/${S.courseA.id}/today`, await tokenOf(S.ta.email));
  expect(JSON.stringify(taApi.body)).not.toMatch(/EMAIL_MISMATCH|COURSE_SETUP/);
  // Admin
  await page.context().clearCookies();
  await apiLogin(page, S.adm.email);
  await page.goto("/");
  await ready(page);
  const h = await page.getByRole("heading", { level: 1 }).innerText();
  const adK = await main(page).locator("li[data-kind]").evaluateAll((els) => els.map((e) => e.getAttribute("data-kind")));
  note(`Admin: h1="${h}"; kind=${[...new Set(adK)].join(",")}`);
  expect(h).toMatch(/^(\d+ việc cần xử lý hôm nay|Không có việc cần xử lý hôm nay\.)$/);
  expect(adK.every((k) => ["LLM_PROVIDER_ERROR", "LLM_BUDGET_EXHAUSTED", "LLM_BUDGET_WARN", "COURSE_NO_TEACHER", "INVITE_EXPIRED", "SYSTEM", "ESCALATION_OVERDUE"].includes(k ?? "") || /^[A-Z_]+$/.test(k ?? ""))).toBe(true);
});

test("[TC-P211-36] duyệt hết yêu cầu ở /class/members rồi về / : hàng JOIN_REQUEST biến ≤ 60 s", async ({ page }, info) => {
  only(info, "desktop");
  const C = await newCourse(S.gv1.id, "W");
  const cj = await joinInfo(C.id, S.gv1.email);
  expect((await call("PUT", `/courses/${C.id}/join-settings`, cj.token, { require_approval: true, version: cj.version })).status).toBe(200);
  const code = String((await joinInfo(C.id, S.gv1.email)).join_code);
  for (const tag of ["w1", "w2"]) {
    const e = await freshStudent(tag);
    expect((await call("POST", "/courses/join", await tokenOf(e), { code })).status).toBe(200);
  }
  await apiLogin(page, S.gv1.email);
  await page.goto(`/?course=${C.id}`);
  await ready(page);
  const row = main(page).locator('li[data-kind="JOIN_REQUEST"]');
  await expect(row).toBeVisible({ timeout: 40_000 });
  await row.getByRole("link", { name: "Xem yêu cầu" }).click();
  await expect(page).toHaveURL(/\/class\/members/);
  await expect(page.getByRole("heading", { name: "Thành viên lớp", level: 1 })).toBeVisible({ timeout: 30_000 });
  await page.getByRole("checkbox", { name: /Chọn tất cả/i }).check();
  await page.getByRole("button", { name: /^Duyệt 2/ }).click();
  await expect(page.getByRole("status").filter({ hasText: "Đã duyệt 2 yêu cầu" })).toBeVisible({ timeout: 30_000 });
  const t0 = Date.now();
  await page.getByRole("link", { name: "Hôm nay", exact: true }).first().click();
  await expect(page).toHaveURL(/\/(\?.*)?$/);
  await expect(main(page).locator('li[data-kind="JOIN_REQUEST"]')).toHaveCount(0, { timeout: 60_000 });
  const dt = (Date.now() - t0) / 1000;
  note(`hàng JOIN_REQUEST biến sau ${dt.toFixed(1)} s kể từ lúc quay về Hôm nay`);
  expect(dt).toBeLessThanOrEqual(60);
});

test("[TC-P211-38] / không dữ liệu mô phỏng cũ; grep mock = 0; việc của phase chưa làm không hiện; PROGRESS.md Nợ có Provider", async ({ page }, info) => {
  only(info, "desktop");
  const olds = ["Phiếu hỗ trợ", "QUIZ01", "Bài tập 03"];
  const roles: Array<[string, () => Promise<unknown>]> = [
    ["SV qc2iu.sv1", () => apiLogin(page, em("sv1"))],
    ["SV seed sv.gioi", () => loginSeed(page, "sv.gioi")],
    ["GV qc2iu.gv1", () => apiLogin(page, S.gv1.email)],
    ["GV seed teacher", () => loginSeed(page, "teacher")],
    ["TA qc2iu.ta", () => apiLogin(page, S.ta.email)],
    ["Admin qc2iu.adm", () => apiLogin(page, S.adm.email)],
  ];
  const out: string[] = [];
  for (const [label, login] of roles) {
    await page.context().clearCookies();
    await login();
    await page.goto("/");
    await ready(page);
    await page.waitForTimeout(1200);
    const txt = await main(page).innerText();
    for (const o of olds) expect(txt, `${label} chứa "${o}"`).not.toContain(o);
    expect(await main(page).getByText(/\bticket\b|Phiếu hỗ trợ|Chấm bài|Quiz/i).count(), `${label}: việc phase chưa làm`).toBe(0);
    out.push(label);
  }
  const root = __dirname + "/../../..";
  const g = execFileSync("bash", ["-c", "grep -rn 'mock' 'frontend/src/app/page.tsx' 'frontend/src/app/(app)' frontend/src/features/today 2>/dev/null | wc -l"], { cwd: root }).toString().trim();
  const debt = execFileSync("bash", ["-c", "grep -n 'Provider' docs/PROGRESS.md | head -3 | cut -c1-200"], { cwd: root }).toString().trim();
  note(`6 lượt đăng nhập (${out.join(", ")}) sạch 3 chuỗi cũ; grep 'mock' = ${g}; PROGRESS.md 'Provider': ${debt.slice(0, 160)}`);
  expect(g).toBe("0");
});

test("[TC-P211-41] /: 500 → lỗi chuẩn + Thử lại đúng 1 request; làm mới thất bại khi đã có dữ liệu giữ dữ liệu cũ; offline → OfflineBanner", async ({ page, context }, info) => {
  only(info, "desktop");
  await apiLogin(page, S.gv1.email);
  const todayRe = /\/api\/v1\/(me\/today|courses\/[^/]+\/today)/;
  let fail = true;
  const calls: number[] = [];
  await page.route(todayRe, (r) => { calls.push(Date.now()); return fail ? json(r, 500, { code: "INTERNAL", message: "m", trace_id: "c".repeat(32) }) : r.fallback(); });
  await page.goto("/");
  await expect(page.getByRole("alert").filter({ hasText: "Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." })).toBeVisible({ timeout: 40_000 });
  await page.waitForTimeout(1500);
  const before = calls.length;
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await page.waitForTimeout(1500);
  note(`Thử lại: ${calls.length - before} request /today`);
  expect(calls.length - before).toBe(1);
  // đã có dữ liệu: làm mới thất bại ⇒ dữ liệu cũ vẫn hiện
  const rowsBefore = await main(page).locator("li[data-kind]").count();
  expect(rowsBefore).toBeGreaterThan(0);
  fail = true;
  await page.evaluate(() => { document.dispatchEvent(new Event("visibilitychange")); window.dispatchEvent(new Event("focus")); });
  await page.waitForTimeout(3000);
  const rowsAfter = await main(page).locator("li[data-kind]").count();
  note(`làm mới 500: hàng trước ${rowsBefore}, sau ${rowsAfter}; request trong lúc đó ${calls.length}`);
  expect(rowsAfter).toBe(rowsBefore);
  fail = false;
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await context.setOffline(true);
  await expect(page.locator("[data-part=offline-banner]")).toBeVisible({ timeout: 20_000 });
  note("offline: OfflineBanner hiện: " + (await page.locator("[data-part=offline-banner]").innerText()).replace(/\s+/g, " ").slice(0, 80));
  await context.setOffline(false);
});

test("[TC-P212-06] 7 tài khoản seed: số mục nav SV 7 / TA 12 / GV 15 / Admin 6; D chỉ Hôm nay", async ({ page }, info) => {
  only(info, "desktop");
  const accts: Array<[string, string]> = [["sv.gioi", "SV"], ["sv.kha", "SV"], ["sv.nguyco", "SV"], ["sv.moi", "SV-D"], ["ta", "TA"], ["teacher", "GV"], ["admin", "Admin"]];
  const got: Record<string, number> = {};
  const labels: Record<string, string[]> = {};
  for (const [u, role] of accts) {
    await page.context().clearCookies();
    await loginSeed(page, u);
    await page.goto("/");
    await expect(page.locator("[data-part=sidebar] nav a").first()).toBeVisible({ timeout: 40_000 });
    await page.waitForTimeout(1500);
    got[u] = await page.locator("[data-part=sidebar] nav a").count();
    labels[role] = (await page.locator("[data-part=sidebar] nav a").allInnerTexts()).map((s) => s.trim());
    await page.getByRole("button", { name: /^Tài khoản: / }).click();
    expect(await page.getByText("Đổi vai").count()).toBe(0);
    await page.keyboard.press("Escape");
  }
  note(`nav: ${JSON.stringify(got)}; GV: ${labels.GV.join("/")}`);
  // chỉ ghi số đo; kết luận P/F/LT dựa vào US.md (xem ghi chú tổng hợp)
  expect(got["sv.moi"]).toBe(1);
});
