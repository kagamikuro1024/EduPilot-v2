import { execSync } from "node:child_process";
import { expect, type Page, test } from "@playwright/test";
import { call, courseId, login, needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — US-P10-05 / F17: màn /insights thật trên ep-qc (đã có ≥ 1 báo cáo ở lớp 761987). Các kiểm không chặn nhau (expect.soft).
needReal();

async function openInsights(page: Page, user = "teacher") {
  await uiLogin(page, user);
  await page.goto("/insights");
  await expect(page.getByRole("heading", { name: "Lỗ hổng kiến thức", level: 1 })).toBeVisible();
}
// lớp đang chọn ở thanh trên cùng (picker) — các kiểm API dùng đúng lớp màn đang hiển thị
const shownCode = async (page: Page) => (await page.getByRole("banner").innerText()).match(/\b(\d{6})\b/)?.[1] ?? "761987";

test("@real INS-1 đọc latest, cấu trúc màn, meta, Xuất (TC-05-02/04/05/09/10/16/17/18)", async ({ page, request }) => {
  const reads: string[] = [];
  page.on("request", (r) => { if (r.url().includes("/api/v1/courses/") && r.url().includes("/insights/")) reads.push(`${r.method()} ${r.url().split("/api/v1")[1]}`); });
  await openInsights(page);
  await expect(page.getByText("Chủ đề nên dạy lại", { exact: true })).toBeVisible();
  expect.soft(reads.every((r) => r.startsWith("GET ") && r.includes("/knowledge-gap/latest")), `chỉ GET latest: ${reads.join(" | ")}`).toBe(true);
  const tk = await login(request, "teacher");
  const cid = await courseId(request, "teacher", await shownCode(page));
  const lat = await call(request, tk, "GET", `/courses/${cid}/insights/knowledge-gap/latest`);
  const rep = (lat.json as { report: { payload: { topics: { name: string; few_asked?: boolean; examples: string[] }[]; uncovered: { text: string; count: number }[] } } }).report;
  for (const t of rep.payload.topics.slice(0, 3)) await expect.soft(page.getByText(t.name, { exact: false }).first(), `chủ đề «${t.name}» khớp payload`).toBeVisible();
  await shot(page, "ins-1440");
  const body = await page.locator("main").innerText();
  expect.soft(body.match(/Câu hỏi đã ẩn danh, dùng để cải thiện giảng dạy/g)?.length ?? 0, "TC-05-18 (P-11-128): không còn dòng meta").toBe(0);
  expect.soft(await page.getByRole("button", { name: "Tạo báo cáo mới" }).count(), "một nút Tạo báo cáo mới").toBe(1);
  const moreTopics = await page.getByText("Chủ đề khác").count();
  const shown = await page.locator("main").getByRole("button", { name: /Tạo thread ghim/ }).filter({ visible: true }).count();
  expect.soft(shown, `TC-05-04: ≤ 8 chủ đề ở khung nhìn đầu (có «Chủ đề khác»: ${moreTopics})`).toBeLessThanOrEqual(8);
  const others = page.getByText(/^Chủ đề khác/).first();
  if (await others.count()) await others.click();
  if (rep.payload.topics.some((t) => t.few_asked)) expect.soft(await page.getByText("Ít người hỏi").filter({ visible: true }).count(), "TC-05-05 nhãn «Ít người hỏi» hiện ở ít nhất một chủ đề").toBeGreaterThan(0);
  if (rep.payload.uncovered.length) {
    await expect.soft(page.getByText("Tài liệu chưa đề cập", { exact: true })).toBeVisible();
    for (const u of rep.payload.uncovered) await expect.soft(page.getByText(u.text)).toBeVisible();
  }
  await expect.soft(page.getByText(/^So với báo cáo ngày/)).toBeVisible();
  // Xuất Markdown / PDF
  const menu = page.getByRole("button", { name: /Thêm|Khác|Tuỳ chọn|Xuất/ }).first();
  await menu.click();
  const dl = page.waitForRequest((r) => r.url().includes("/export?format=md"), { timeout: 8000 }).catch(() => null);
  await page.getByRole("menuitem", { name: "Xuất Markdown" }).or(page.getByText("Xuất Markdown")).first().click();
  expect.soft(await dl, "TC-05-16: tải GET …/export?format=md").not.toBeNull();
  await menu.click();
  await page.evaluate(() => { (window as unknown as { __printed: number }).__printed = 0; window.print = () => { (window as unknown as { __printed: number }).__printed++; }; });
  const pdfReq: string[] = [];
  page.on("request", (r) => { if (/pdf/i.test(r.url())) pdfReq.push(r.url()); });
  await page.getByText("Xuất PDF").first().click();
  expect.soft(await page.evaluate(() => (window as unknown as { __printed: number }).__printed), "TC-05-17: mở hộp in (window.print)").toBe(1);
  expect.soft(pdfReq, "không tải PDF từ máy chủ").toEqual([]);
});

test("@real INS-2 thread ghim + buổi ôn tập (TC-05-11/12/14/15)", async ({ page, request }) => {
  await openInsights(page);
  const cid = await courseId(request, "teacher", await shownCode(page));
  const before = Number(sql(`select count(*) from forum_threads where course_id='${cid}' and pinned_at is not null`)[0][0]);
  await page.getByRole("button", { name: "Tạo thread ghim" }).first().click();
  const grp = page.getByRole("group", { name: "Tạo thread ghim" });
  await expect(grp.getByLabel("Tiêu đề")).toBeVisible();
  const title = `QC ghim ${Date.now().toString(36)}`;
  expect.soft(Number(sql(`select count(*) from forum_threads where course_id='${cid}' and pinned_at is not null`)[0][0]), "TC-05-11: mở nháp chưa tạo thread ghim nào").toBe(before);
  await grp.getByLabel("Tiêu đề").fill(title);
  const draft = await grp.getByLabel("Nội dung").inputValue();
  expect.soft(draft.length, "nháp có nội dung từ khuyến nghị").toBeGreaterThan(5);
  await grp.getByRole("button", { name: /Ghim|Đăng/ }).click();
  await expect.poll(() => sql(`select count(*) from forum_threads where course_id='${cid}' and title='${title}'`)[0][0], { timeout: 20_000 }).toBe("1");
  const th = sql(`select pinned_at is not null, (select email from users where id=author_id) from forum_threads where course_id='${cid}' and title='${title}'`)[0];
  expect.soft(th[0], "TC-05-12: thread ghim").toBe("t");
  expect.soft(th[1], "tác giả = GV").toBe("teacher@edupilot.local");
  const bodyDb = sql(`select body from forum_threads where course_id='${cid}' and title='${title}'`)[0][0];
  const lat = await call(request, await login(request, "teacher"), "GET", `/courses/${cid}/insights/knowledge-gap/latest`);
  const ex = ((lat.json as { report: { payload: { topics: { examples: string[] }[] } } }).report.payload.topics).flatMap((t) => t.examples);
  expect.soft(ex.filter((e) => bodyDb.includes(e)), "TC-05-12: nội dung không chứa câu hỏi mẫu").toEqual([]);
  const code = await shownCode(page);
  const kha = await login(request, code === "761988" ? "sv34" : "sv.kha");
  const list = await call(request, kha, "GET", `/courses/${cid}/threads?limit=50`);
  expect.soft(JSON.stringify(list.json).includes(title), "SV của lớp thấy thread").toBe(true);
  // buổi ôn tập
  await page.getByRole("button", { name: /Tạo buổi ôn tập/ }).first().click();
  const g2 = page.getByRole("group", { name: "Tạo buổi ôn tập" });
  await expect(g2.getByLabel("Tên buổi")).toBeVisible();
  expect.soft(await g2.getByLabel("Tên buổi").inputValue(), "TC-05-14: tiêu đề «Ôn tập: <chủ đề>»").toMatch(/^Ôn tập: /);
  const t0 = new Date(Date.now() + 3 * 86400_000);
  const f = (d: Date) => d.toISOString().slice(0, 16);
  await g2.getByLabel("Bắt đầu").fill(f(t0));
  await g2.getByLabel("Kết thúc").fill(f(new Date(t0.getTime() + 2 * 3600_000)));
  await g2.getByRole("button", { name: /Lưu|Tạo/ }).first().click();
  await expect.poll(() => sql(`select count(*) from calendar_events where course_id='${cid}' and title like 'Ôn tập:%'`)[0][0], { timeout: 20_000 }).not.toBe("0");
  expect.soft(sql(`select type from calendar_events where course_id='${cid}' and title like 'Ôn tập:%' order by created_at desc limit 1`)[0][0], "TC-05-14: type OTHER").toBe("OTHER");
  void before;
});

test("@real INS-3 quyền, lỗi, 375 px (TC-05-19/20/21)", async ({ page, browser }) => {
  for (const u of ["sv.kha", "admin"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: process.env.QC_BASE ?? "https://localhost:943" });
    const p = await ctx.newPage();
    await uiLogin(p, u);
    await p.goto("/insights");
    await p.waitForTimeout(1500);
    expect.soft(await p.getByRole("link", { name: "Lỗ hổng kiến thức" }).count(), `${u}: menu không có mục /insights`).toBe(0);
    expect.soft(await p.getByRole("button", { name: "Tạo báo cáo mới" }).count(), `${u}: không có nút tạo báo cáo`).toBe(0);
    await shot(p, `ins-direct-${u}`);
    await ctx.close();
  }
  await uiLogin(page, "teacher");
  let fail = true;
  await page.route("**/knowledge-gap/latest", (r) => (fail ? r.fulfill({ status: 500, json: { code: "INTERNAL", message: "x" } }) : r.continue()));
  await page.goto("/insights");
  const retry = page.getByRole("button", { name: /Thử lại/ }).first();
  await expect(retry).toBeVisible();
  await shot(page, "ins-error");
  fail = false;
  await retry.click();
  await expect(page.getByText("Chủ đề nên dạy lại", { exact: true })).toBeVisible();
  await page.setViewportSize({ width: 375, height: 812 });
  await page.reload();
  await expect(page.getByText("Chủ đề nên dạy lại", { exact: true })).toBeVisible();
  await noOverflow(page);
  const sm = await page.locator("main button, main summary, main [aria-expanded]").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && r.height < 44; }).map((e) => `${(e.textContent ?? "").trim().slice(0, 24)}:${Math.round(e.getBoundingClientRect().height)}`));
  await shot(page, "ins-375");
  expect.soft(sm, "TC-05-21: vùng chạm ≥ 44 px").toEqual([]);
});

