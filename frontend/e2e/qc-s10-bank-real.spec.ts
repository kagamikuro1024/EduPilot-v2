import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");

// QC sprint 10, US-P9-04: chọn hàng → thanh hành động hàng loạt; Drawer sửa câu (GV, 1440).
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
test("bank bulk + drawer", async ({ browser }) => {
  test.setTimeout(120_000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.waitForTimeout(800);
  if (!(await page.locator("[data-part=topbar]").getByText("761987").count())) { await page.getByRole("button", { name: /Chọn lớp/ }).click(); await page.getByText("761987").first().click(); await page.waitForTimeout(800); }
  await page.goto("/questions"); await page.waitForTimeout(1500);
  await page.getByRole("combobox").first().waitFor().catch(() => {});
  const boxes = page.getByRole("checkbox");
  for (let i = 1; i <= 3; i++) await boxes.nth(i).check();
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${OUT}/bank-bulk-1440.png` });
  console.log("TOOLBAR", JSON.stringify(await page.getByRole("toolbar").allInnerTexts()));
  for (let i = 1; i <= 3; i++) await boxes.nth(i).uncheck();
  await page.getByRole("row").nth(1).click();
  await page.waitForTimeout(800);
  await page.screenshot({ path: `${OUT}/bank-drawer-1440.png` });
  const src = page.getByText(/Tài liệu, trang/).first();
  console.log("SRC", await src.evaluate((e) => ({ tag: e.tagName, href: (e.closest("a") as HTMLAnchorElement | null)?.href ?? null, parent: e.parentElement?.outerHTML.slice(0, 200) })));
  await ctx.close();
});
