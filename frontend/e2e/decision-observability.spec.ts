import { expect, test, type Page } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { err, getLog, reset, script } from "./support/llm-fixtures";
import { asJwt } from "./support/session";

// US-DM-07: bảng «Quyết định nhanh» ở /observability (ADMIN) — máy chủ giả theo hợp đồng thật.
const OBS = "/api/v1/admin/observability/decision";
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});

const row = (gate: string, over: Record<string, unknown> = {}) => ({
  gate, state: "off", courses_enabled: 0, courses_total: 12, calls: 0, fast_pct: 0, fallback_pct: 0, fail_closed_pct: 0, p50_ms: 0, p95_ms: 0, error_pct: 0, cost_vnd: "0", ...over,
});
const body = (over: Record<string, unknown>[] = []) => ({
  gates: [
    row("CHAT_ROUTING", { state: "on", courses_enabled: 3, calls: 200, fast_pct: 80, fallback_pct: 19.5, fail_closed_pct: 0.5, p50_ms: 426, p95_ms: 682, error_pct: 2, cost_vnd: "1234" }),
    row("INJECTION_FLAG"),
    row("THREADS_PRECHECK", { state: "on", courses_enabled: 1, calls: 40, fast_pct: 60, fallback_pct: 30, fail_closed_pct: 10, p50_ms: 500, p95_ms: 900, error_pct: 0, cost_vnd: "80" }),
    ...over,
  ],
});

async function openObs(page: Page, scripts: Record<string, unknown[]>, role: "ADMIN" | "TEACHER" = "ADMIN") {
  await reset(page);
  await script(page, scripts as never);
  await asJwt(page, role, { email: "admin@ptit.edu.vn" });
  await page.goto("/observability");
}

test("bảng theo cổng: tên đúng bảng, cổng tắt hiện «—», số định dạng, một chỗ cho mỗi số", async ({ page }) => {
  await openObs(page, { [`GET ${OBS}`]: [{ body: body() }] });
  const sec = page.locator("[data-part=decision-observability]");
  await expect(sec.getByRole("heading", { name: "Quyết định nhanh" })).toBeVisible();
  const t = sec.getByRole("table", { name: "Quyết định nhanh theo cổng" });
  await expect(t.locator("tbody tr")).toHaveCount(3);
  const r = (n: number) => t.locator("tbody tr").nth(n);
  await expect(r(0)).toContainText("Định tuyến câu hỏi chat");
  await expect(r(0)).toContainText("Bật ở 3/12 lớp");
  await expect(r(0)).toContainText("200");
  await expect(r(0)).toContainText("80%");
  await expect(r(0)).toContainText("426 ms");
  await expect(r(0)).toContainText("682 ms");
  await expect(r(0)).toContainText("1.234 đ");
  await expect(r(1)).toContainText("Cờ chèn lệnh khi chấm");
  await expect(r(1)).toContainText("Tắt");
  expect((await r(1).locator("td").allTextContents()).filter((x) => x.trim() === "—")).toHaveLength(8);
  await expect(r(2)).toContainText("10%");
  // chỉ số decision không lặp ở dải trạng thái đầu trang (AC2)
  await expect(page.locator("[data-part=obs-today]")).not.toContainText(/Quyết định|đường nhanh/i);
});

test("chọn cửa sổ 24 giờ / 7 ngày gọi lại với window; chưa có lời gọi → một dòng", async ({ page }) => {
  await openObs(page, { [`GET ${OBS}`]: [{ body: body() }, { body: { gates: [row("CHAT_ROUTING"), row("INJECTION_FLAG"), row("THREADS_PRECHECK")] } }] });
  const sec = page.locator("[data-part=decision-observability]");
  await expect(sec.getByRole("table")).toBeVisible();
  expect((await getLog(page, OBS))[0].search).toBe("?window=24h");
  await sec.getByRole("radio", { name: "7 ngày" }).click();
  await expect(sec.getByText("Chưa có lời gọi nào")).toBeVisible();
  expect((await getLog(page, OBS))[1].search).toBe("?window=7d");
});

test("lỗi API → Thử lại (không spinner riêng)", async ({ page }) => {
  await openObs(page, { [`GET ${OBS}`]: [err(503, "SERVICE_UNAVAILABLE")] });
  const sec = page.locator("[data-part=decision-observability]");
  await expect(sec.getByRole("button", { name: "Thử lại" })).toBeVisible({ timeout: 20_000 });
});

test("TEACHER không thấy bảng và không gọi API decision (AC6)", async ({ page }) => {
  await openObs(page, { [`GET ${OBS}`]: [{ body: body() }] }, "TEACHER");
  await expect(page.getByRole("heading", { name: "Quan sát AI" })).toBeVisible();
  await expect(page.locator("[data-part=decision-observability]")).toHaveCount(0);
  expect((await getLog(page, OBS)).length).toBe(0);
});
