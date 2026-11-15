import { expect, type Page, test } from "@playwright/test";
import { call, courseId, login, needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — US-P10-06 /analytics tab «Điểm thi» (TC-06-01/09/10/19/20/23) trên ep-qc thật.
needReal();

async function openExams(page: Page, courseQuery = "") {
  await page.goto(`/analytics${courseQuery}`);
  const tab = page.getByRole("tab", { name: "Điểm thi" }).or(page.getByRole("button", { name: "Điểm thi" })).first();
  await expect(tab).toBeVisible();
  await tab.click();
}

test("@real AN-1 bài mới nhất, histogram, một dòng chữ, xu hướng, bảng câu sai (TC-06-01)", async ({ page, request }) => {
  await uiLogin(page, "teacher");
  await openExams(page);
  await expect(page.getByRole("figure", { name: "Phân bố điểm" })).toBeVisible({ timeout: 30_000 });
  await expect(page.getByText(/Trung bình [\d,.]+ · trung vị [\d,.]+ · đạt [\d,.]+ ?% · \d+ bài nộp/)).toBeVisible();
  await expect(page.getByText("Xu hướng qua các bài thi")).toBeVisible();
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" })).toBeVisible({ timeout: 30_000 });
  await shot(page, "an1-1440");
  const cid = await courseId(request, "teacher", (await page.getByRole("banner").innerText()).match(/\b(\d{6})\b/)?.[1] ?? "761987");
  const latest = sql(`select title from exams where course_id='${cid}' and status='PUBLISHED' order by published_at desc nulls last limit 1`)[0]?.[0];
  const body = await page.locator("main").innerText();
  expect.soft(latest ? body.includes(latest) : true, `TC-06-01: mặc định chọn bài PUBLISHED mới nhất «${latest}»`).toBe(true);
  const kpiCards = await page.locator("main [class*=kpi], main [data-part=kpi]").count();
  expect.soft(kpiCards, "không tường thẻ KPI").toBe(0);
});

test("@real AN-2 liên kết «Xem ở Báo cáo lỗ hổng» theo chủ đề trùng báo cáo (TC-06-09/10)", async ({ page, request }) => {
  await uiLogin(page, "teacher");
  await openExams(page);
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" })).toBeVisible({ timeout: 30_000 });
  const code = (await page.getByRole("banner").innerText()).match(/\b(\d{6})\b/)?.[1] ?? "761987";
  const cid = await courseId(request, "teacher", code);
  const tk = await login(request, "teacher");
  const rep = await call(request, tk, "GET", `/courses/${cid}/insights/knowledge-gap/latest`);
  const names: string[] = rep.status === 200 ? ((rep.json as { report: { payload: { topics: { name: string }[] } } }).report.payload.topics.map((t) => t.name.toLowerCase())) : [];
  const rows = page.getByRole("table", { name: "Câu sai nhiều nhất" }).locator("tbody tr");
  const n = await rows.count();
  let expectLinks = 0;
  for (let i = 0; i < n; i++) {
    const txt = (await rows.nth(i).innerText()).toLowerCase();
    if (names.some((nm) => txt.includes(nm))) expectLinks++;
  }
  const links = await page.getByRole("link", { name: "Xem ở Báo cáo lỗ hổng" }).count();
  expect.soft(links, `TC-06-09: ${links} liên kết = ${expectLinks} hàng có chủ đề trùng (${n} hàng, ${names.length} chủ đề báo cáo)`).toBe(expectLinks);
  await shot(page, "an2-links");
  // TC-06-10: lớp chưa có báo cáo → không liên kết (QCT3 mới tạo, GV qct2)
  const ctx = await page.context().browser()!.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const p2 = await ctx.newPage();
  await uiLogin(p2, "qct2");
  const c3 = sql("select id from courses where class_code='QCT3'")[0][0];
  await openExams(p2, `?course=${c3}`);
  await p2.waitForTimeout(2500);
  expect.soft(await p2.getByRole("link", { name: "Xem ở Báo cáo lỗ hổng" }).count(), "TC-06-10: lớp chưa có báo cáo — không liên kết").toBe(0);
  await ctx.close();
});

test("@real AN-3 lớp mới không bài công bố → EmptyState, API rỗng có cấu trúc (TC-06-19)", async ({ page, request }) => {
  const c3 = sql("select id from courses where class_code='QCT3'")[0][0];
  const tk = await login(request, "qct2");
  for (const p of ["overview", "trend", "hard-items", "declining"]) {
    const r = await call(request, tk, "GET", `/courses/${c3}/analytics/exams/${p}`);
    expect.soft(r.status, `TC-06-19: API ${p} → ${r.status}`).toBe(200);
  }
  await uiLogin(page, "qct2");
  await openExams(page, `?course=${c3}`);
  await expect(page.getByText(/Chưa có bài thi/).first()).toBeVisible({ timeout: 30_000 });
  expect.soft(await page.getByText(/Chưa có bài thi/).count(), "một dòng EmptyState").toBeLessThanOrEqual(2);
  await shot(page, "an3-empty");
});

test("@real AN-4 overview 500 → PageState + Thử lại (TC-06-20)", async ({ page }) => {
  await uiLogin(page, "teacher");
  let n = 0;
  await page.route("**/analytics/exams/overview*", (r) => { n++; return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi hệ thống.", trace_id: "qc" }) }); });
  await openExams(page);
  const retry = page.getByRole("button", { name: "Thử lại" }).first();
  await expect(retry).toBeVisible({ timeout: 30_000 });
  await shot(page, "an4-error");
  const before = n;
  await page.unroute("**/analytics/exams/overview*");
  await retry.click();
  await expect(page.getByRole("figure", { name: "Phân bố điểm" })).toBeVisible({ timeout: 30_000 });
  expect.soft(before, "đã gọi overview ít nhất một lần").toBeGreaterThan(0);
});

test("@real AN-5 375 px: biểu đồ co, bảng cuộn trong khung, không tràn (TC-06-23)", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await uiLogin(page, "teacher");
  await openExams(page);
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" })).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(1500);
  await shot(page, "an5-375");
  await noOverflow(page, true);
  const fig = await page.getByRole("figure", { name: "Phân bố điểm" }).boundingBox();
  expect.soft(fig && fig.x >= -1 && fig.x + fig.width <= 376, `histogram nằm trong 375 px (${JSON.stringify(fig)})`).toBe(true);
});
