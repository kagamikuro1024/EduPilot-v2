import { expect, test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");

// QC FEAT-uiux-91 đợt 4, lượt kiểm lại (tc-wave-4.md): UX12-11/14/19, UX16-1/5/21 ở 375 / 1440 trên stack ep-qc.
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";
async function open(browser: import("@playwright/test").Browser, w: number, h: number) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: h }, hasTouch: w < 500, isMobile: w < 500, locale: "vi-VN" });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30_000 });
  await page.locator("[data-part=topbar]").waitFor({ timeout: 15_000 });
  await page.waitForTimeout(1200);
  const ids = await page.evaluate(async () => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const r = await (await fetch("/api/v1/me/courses", { headers: { authorization: `Bearer ${a.access_token}` } })).json();
    return Object.fromEntries(r.items.map((i: { course: { class_code: string; id: string } }) => [i.course.class_code, i.course.id]));
  }) as Record<string, string>;
  return { ctx, page, ids };
}
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
const bar = async (page: Page) => (await page.getByRole("button", { name: /Lưu điểm danh|Đã lưu/ }).first().innerText()).replace(/\s+/g, " ");

test("TC-UX12-11/19 điểm danh 375: chạm 2 SV sang Vắng, lưu ≤ 60 s, không hộp thoại", async ({ browser }, info) => {
  desktopOnly(info);
  test.setTimeout(150_000);
  const { ctx, page, ids } = await open(browser, 375, 812);
  page.setDefaultTimeout(10_000);
  const dialogs: string[] = [];
  page.on("dialog", (d) => { dialogs.push(d.message()); void d.dismiss(); });
  await page.goto(`/attendance?course=${ids["761987"]}`);
  await page.waitForTimeout(4500);
  const names = await page.locator("main tbody tr[data-part=student-row] td:first-child span span:first-child").allInnerTexts();
  const pick = [names[18], names[19]];
  console.log("INFO chọn:", pick.join(", "), "| thanh trước:", await bar(page));
  const v0 = Number(((await bar(page)).match(/(\d+) vắng/) ?? [])[1] ?? 0); // dữ liệu giữ qua các lượt: đếm theo số ban đầu
  const t0 = Date.now();
  for (const n of pick) {
    await page.getByRole("button", { name: new RegExp(n) }).first().click();
    await page.locator("label:visible, button:visible").filter({ hasText: /^Vắng$/ }).first().click();
    await page.waitForTimeout(400);
  }
  const mid = await bar(page);
  console.log("INFO thanh sau 2 lần chạm:", mid);
  expect(mid, "số vắng cập nhật").toContain(`${v0 + 2} vắng`);
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  await page.waitForTimeout(2000);
  const after = await bar(page);
  const secs = Math.round((Date.now() - t0) / 1000);
  console.log("INFO sau lưu:", after, "| hộp thoại:", await page.getByRole("dialog").count(), "| confirm():", dialogs.length, "|", secs, "s");
  expect(after).toMatch(/Đã lưu/);
  expect(await page.getByRole("dialog").count()).toBe(0);
  expect(dialogs.length).toBe(0);
  expect(secs).toBeLessThan(60);
  for (const n of pick) { // trả lại
    await page.getByRole("button", { name: new RegExp(n) }).first().click();
    await page.locator("label:visible, button:visible").filter({ hasText: /^Có mặt$/ }).first().click();
    await page.waitForTimeout(300);
  }
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  await page.waitForTimeout(1500);
  await ctx.close();
});