test("@real INS-4 chọn kỳ (TC-04-04b)", async ({ page, request }) => {
  await openInsights(page);
  const labels = ["7 ngày", "14 ngày", "30 ngày", "Toàn kỳ"];
  const grp = page.getByRole("radiogroup").first();
  const opts = await grp.getByRole("radio").allInnerTexts();
  expect(opts.map((o) => o.trim()), "TC-04-04b: đúng bốn lựa chọn, không lựa chọn thứ năm").toEqual(labels);
  await expect(grp.getByRole("radio", { name: "14 ngày" }), "mặc định 14").toBeChecked();
  const tip = await grp.getByRole("radio", { name: "Toàn kỳ" }).evaluate((el) => el.getAttribute("title") ?? el.closest("[title]")?.getAttribute("title") ?? "");
  expect.soft(tip, "tooltip trên nhãn «Toàn kỳ»").toBe("120 ngày gần nhất");
  expect.soft(await page.locator("main svg[class*=info], main [aria-label*=Info]").count(), "không icon Info").toBe(0);
  await noOverflow(page, true);
  // «Toàn kỳ» → generate period_days = 120
  const cid = await courseId(request, "teacher", await shownCode(page));
  sql(`update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='${cid}'`);
  execSync("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del\"");
  let sent = "";
  await page.route("**/insights/knowledge-gap/generate", (r) => { sent = r.request().postData() ?? ""; return r.continue(); });
  await grp.getByRole("radio", { name: "Toàn kỳ" }).click();
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  await expect.poll(() => sent, { timeout: 15_000 }).toContain("120");
  expect.soft(sent, "generate gửi period_days=120").toMatch(/"period_days":\s*120/);
  await shot(page, "ins4-period");
  await page.setViewportSize({ width: 375, height: 812 });
  await noOverflow(page, true);
});

