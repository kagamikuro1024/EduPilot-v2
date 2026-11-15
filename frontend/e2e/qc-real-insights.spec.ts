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
  expect.soft(body.match(/Câu hỏi đã ẩn danh, dùng để cải thiện giảng dạy/g)?.length ?? 0, "TC-05-18: đúng một dòng meta").toBe(1);
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
