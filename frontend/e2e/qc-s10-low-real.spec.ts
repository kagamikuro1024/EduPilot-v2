import { writeFileSync } from "node:fs";
import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");

// QC sprint 10: kiểm lại L11 (menu Tạo câu hỏi), L15 (lớp chưa có câu), L16 (nhãn nguồn trong Drawer), L12 (hộp thoại Forms).
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
const PW = process.env.QC_PW ?? "";
async function login(browser: import("@playwright/test").Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill(`${user}@edupilot.local`);
  await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.waitForTimeout(800);
  return { ctx, page };
}
async function pickClass(page: import("@playwright/test").Page, code: string) {
  if (!(await page.locator("[data-part=topbar]").getByText(code).count())) { await page.getByRole("button", { name: /Chọn lớp/ }).click(); await page.getByText(code).first().click(); await page.waitForTimeout(800); }
}
test("GV: menu, Drawer, Forms", async ({ browser }) => {
  test.setTimeout(120_000);
  const out: string[] = [];
  const { ctx, page } = await login(browser, "teacher");
  await pickClass(page, "761987");
  await page.goto("/questions"); await page.waitForTimeout(1500);
  await page.getByRole("button", { name: "Tạo câu hỏi" }).first().click(); await page.waitForTimeout(400);
  out.push("menu: " + (await page.getByRole("menuitem").allInnerTexts()).join(" | "));
  await page.screenshot({ path: `${OUT}/low-menu-1440.png` });
  const soan = page.getByRole("menuitem", { name: /Soạn tay/ });
  if (await soan.count()) { await soan.first().click(); await page.waitForTimeout(400); out.push("sau Soạn tay: " + (await page.getByRole("menuitem").allInnerTexts()).join(" | ")); await page.screenshot({ path: `${OUT}/low-menu-soan-tay-1440.png` }); }
  await page.keyboard.press("Escape");
  await page.getByRole("row", { name: /DNS dùng để làm gì/ }).first().click(); await page.waitForTimeout(800);
  const src = page.getByRole("link", { name: /trang \d/ }).first();
  out.push("nguồn: " + (await src.count() ? await src.innerText() : "không có liên kết"));
  await page.screenshot({ path: `${OUT}/low-drawer-1440.png` });
  writeFileSync(`${OUT}/notes-low-gv.txt`, out.join("\n"));
  await ctx.close();
});
test("SV lớp chưa có câu", async ({ browser }) => {
  const { ctx, page } = await login(browser, "sv.gioi");
  await pickClass(page, "761988");
  await page.goto("/practice"); await page.waitForTimeout(1500);
  writeFileSync(`${OUT}/notes-low-empty.txt`, JSON.stringify(await page.locator("main").innerText()));
  await page.screenshot({ path: `${OUT}/low-practice-empty-1440.png` });
  await ctx.close();
});
