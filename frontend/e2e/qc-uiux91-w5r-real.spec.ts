import { expect, test, type Page } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc: chạy bằng `QC_REAL=1 playwright test -c playwright.qc-real.config.ts`");
const PW = process.env.QC_PW ?? "Edupilot#Seed-2026";

async function login(page: Page, email: string) {
  await page.goto("/login");
  await page.locator("input[type=email]").fill(email);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.locator("[data-part=topbar]").waitFor();
}

const tipOf = (page: Page, el: ReturnType<Page["locator"]>) =>
  el.evaluate((e) => {
    const id = e.getAttribute("aria-describedby");
    const t = id ? document.getElementById(id) : null;
    return { id, text: t?.textContent?.trim() ?? null, role: t?.getAttribute("role") ?? null, opacity: t ? getComputedStyle(t).opacity : null, title: e.getAttribute("title") };
  });

test("TC-UX01-4 #3 nút «Mở lớp»: tooltip hiện khi focus / rê", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "admin@edupilot.local");
  await page.goto("/admin/courses");
  await page.getByRole("button", { name: "Mở lớp" }).first().click();
  const submit = page.locator("form button[type=submit]").filter({ hasText: "Mở lớp" }).first();
  await submit.waitFor();
  for (const [l, v] of [["Học phần", "QC101"], ["Mã lớp", "QC999001"], ["Tên lớp", "Lớp thử QC tooltip"], ["Học kỳ", "2026-2027-HK1"]]) await page.getByLabel(l, { exact: true }).fill(v);
  await expect(submit).toBeEnabled();
  const before = await tipOf(page, submit);
  console.log("INFO UX01-4 trước focus", JSON.stringify(before), "disabled:", await submit.isDisabled(), "aria-disabled:", await submit.getAttribute("aria-disabled"));
  await submit.focus();
  await page.waitForTimeout(400);
  const f = await tipOf(page, submit);
  console.log("INFO UX01-4 focus", JSON.stringify(f));
  await page.locator("body").click({ position: { x: 5, y: 400 } });
  await page.mouse.move(0, 0);
  await page.waitForTimeout(300);
  await submit.hover({ force: true });
  await page.waitForTimeout(400);
  const h = await tipOf(page, submit);
  console.log("INFO UX01-4 hover", JSON.stringify(h), "từ:", (h.text ?? "").split(/\s+/).length);
  expect(f.text, "UX01-4: có mô tả").toBe("Người được gán nhận thông báo kèm mã tham gia.");
  expect(f.role).toBe("tooltip");
  expect(Number(f.opacity), "UX01-4: tooltip hiện khi focus").toBeGreaterThan(0.9);
  expect(Number(h.opacity), "UX01-4: tooltip hiện khi rê").toBeGreaterThan(0.9);
  await ctx.close();
});

test("TC-UX01-11 #34 Switch «Cần giảng viên duyệt»: tooltip thật, hiện khi focus bàn phím", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "teacher@edupilot.local");
  await page.goto("/class/settings");
  const sw = page.getByRole("switch", { name: /Cần giảng viên duyệt/ });
  await sw.waitFor();
  const before = await tipOf(page, sw);
  await page.keyboard.press("Tab"); // bắt đầu từ thân trang
  await sw.focus();
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Tab");
  await page.waitForTimeout(400);
  const f = await tipOf(page, sw);
  console.log("INFO UX01-11 trước", JSON.stringify(before), "| focus", JSON.stringify(f), "| focused:", await sw.evaluate((e) => e === document.activeElement));
  await page.locator("body").click({ position: { x: 5, y: 400 } });
  await page.mouse.move(0, 0);
  await page.waitForTimeout(300);
  await sw.hover();
  await page.waitForTimeout(400);
  const h = await tipOf(page, sw);
  console.log("INFO UX01-11 hover", JSON.stringify(h));
  expect(f.text).toBe("Sinh viên vào bằng mã sẽ chờ bạn duyệt.");
  expect(f.role).toBe("tooltip");
  expect(f.title, "UX01-11: không còn title native").toBeNull();
  expect(Number(f.opacity), "UX01-11: hiện khi focus bàn phím").toBeGreaterThan(0.9);
  expect(Number(h.opacity), "UX01-11: hiện khi rê").toBeGreaterThan(0.9);
  await ctx.close();
});

test("TC-UX06-7 lớp ARCHIVED không được nhớ / không là lớp mặc định", async ({ browser }, info) => {
  test.skip(info.project.name !== "desktop");
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  await login(page, "teacher@edupilot.local");
  const courses = await page.evaluate(async () => {
    const a = await (await fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })).json();
    const m = await (await fetch("/api/v1/me/courses", { headers: { authorization: `Bearer ${a.access_token}` } })).json();
    return m.items.map((i: { course: { id: string; class_code: string; status: string } }) => [i.course.id, i.course.class_code, i.course.status]);
  });
  console.log("INFO UX06-7 lớp của teacher:", JSON.stringify(courses));
  const arch = courses.find((c: string[]) => c[2] === "ARCHIVED");
  expect(arch, "cần lớp ARCHIVED của teacher (QC196964)").toBeTruthy();
  const label = () => page.locator("[data-part=topbar]").getByRole("button", { name: /Chọn lớp/ }).first().getAttribute("aria-label");
  await page.evaluate((id) => localStorage.setItem("ep:ui:course", id), arch[0]);
  await page.goto("/");
  await page.waitForTimeout(3500);
  const stored = await page.evaluate(() => localStorage.getItem("ep:ui:course"));
  const lb = await label();
  console.log("INFO UX06-7 sau nạp: ep:ui:course =", stored, "| nhãn:", lb);
  expect(stored, "UX06-7: lớp lưu trữ bị xoá khỏi ep:ui:course").not.toBe(arch[0]);
  expect(lb ?? "").not.toContain(arch[1]);
  // chọn tay lớp lưu trữ từ bộ chọn (nếu có): tải lại không giữ
  await page.locator("[data-part=topbar]").getByRole("button", { name: /Chọn lớp/ }).first().click();
  const opt = page.getByRole("option", { name: new RegExp(arch[1]) }).or(page.getByRole("menuitemradio", { name: new RegExp(arch[1]) })).or(page.getByRole("menuitem", { name: new RegExp(arch[1]) }));
  const n = await opt.count();
  console.log("INFO UX06-7 bộ chọn có mục lớp lưu trữ:", n);
  if (n) {
    await opt.first().click();
    await page.waitForTimeout(800);
    console.log("INFO UX06-7 sau chọn tay: nhãn", await label(), "| stored", await page.evaluate(() => localStorage.getItem("ep:ui:course")));
    await page.reload();
    await page.waitForTimeout(3000);
    const lb2 = await label();
    console.log("INFO UX06-7 sau tải lại: nhãn", lb2, "| stored", await page.evaluate(() => localStorage.getItem("ep:ui:course")));
    expect(lb2 ?? "").not.toContain(arch[1]);
  }
  await ctx.close();
});
