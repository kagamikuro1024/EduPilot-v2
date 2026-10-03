import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { base, reset, script } from "./support/llm-fixtures";
import { asDemo, asJwt, type DemoRole } from "./support/session";

// US-PU-05 AC1: 7 route đại diện × 2 bề rộng = 14 ảnh mốc (SRS 8.3). Đồng hồ đóng băng 29/10/2026 09:20 giờ VN, animation tắt,
// font đã nạp. Cập nhật ảnh mốc CHỈ bằng `--update-snapshots` kèm giải thích trong handoff.
// Ảnh mốc sinh trên macOS arm64; nếu CI Linux lệch do khử răng cưa, sinh lại trong CI và giải thích (xem handoff).
test.describe.configure({ retries: 0 });
test.use({ reducedMotion: "reduce", colorScheme: "light" });
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

const FROZEN = new Date("2026-10-29T09:20:00+07:00");
const WIDTHS = [
  { w: 1440, h: 900 },
  { w: 390, h: 844 },
] as const;
const ROUTES: Array<{ name: string; path: string; role?: DemoRole; admin?: boolean }> = [
  { name: "home", path: "/", role: "student" },
  { name: "chat", path: "/chat", role: "student" },
  { name: "threads", path: "/threads", role: "student" },
  { name: "inbox", path: "/inbox", role: "teacher" },
  { name: "gradebook", path: "/gradebook", role: "teacher" },
  { name: "settings-llm", path: "/settings/llm", role: "admin", admin: true }, // Admin bằng token dev (dán ở cổng); màn THẬT với dữ liệu giả cố định
  { name: "dev-ui", path: "/dev/ui" },
];

async function settle(page: Page) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForLoadState("networkidle");
  await page.waitForTimeout(300);
}

for (const r of ROUTES) {
  for (const { w, h } of WIDTHS) {
    test(`${r.name} @${w}`, async ({ page, context }) => {
      if (r.admin) await acquireFakeApi(); // màn thật dùng máy chủ giả dùng chung
      try {
        await shoot(page, context, r, w, h);
      } finally {
        if (r.admin) releaseFakeApi();
      }
    });
    async function shoot(page: Page, context: BrowserContext, r: (typeof ROUTES)[number], w: number, h: number) {
      await page.setViewportSize({ width: w, height: h });
      await page.clock.setFixedTime(FROZEN);
      if (r.admin) {
        await reset(page);
        await script(page, base()); // dữ liệu cố định theo hợp đồng thật (support/llm-fixtures.ts)
      }
      if (r.admin) await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn", fullName: "Đỗ Hoàng Nam" }); // phiên thật giả lập (refresh → JWT ADMIN)
      else if (r.role) await asDemo(context, r.role);
      await page.goto(r.path);
      await page.locator("main").first().waitFor();
      await settle(page);
      await expect(page).toHaveScreenshot(`${r.name}-${w}.png`, { maxDiffPixelRatio: 0.005, threshold: 0.2, animations: "disabled", caret: "hide" });
    }
  }
}
