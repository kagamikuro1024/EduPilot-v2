import { test } from "@playwright/test";

test.skip(!process.env.QC_REAL, "cần stack thật ep-qc");

// QC sprint 10, US-P9-11: hộp thoại Nhập từ Forms trên stack thật (GV, 1440).
const OUT = process.env.QC_OUT ?? "/tmp/qc10-shots";
test("Nhập từ Forms: hộp thoại", async ({ browser }) => {
  test.setTimeout(120_000);
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: "vi-VN" });
  const page = await ctx.newPage();
  await page.goto("/login");
  await page.locator("input[type=email]").fill("teacher@edupilot.local");
  await page.locator("input[type=password]").fill(process.env.QC_PW ?? "");
  await page.keyboard.press("Enter");
  await page.waitForURL((u) => !u.pathname.startsWith("/login"));
  await page.waitForTimeout(1000);
  const sw = page.locator("[data-part=topbar]").getByText(/76198\d/).first();
  if (!(await page.locator("[data-part=topbar]").getByText("761987").count())) { await sw.click(); await page.getByText("761987").first().click(); await page.waitForTimeout(800); }
  await page.goto("/questions"); await page.waitForTimeout(1500); await page.screenshot({ path: `${OUT}/questions-761987-1440.png` });
  await page.goto("/gradebook"); await page.waitForTimeout(1500); await page.screenshot({ path: `${OUT}/gradebook-761987-1440.png` });
  const names = await page.getByRole("button").evaluateAll((els) => els.map((e) => (e.getAttribute("aria-label") ?? e.textContent ?? "").trim()).slice(0, 40));
  console.log("BUTTONS", JSON.stringify(names));
  await page.getByRole("button", { name: "Thêm hành động" }).first().click();
  await page.screenshot({ path: `${OUT}/forms-0-menu.png` });
  await page.getByText(/Nhập từ Forms/).first().click();
  await page.waitForTimeout(800);
  await page.screenshot({ path: `${OUT}/forms-1-dialog.png` });
  await page.locator("input[type=file]").setInputFiles("/tmp/forms-qc.csv");
  await page.waitForTimeout(2500);
  await page.screenshot({ path: `${OUT}/forms-2-uploaded.png` });
  await ctx.close();
});