test("@real INS-5 nút nổi bật theo has_new_data (TC-05-06) + không dữ liệu mới (TC-05-08)", async ({ page, request }) => {
  await openInsights(page);
  const cid = await courseId(request, "teacher", await shownCode(page));
  const tk = await login(request, "teacher");
  const lat = await call(request, tk, "GET", `/courses/${cid}/insights/knowledge-gap/latest`);
  const has = (lat.json as { has_new_data: boolean }).has_new_data;
  const btn = page.getByRole("button", { name: "Tạo báo cáo mới" });
  await expect(btn).toHaveCount(1);
  const variant = await btn.getAttribute("data-variant");
  expect.soft(variant === "primary", `TC-05-06: has_new_data=${has} ↔ nút ${variant}`).toBe(has);
  await shot(page, `ins5-has-new-${has}`);
  if (!has) {
    const n0 = Number(sql(`select count(*) from insight_reports where course_id='${cid}'`)[0][0]);
    await btn.click();
    await expect(page.getByText(/Chưa có câu hỏi mới từ ngày/)).toBeVisible({ timeout: 20_000 });
    expect.soft(await page.getByText(/Chưa có câu hỏi mới từ ngày/).count(), "TC-05-08: đúng một dòng").toBe(1);
    expect.soft(Number(sql(`select count(*) from insight_reports where course_id='${cid}'`)[0][0]), "TC-05-08: không bản ghi mới").toBe(n0);
    await shot(page, "ins5-no-new-data");
  }
});

