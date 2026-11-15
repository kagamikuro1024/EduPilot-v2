import { test } from "@playwright/test";
import { asDemo, type DemoRole } from "./support/session";
import { FROZEN, LONG_EXP } from "./support/screens";
import { mockStaffApi } from "./support/staffMock";
import { settleGoto } from "./support/hydrate";

// TẠM (không commit): ảnh trước / sau cho PM. SHOTS_DIR, SHOTS="tên|đường dẫn|vai;…"
const DIR = process.env.SHOTS_DIR ?? "/tmp/shots/x";
const LIST = (process.env.SHOTS ?? "").split(";").filter(Boolean).map((x) => x.split("|"));
test.beforeEach(({ page }) => settleGoto(page));
for (const [name, path, role] of LIST) for (const w of [1440, 375]) {
  test(`shot ${name} ${w}`, async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop");
    await page.setViewportSize({ width: w, height: w === 375 ? 812 : 900 });
    await page.clock.setFixedTime(FROZEN);
    await asDemo(context, role as DemoRole, { expIn: LONG_EXP });
    await mockStaffApi(page).catch(() => undefined);
    await page.goto(path);
    await page.locator("main").waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.waitForTimeout(700);
    await page.screenshot({ path: `${DIR}/${name}-${w}.png` });
  });
}
