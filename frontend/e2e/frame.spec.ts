import { mkdirSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { measureFrame, type FrameRow } from "./support/frame";
import { settleGoto } from "./support/hydrate";
import { ROLES, routesFor } from "./support/routes";
import { asDemo, type DemoRole } from "./support/session";

// Hợp đồng khung trang (DESIGN.md §23, sprint 10.7). Đo DOM qua móc `data-ep-*` trên MỌI route × vai × 1440 / 375 × (có dữ liệu, rỗng).
// Chế độ báo cáo (mặc định): in và đính kèm bảng độ lệch, KHÔNG làm hỏng test. `FRAME_STRICT=1` (CI sau lô 7): mỗi độ lệch là một lỗi.
// Route mới tự vào ma trận vì `routesFor` duyệt bảng điều hướng `nav.ts`.
const STRICT = process.env.FRAME_STRICT === "1";
const SIZES = [
  { name: "1440", w: 1440, h: 900, wide: true },
  { name: "375", w: 375, h: 812, wide: false },
] as const;

test.beforeEach(({ page }) => settleGoto(page));

for (const role of ROLES) {
  test(`frame ${role}: mọi route × 1440 / 375 × (dữ liệu, rỗng)`, async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    test.setTimeout(300_000);
    await asDemo(context, role as DemoRole);
    const rows: Array<FrameRow & { route: string; size: string; state: string }> = [];
    for (const size of SIZES) {
      await page.setViewportSize({ width: size.w, height: size.h });
      for (const route of routesFor(role)) {
        for (const state of ["data", "empty"] as const) {
          await page.goto(state === "empty" ? `${route}${route.includes("?") ? "&" : "?"}state=empty` : route);
          await page.locator("main, body").first().waitFor();
          await page.waitForLoadState("networkidle").catch(() => undefined);
          rows.push({ route, size: size.name, state, ...(await measureFrame(page, { wide: size.wide })) });
        }
      }
    }
    const bad = rows.filter((r) => r.devs.length);
    const table = rows.map((r) => `${r.devs.length ? "✗" : "✓"} ${role} ${r.route} ${r.size} ${r.state} kind=${r.kind ?? "-"} cột=${r.cols} đầu=${r.headerH ?? "-"}${r.devs.length ? " :: " + r.devs.map((d) => `${d.code} ${d.detail}`).join("; ") : ""}`).join("\n");
    mkdirSync("test-results/frame", { recursive: true }); // bảng độ lệch cho người đọc (không phụ thuộc trạng thái đạt / đỏ của test)
    writeFileSync(`test-results/frame/frame-${role}.json`, JSON.stringify(rows, null, 1));
    await info.attach(`frame-${role}.txt`, { body: table, contentType: "text/plain" });
    await info.attach(`frame-${role}.json`, { body: JSON.stringify(rows, null, 1), contentType: "application/json" });
    console.log(`frame ${role}: ${bad.length} / ${rows.length} lượt lệch`);
    if (STRICT && bad.length) throw new Error(bad.map((r) => `${r.route} ${r.size} ${r.state}: ${r.devs.map((d) => `${d.code} ${d.detail}`).join("; ")}`).join("\n"));
  });
}