test("@real INS-6 job chạy nền (TC-05-07)", async ({ page, request }) => {
  await openInsights(page);
  const cid = await courseId(request, "teacher", await shownCode(page));
  const tk = await login(request, "teacher");
  const unread0 = ((await call(request, tk, "GET", "/notifications?limit=50")).json as { items: { id: string }[] }).items.map((n) => n.id);
  sql(`update insight_reports set data_watermark = data_watermark - interval '1 day' where course_id='${cid}'`);
  require_rl_clear();
  const stages: string[] = [];
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  for (let i = 0; i < 40; i++) {
    const t = await page.getByRole("status").allInnerTexts();
    for (const x of t) if (/^Đang /.test(x.trim()) && !stages.includes(x.trim())) stages.push(x.trim());
    if (stages.length) break;
    await page.waitForTimeout(250);
  }
  expect.soft(stages.length, `TC-05-07: có chỉ báo giai đoạn (${stages.join(" | ")})`).toBeGreaterThan(0);
  await page.goto("/threads");
  await expect(page).toHaveURL(/\/threads/);
  await page.goto("/insights");
  await expect(page.getByRole("heading", { name: "Lỗ hổng kiến thức", level: 1 })).toBeVisible();
  expect.soft(await page.getByText("Chủ đề nên dạy lại", { exact: true }).count(), "báo cáo cũ / mới vẫn xem được khi quay lại").toBeGreaterThan(0);
  await expect.poll(async () => ((await call(request, tk, "GET", "/notifications?limit=50")).json as { items: { id: string; type: string; title: string }[] }).items.filter((n) => !unread0.includes(n.id)).map((n) => `${n.type}:${n.title}`).join(" | "), { timeout: 120_000, intervals: [3_000] }).toMatch(/INSIGHT|báo cáo|Báo cáo|lỗ hổng/i);
  await shot(page, "ins6-after");
});

test("@real INS-7 POST threads lỗi 500 giữ nháp (TC-05-13)", async ({ page }) => {
  await openInsights(page);
  await page.route("**/api/v1/courses/*/threads", (r) => (r.request().method() === "POST" ? r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Lỗi hệ thống.", trace_id: "qc" }) }) : r.continue()));
  await page.getByRole("button", { name: "Tạo thread ghim" }).first().click();
  const grp = page.getByRole("group", { name: "Tạo thread ghim" });
  const title = `QC nháp ${Date.now().toString(36)}`;
  await grp.getByLabel("Tiêu đề").fill(title);
  const body = (await grp.getByLabel("Nội dung").inputValue()) + " (QC sửa)";
  await grp.getByLabel("Nội dung").fill(body);
  await grp.getByRole("button", { name: /Ghim|Đăng/ }).click();
  await page.waitForTimeout(1500);
  await expect(grp.getByLabel("Tiêu đề")).toHaveValue(title);
  expect.soft(await grp.getByLabel("Nội dung").inputValue(), "giữ nháp sau lỗi").toBe(body);
  expect.soft(await page.getByRole("alert").filter({ visible: true }).count() + (await page.getByText(/Chưa (gửi|tạo|ghim) được|Lỗi/).filter({ visible: true }).count()), "báo lỗi tại chỗ").toBeGreaterThan(0);
  await shot(page, "ins7-error");
  await page.unroute("**/api/v1/courses/*/threads");
  await page.reload();
  await expect(page.getByRole("heading", { name: "Lỗ hổng kiến thức", level: 1 })).toBeVisible();
  await page.getByRole("button", { name: "Tạo thread ghim" }).first().click();
  const g2 = page.getByRole("group", { name: "Tạo thread ghim" });
  await expect.soft(g2.getByLabel("Nội dung"), "TC-05-13: tải lại → nháp (nội dung) còn").toHaveValue(body);
  await expect.soft(g2.getByLabel("Tiêu đề"), "luật «không mất chữ đã gõ» (51c14cd8): tiêu đề cũng còn sau tải lại").toHaveValue(title);
});
function require_rl_clear() {
  // bộ đếm tạo báo cáo (3 việc / 60 phút / lớp) làm phép thử thất bại giả
  execSync("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:insight-generate:*' | xargs -r redis-cli del\"");
}
