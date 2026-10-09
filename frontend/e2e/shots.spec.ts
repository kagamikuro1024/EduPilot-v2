import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
import { test } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { asDemo, type DemoRole } from "./support/session";

// Chụp ảnh bằng chứng của sprint 5.5 (chỉ chạy khi đặt SHOTS; ghi vào thư mục handoff). Không phải ảnh mốc — ảnh mốc là visual.spec.ts.
//   SHOTS=ui03 → docs/sprints/5.5/handoff/ui-03/<vai>-<bề rộng>.png (khung `/` của bốn vai ở 1440 / 1024 / 375)
test.beforeEach(({ page }) => settleGoto(page));
test.skip(!process.env.SHOTS, "đặt SHOTS=ui03 để chụp");

const OUT = resolve(__dirname, "../../docs/sprints/5.5/handoff");
const ROLES: DemoRole[] = ["student", "ta", "teacher", "admin"];

if (process.env.SHOTS === "ui03") {
  for (const role of ROLES) {
    test(`ui-03 khung / ${role}`, async ({ page, context }, info) => {
      test.skip(info.project.name !== "desktop", "chụp một lần");
      const dir = resolve(OUT, "ui-03");
      mkdirSync(dir, { recursive: true });
      await asDemo(context, role);
      for (const w of [1440, 1024, 375]) {
        await page.setViewportSize({ width: w, height: 900 });
        await page.goto("/");
        await page.locator("main").first().waitFor();
        await page.evaluate(() => document.fonts.ready);
        await page.waitForTimeout(300);
        await page.screenshot({ path: resolve(dir, `${role}-${w}.png`) });
      }
    });
  }
}
