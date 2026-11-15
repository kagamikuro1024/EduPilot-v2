import { expect, test } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { asDemo } from "./support/session";

// DEF-R1.5-02 (TC-00-36, AC10): mã lớp dài không đẩy nút hồ sơ ra ngoài khung ở 375 / 390 px.
test.beforeEach(({ page }) => settleGoto(page));
test.beforeEach(({}, info) => test.skip(info.project.name !== "desktop", "tự đặt bề rộng"));

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const LONG = "QC-AS3r108-ABCDEF"; // 17 ký tự, trong giới hạn 3–20 của class_code

for (const width of [375, 390]) {
  test(`topbar ${width}: TA ở lớp có mã dài không cuộn ngang, nút hồ sơ trong khung`, async ({ page, context }) => {
    await asDemo(context, "ta");
    await page.route("**/api/v1/me/courses**", (r) =>
      r.fulfill({
        status: 200,
        contentType: "application/json",
        headers: cors,
        body: JSON.stringify({
          items: [{ course: { id: "00000000-0000-7000-8000-00000000c001", class_code: LONG, subject_code: "INT1006", name: "QC điểm danh " + LONG, semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TA", enrollment_status: "ACTIVE" }],
          next_cursor: null,
        }),
      }),
    );
    await page.setViewportSize({ width, height: 812 });
    await page.goto("/threads");
    await expect(page.locator("[data-part=topbar]")).toBeVisible();
    await expect(page.getByRole("button", { name: /Tài khoản/ })).toBeVisible();
    const over = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, iw: window.innerWidth, right: document.querySelector("[data-part=topbar] [aria-label^='Tài khoản']")?.getBoundingClientRect().right ?? 0 }));
    expect(over.sw).toBeLessThanOrEqual(over.iw);
    expect(over.right).toBeLessThanOrEqual(over.iw);
  });
}
