import { mkdirSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { measureFrame, type FrameRow } from "./support/frame";
import { settleGoto } from "./support/hydrate";
import { ROLES, routesFor } from "./support/routes";
import { BASE_URL } from "./support/env";
import { asDemo, type DemoRole } from "./support/session";

// Hợp đồng khung trang (DESIGN.md §23, sprint 10.7). Đo DOM qua móc `data-ep-*` trên MỌI route × vai × 1440 / 375 × (có dữ liệu, rỗng).
// Mặc định (sau lô 7, nợ khung rỗng): mỗi độ lệch là một lỗi. `FRAME_REPORT=1`: chỉ in và đính kèm bảng độ lệch, KHÔNG làm hỏng test (dùng khi đo).
// Route mới tự vào ma trận vì `routesFor` duyệt bảng điều hướng `nav.ts`.
const STRICT = process.env.FRAME_REPORT !== "1";
const SIZES = [
  { name: "1440", w: 1440, h: 900, wide: true },
  { name: "375", w: 375, h: 812, wide: false },
] as const;

// Không có trạng thái đang tải / lỗi của vùng để đo ở bản dựng mô phỏng: `/join` chỉ có biểu mẫu; `/chat` là hội thoại mô phỏng (DemoChat) không nghe ?state=; `/settings/integrations` của Admin chưa chọn lớp nên chưa tải gì. Các yêu cầu /api/v1 bị chặn ở đó là của khung ứng dụng. Nợ sprint 11 (nối API thật): đo lại hai route sau.
const FORM_ONLY = /^\/(join|chat|settings\/integrations)(\/|$)/;

test.beforeEach(({ page }) => settleGoto(page));

for (const role of ROLES) {
  test(`frame ${role}: mọi route × 1440 / 375 × (dữ liệu, rỗng, đang tải, lỗi)`, async ({ page, context }, info) => {
    test.skip(info.project.name !== "desktop", "đo một lần");
    test.setTimeout(300_000);
    await asDemo(context, role as DemoRole);
    const rows: Array<FrameRow & { route: string; size: string; state: string }> = [];
    for (const size of SIZES) {
      await page.setViewportSize({ width: size.w, height: size.h });
      for (const route of routesFor(role)) {
        for (const state of ["data", "empty", "loading", "error"] as const) {
          // đang tải / lỗi vào bằng CẢ HAI đường: ?state= (màn mô phỏng) và chặn /api/v1 (màn nối API), như QC (TL-1 §12.2)
          const calls = new Set<string>(); // số yêu cầu /api/v1 KHÁC NHAU bị chặn (TL 14.7)
          if (state === "loading" || state === "error") {
            await page.route(/\/api\/v1\/(?!auth\/|me(\?|$)|me\/courses)/, async (r) => {
              calls.add(`${r.request().method()} ${new URL(r.request().url()).pathname}`);
              if (state === "error") return r.fulfill({ status: 500, contentType: "application/json", headers: { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" }, body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t" }) });
              await new Promise((x) => setTimeout(x, 5000));
              return r.abort().catch(() => undefined);
            });
          }
          await page.goto(state === "data" ? route : `${route}${route.includes("?") ? "&" : "?"}state=${state}`);
          await page.locator("main, body").first().waitFor();
          if (state === "loading" || state === "error") await page.waitForTimeout(800);
          else await page.waitForLoadState("networkidle").catch(() => undefined);
          rows.push({ route, size: size.name, state, ...(await measureFrame(page, { wide: size.wide, state, calls: FORM_ONLY.test(route) ? 0 : calls.size })) });
          await page.unrouteAll({ behavior: "ignoreErrors" });
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