test("TC-UX12-14/15 cảnh báo công thức 375 (lớp 761988): một khối ≤ 56 px", async ({ browser }, info) => {
  desktopOnly(info);
  const { ctx, page, ids } = await open(browser, 375, 812);
  await page.goto(`/attendance?course=${ids["761988"]}`);
  await page.waitForTimeout(4500);
  const w = await page.evaluate(() => {
    const t = [...document.querySelectorAll("main *")].find((e) => e.children.length === 0 && /chưa xác nhận/.test(e.textContent ?? ""));
    let n = t?.parentElement ?? null;
    while (n && n.tagName !== "MAIN" && n.querySelectorAll("button, a").length === 0) n = n.parentElement; // khối nhỏ nhất chứa chữ và nút
    return n ? { h: Math.round(n.getBoundingClientRect().height), t: (n as HTMLElement).innerText.replace(/\s+/g, " ") } : null;
  });
  console.log("INFO khối cảnh báo 375:", JSON.stringify(w));
  expect(w, "có cảnh báo + Xem công thức").not.toBeNull();
  expect(w!.h).toBeLessThanOrEqual(56);
  await ctx.close();
});

test("TC-UX16-1/5 sổ điểm 1440: ô nhập đệm 0 khi không focus, giá trị thô khi focus; ⋯ tiêu đề chỉ hiện khi rê / focus", async ({ browser }, info) => {
  desktopOnly(info);
  const { ctx, page, ids } = await open(browser, 1440, 900);
  await page.goto(`/gradebook?course=${ids["761987"]}`);
  await page.waitForTimeout(5000);
  const vals = await page.locator("main tbody tr input").evaluateAll((es) => es.map((e) => (e as HTMLInputElement).value).filter((v) => v !== ""));
  console.log("INFO ô nhập (không focus):", JSON.stringify([...new Set(vals)]));
  expect(vals.length).toBeGreaterThan(0);
  for (const v of vals) expect(v, "ít nhất 2 chữ số thập phân, dấu phẩy").toMatch(/^\d+,\d{2,}$/);
  const first = page.locator("main tbody tr input").first();
  await first.focus();
  console.log("INFO ô nhập focus:", await first.inputValue());
  await page.keyboard.press("Escape");
  await page.mouse.move(5, 5);
  const menus = page.locator("main thead th button");
  const n = await menus.count();
  expect(n, "có nút ⋯ ở tiêu đề cột nhập").toBeGreaterThan(0);
  const op = async () => menus.evaluateAll((es) => es.map((e) => Number(getComputedStyle(e).opacity)));
  const before = await op();
  const th = page.locator("main thead th:has(button)").first();
  const box = (await th.boundingBox())!;
  await page.mouse.move(box.x + 20, box.y + 20);
  await page.waitForTimeout(500);
  const hover = await menus.first().evaluate((e) => Number(getComputedStyle(e).opacity));
  await page.mouse.move(5, 5);
  await menus.first().focus();
  await page.waitForTimeout(400);
  const focus = await menus.first().evaluate((e) => Number(getComputedStyle(e).opacity));
  console.log("INFO ⋯ tiêu đề: trước", JSON.stringify(before), "| rê", hover, "| focus", focus);
  expect(Math.max(...before), "trước rê / focus ẩn").toBe(0);
  expect(hover, "rê hiện").toBe(1);
  expect(focus, "focus hiện").toBe(1);
  await ctx.close();
});

test("TC-UX16-21 sổ điểm 375: ô nhập cao ≥ 44, cuộn ngang trong bảng, không tràn trang", async ({ browser }, info) => {
  desktopOnly(info);
  const { ctx, page, ids } = await open(browser, 375, 812);
  await page.goto(`/gradebook?course=${ids["761987"]}`);
  await page.waitForTimeout(5000);
  const r = await page.evaluate(() => ({
    ov: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    inp: [...document.querySelectorAll("main tbody tr:first-child input")].map((e) => Math.round(e.getBoundingClientRect().height)),
  }));
  console.log("INFO ô nhập 375:", JSON.stringify(r));
  expect(r.ov).toBe(false);
  expect(r.inp.length).toBeGreaterThan(0);
  for (const h of r.inp) expect(h).toBeGreaterThanOrEqual(44);
  await ctx.close();
});
