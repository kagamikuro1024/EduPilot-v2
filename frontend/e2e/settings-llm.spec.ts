import AxeBuilder from "@axe-core/playwright";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { BASE_URL } from "./support/env";
import { expect, test, type Page } from "@playwright/test";
import { loadAudit, runAudit } from "./support/audit";
import { CHAIN3, IDS, base, budget, err, getLog, jwt, openLlm, provider, providers, reset, script, usage } from "./support/llm-fixtures";
import { asDemo, type DemoRole } from "./support/session";

// US-P1-05: màn /settings/llm THẬT, thử với máy chủ giả theo hợp đồng thật (e2e/support/llm-fixtures.ts). Ca @real (gateway thật) không chạy ở CI.
const P = "/api/v1/admin/llm";
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});
const putRoutes = (page: Page) => getLog(page, `${P}/routes`).then((l) => l.filter((x) => x.method === "PUT"));

test("sections: 4 phần đúng thứ tự, không lồng, khoảng cách đều, 0 nút primary lúc đầu", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN");
  await expect(page.getByRole("heading", { name: "Kết nối nhà cung cấp", level: 2 })).toBeVisible();
  const heads = await page.locator("[data-part=settings-section] h2:visible").allTextContents();
  expect(heads).toEqual(["Kết nối nhà cung cấp", "Mô hình theo tác vụ", "Chuỗi dự phòng", "Mô hình tìm kiếm tài liệu"]);
  expect(await page.locator("[data-part=settings-section] [data-part=settings-section]").count()).toBe(0);
  const gaps = await page.locator("[data-part=settings-section]").evaluateAll((els) => els.slice(1).map((e, i) => Math.round(e.getBoundingClientRect().top - els[i].getBoundingClientRect().bottom)));
  expect(Math.max(...gaps) - Math.min(...gaps)).toBeLessThanOrEqual(1);
  expect(await page.locator('main [data-variant="primary"]:visible').count()).toBe(0);
  await expect(page.getByRole("heading", { name: "Mức dùng và ngân sách" })).toBeVisible();
});

test("provider row: Test kết nối ở mỗi hàng, lỗi ngay dưới hàng, thêm nhà bấm đúp Lưu chỉ 1 POST", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", {
    [`POST ${P}/providers/${IDS.gemini}/test`]: [{ body: { ok: false, error_kind: "AUTH", message: "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.", latency_ms: 90 } }],
    [`POST ${P}/providers/${IDS.openai}/test`]: [{ body: { ok: true, latency_ms: 420 } }],
    [`POST ${P}/providers`]: [{ status: 201, body: provider({ id: "00000000-0000-7000-8000-000000000099", name: "Máy chủ trường", type: "fake" }), delay: 400 }],
  });
  const rows = page.locator("[data-part=provider-row]");
  await expect(rows).toHaveCount(2);
  await expect(page.getByRole("button", { name: "Test kết nối" })).toHaveCount(2);
  expect(await page.locator("main").getByRole("button", { name: "Test kết nối" }).evaluateAll((bs) => bs.every((b) => b.closest("[data-part=provider-row]") && b.getAttribute("data-variant") === "secondary"))).toBe(true);
  await expect(rows.nth(0).locator("[data-part=provider-status]")).toContainText("Đã kết nối · kiểm tra lúc");
  await expect(rows.nth(1).locator("[data-part=provider-status]")).toContainText("Lỗi xác thực");
  await rows.nth(1).getByRole("button", { name: "Test kết nối" }).click();
  await expect(rows.nth(1).getByText("Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.")).toBeVisible();
  await rows.nth(0).getByRole("button", { name: "Test kết nối" }).click();
  await expect(rows.nth(0).getByText("Kết nối tốt · 420 ms")).toBeVisible();
  await expect(page.locator('[role="status"]:has-text("Thành công")')).toHaveCount(0);

  await page.getByRole("button", { name: "Thêm nhà cung cấp" }).click();
  const form = page.locator("[data-part=provider-form]");
  await form.getByLabel("Loại nhà cung cấp").selectOption("fake");
  await form.getByLabel(/Tên hiển thị/).fill("Máy chủ trường");
  const save = form.getByRole("button", { name: "Lưu", exact: true });
  await save.dblclick();
  await expect.poll(async () => (await getLog(page, `${P}/providers`)).filter((l) => l.method === "POST").length).toBe(1);
  await page.waitForTimeout(700);
  expect((await getLog(page, `${P}/providers`)).filter((l) => l.method === "POST")).toHaveLength(1);
});

test("key write-only: khoá không bao giờ vào DOM / storage / console; ô khoá trống sau lưu", async ({ page, context }) => {
  const CANARY = "sk-CANARY-abcd1234wxyz";
  const logs: string[] = [];
  page.on("console", (m) => logs.push(m.text()));
  await openLlm(page, context, "ADMIN", { [`POST ${P}/providers`]: [{ status: 201, body: provider({ id: "00000000-0000-7000-8000-000000000098", name: "Mới" }) }] });
  await expect(page.locator("[data-part=provider-row]")).toHaveCount(2);
  await page.getByRole("button", { name: "Thêm nhà cung cấp" }).click();
  const form = page.locator("[data-part=provider-form]");
  const key = form.getByLabel(/Khoá API/);
  await expect(key).toHaveAttribute("type", "password");
  await expect(key).toHaveAttribute("autocomplete", "new-password");
  await expect(key).toHaveAttribute("spellcheck", "false");
  await expect(key).toHaveValue("");
  await form.getByLabel(/Tên hiển thị/).fill("Mới");
  await key.fill(CANARY);
  await form.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(form).toHaveCount(0);
  const body = (await getLog(page, `${P}/providers`)).find((l) => l.method === "POST")!.body;
  expect(JSON.parse(body).api_key).toBe(CANARY);
  const dump = await page.evaluate(() => document.body.innerHTML + JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage }) + location.href + document.cookie);
  expect(dump).not.toContain(CANARY);
  expect(dump).not.toContain("wxyz");
  expect(logs.join("\n")).not.toContain("wxyz");
  for (const v of await page.locator("input[type=password]").evaluateAll((es) => es.map((e) => (e as HTMLInputElement).value))) expect(v).toBe("");
  // Đổi khoá: ô trống khi mở
  await page.locator("[data-part=provider-row]").first().getByRole("button", { name: /Thêm hành động/ }).click();
  await page.getByRole("menuitem", { name: "Đổi khoá" }).click();
  await expect(page.locator("[data-part=provider-form]").getByLabel(/Khoá API/)).toHaveValue("");
  await expect(page.getByText("Để trống để giữ khoá hiện tại")).toBeVisible();
});

test("wrong key: 422 PROVIDER_AUTH_FAILED ⇒ lỗi dưới ô khoá, không thêm hàng, giữ chữ đã gõ", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", {
    [`POST ${P}/providers`]: [{ status: 422, body: { code: "VALIDATION_FAILED", message: "m", trace_id: "c".repeat(32), details: [{ field: "api_key", code: "PROVIDER_AUTH_FAILED", message: "Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá." }] } }],
  });
  await expect(page.locator("[data-part=provider-row]")).toHaveCount(2);
  await page.getByRole("button", { name: "Thêm nhà cung cấp" }).click();
  const form = page.locator("[data-part=provider-form]");
  await form.getByLabel(/Tên hiển thị/).fill("Nhà thử");
  await form.getByLabel(/Khoá API/).fill("bad-key");
  await form.getByRole("button", { name: "Lưu", exact: true }).click();
  const key = form.getByLabel(/Khoá API/);
  await expect(key).toHaveAttribute("aria-invalid", "true");
  await expect(form.getByText("Khoá API không được nhà cung cấp chấp nhận. Kiểm tra lại khoá.")).toBeVisible();
  await expect(form.getByText("Chưa lưu gì.")).toBeVisible();
  await expect(form.getByLabel(/Tên hiển thị/)).toHaveValue("Nhà thử");
  await expect(page.locator("[data-part=provider-row]")).toHaveCount(2);
});

test("routing: đổi mô hình lạc quan ≤ 100 ms, Hoàn tác ghi ngay, Lên bằng phím Enter", async ({ page, context }) => {
  const saved = { task: "CHAT", lane: "INTERACTIVE", chain: [CHAIN3[1], CHAIN3[0], CHAIN3[2]], params: { temperature: 0.2 }, version: 4, reindex_required: false };
  await openLlm(page, context, "ADMIN", { [`PUT ${P}/routes`]: [{ body: saved, delay: 500 }, { body: { ...saved, chain: CHAIN3, version: 5 } }, { body: { ...saved, version: 6 } }] });
  const row = page.locator('[data-part=route-row][data-task=CHAT]');
  const select = row.getByLabel("Mô hình cho Trả lời chat riêng");
  await expect(select).toHaveValue(IDS.m1);
  const t0 = Date.now();
  await select.selectOption(IDS.m2);
  await expect(select).toHaveValue(IDS.m2, { timeout: 100 });
  expect(Date.now() - t0).toBeLessThan(1500);
  await expect(row.locator("[data-part=undo-line], [role=status]").filter({ hasText: "Đã chuyển Trả lời chat riêng sang gpt-4o" })).toBeVisible();
  await expect.poll(async () => (await putRoutes(page)).length).toBe(1);
  const b1 = JSON.parse((await putRoutes(page))[0].body);
  expect(b1).toMatchObject({ task: "CHAT", chain: [IDS.m2, IDS.m1, IDS.m3], version: 3 });
  await page.waitForTimeout(600);
  await row.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(select).toHaveValue(IDS.m1);
  await expect.poll(async () => (await putRoutes(page)).length).toBe(2);
  expect(JSON.parse((await putRoutes(page))[1].body)).toMatchObject({ task: "CHAT", chain: [IDS.m1, IDS.m2, IDS.m3] });

  // thứ tự dự phòng bằng nút, chỉ bàn phím
  const fb = page.locator('[data-part=fallback-row][data-task=CHAT]');
  const down = fb.getByRole("button", { name: /Đưa gpt-4o xuống sau/ });
  await down.focus();
  await page.keyboard.press("Enter");
  await expect.poll(async () => (await putRoutes(page)).length).toBe(3);
  expect(JSON.parse((await putRoutes(page))[2].body).chain).toEqual([IDS.m1, IDS.m3, IDS.m2]);
  await expect(fb.locator("text=Chưa có dự phòng")).toHaveCount(0);
  // tác vụ chỉ có 1 mô hình ⇒ cảnh báo
  await expect(page.locator('[data-part=fallback-row][data-task=CLASSIFY]').getByText("Chưa có dự phòng — nếu nhà cung cấp lỗi, người dùng sẽ thấy câu trả lời rút gọn.")).toBeVisible();
  // nhãn làn
  await expect(row).toContainText("Trả lời ngay");
  await expect(page.locator('[data-part=route-row][data-task=GRADING]')).toContainText("Chạy nền");
});

test("embedding: chỉ mô hình 1536 chiều, đổi cần xác nhận, Để sau không gửi, xác nhận gửi 1 PUT, 422 báo đúng câu", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", {
    [`PUT ${P}/routes`]: [
      { ...err(422, "MODEL_DIMS_MISMATCH") },
      { body: { task: "EMBEDDING", lane: "BATCH", chain: [{ model_id: IDS.e2, provider_id: IDS.gemini, provider_name: "Gemini", model: "gemini-embedding" }], params: {}, version: 3, reindex_required: true } },
    ],
  });
  const sel = page.getByLabel("Đổi mô hình tìm kiếm tài liệu");
  await expect(sel).toBeVisible();
  await expect(page.getByText("1536 chiều").first()).toBeVisible();
  expect(await sel.locator("option").evaluateAll((os) => os.map((o) => o.getAttribute("data-dims")))).toEqual(["1536", "1536"]);
  await sel.selectOption(IDS.e2);
  const dlg = page.getByRole("dialog");
  await expect(dlg).toContainText("lập chỉ mục lại");
  await dlg.getByRole("button", { name: "Để sau" }).click();
  expect(await putRoutes(page)).toHaveLength(0);
  await sel.selectOption(IDS.e2);
  await page.getByRole("dialog").getByRole("button", { name: "Đổi mô hình" }).click();
  await expect(page.getByRole("dialog")).toContainText("Mô hình này không dùng được cho tìm kiếm tài liệu");
  await page.getByRole("dialog").getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(await putRoutes(page)).toHaveLength(2);
  expect(JSON.parse((await putRoutes(page))[1].body)).toMatchObject({ task: "EMBEDDING", chain: [IDS.e2], version: 2 });
  await expect(page.getByText("Cần lập chỉ mục lại", { exact: true })).toBeVisible();
});

test("advanced: ẩn lúc đầu, mở có nhãn đơn vị, ngoài khoảng ⇒ lỗi và không PUT, giữ giá trị khi đóng / mở", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN");
  await expect(page.getByLabel("Nhiệt độ")).toHaveCount(0);
  await expect(page.locator("[data-part=provider-row]").first()).not.toContainText("https://");
  const row = page.locator("[data-part=route-row][data-task=CHAT]");
  const toggle = row.getByRole("button", { name: "Cài đặt nâng cao" });
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  const temp = row.getByLabel("Nhiệt độ");
  await expect(temp).toBeVisible();
  await expect(row.getByText(/Đơn vị: giây, 1–300/)).toBeVisible();
  await temp.fill("3");
  await row.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(temp).toHaveAttribute("aria-invalid", "true");
  expect(await putRoutes(page)).toHaveLength(0);
  await toggle.click();
  await toggle.click();
  await expect(row.getByLabel("Nhiệt độ")).toHaveValue("3");
});

test("usage|budget: 7 cột, số căn phải tabular-nums, cảnh báo 80 %, không vòng tiến độ, rỗng có câu", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", { [`GET ${P}/budget`]: [{ body: budget({ state: "warn", pct_today: 84 }) }] });
  const table = page.getByRole("table", { name: "Mức dùng theo tác vụ" });
  await expect(table).toBeVisible();
  expect(await table.locator("thead th").allTextContents()).toEqual(["Tác vụ", "Lượt gọi", "Token vào / ra", "Chi phí ước tính", "Độ trễ p95", "Lỗi", "Chạy rút gọn"]);
  const cell = table.locator("tbody tr").first().locator("td").nth(3);
  await expect(cell).toHaveText("1.240.000 đ");
  expect(await cell.evaluate((e) => [getComputedStyle(e).textAlign, getComputedStyle(e).fontVariantNumeric])).toEqual(["right", expect.stringContaining("tabular-nums")]);
  await expect(page.getByText("Hôm nay 42.000 đ / 80.000 đ · tháng này 1.240.000 đ / 2.000.000 đ")).toBeVisible();
  await expect(page.getByText("Đã dùng 80 % ngân sách ngày.")).toBeVisible();
  expect(await page.locator("[data-part=budget] svg circle, [data-part=budget] [role=progressbar], [data-part=usage-section] [role=progressbar]").count()).toBe(0);
  // rỗng
  await reset(page);
  await script(page, base({ [`GET ${P}/usage`]: [{ body: usage([]) }] }));
  await page.getByRole("radio", { name: "30 ngày" }).click();
  await expect(page.getByText("Chưa có lượt gọi nào trong khoảng này.")).toBeVisible();
});

test("errors: tải lỗi 503 ⇒ thông báo chuẩn + Thử lại 1 request; LLM_NOT_CONFIGURED ⇒ mặc định + Thêm nhà cung cấp", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", { [`GET ${P}/providers`]: [err(503, "NOT_READY"), err(503, "NOT_READY"), err(503, "NOT_READY"), { body: providers() }] });
  const alert = page.getByRole("alert").filter({ hasText: "Chưa tải được cấu hình." });
  await expect(alert).toContainText("Chưa tải được cấu hình.", { timeout: 10_000 });
  await expect(alert).toContainText("Cấu hình hiện có không bị ảnh hưởng.");
  const before = (await getLog(page, `${P}/providers`)).length;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByRole("heading", { name: "Kết nối nhà cung cấp", level: 2 })).toBeVisible();
  expect((await getLog(page, `${P}/providers`)).length).toBe(before + 1);

  await reset(page);
  await script(page, base({ [`GET ${P}/providers`]: [{ body: { items: [], env_fallback: { active: true, providers: ["fake"] } } }] }));
  await page.reload();
  await expect(page.locator("main input[type=password]")).toBeVisible(); // token mất khi tải lại: đúng thiết kế
  await page.locator("main input[type=password]").fill(jwt("ADMIN"));
  await page.getByRole("button", { name: "Dùng token" }).click();
  await expect(page.getByText("Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Thêm nhà cung cấp" }).first()).toBeVisible();
  await page.locator("[data-part=route-table], main").getByRole("button", { name: "Thêm nhà cung cấp" }).last().click();
  await expect(page.locator("[data-part=provider-form]")).toBeVisible();
});

test("errors: mất mạng khi Lưu ⇒ chữ giữ nguyên, Gửi lại cùng Idempotency-Key; 409 ⇒ hỏi giữ bản nào", async ({ page, context }) => {
  await openLlm(page, context, "ADMIN", { [`POST ${P}/providers`]: [{ status: 201, body: provider({ id: "00000000-0000-7000-8000-000000000097", name: "Nhà thử" }) }] });
  await expect(page.locator("[data-part=provider-row]")).toHaveCount(2);
  await page.getByRole("button", { name: "Thêm nhà cung cấp" }).click();
  const form = page.locator("[data-part=provider-form]");
  await form.getByLabel(/Tên hiển thị/).fill("Nhà thử");
  await form.getByLabel(/Khoá API/).fill("good-key");
  const keys: string[] = [];
  let fail = 3;
  await page.route("**/api/v1/admin/llm/providers", async (route) => {
    if (route.request().method() !== "POST") return route.fallback();
    keys.push(route.request().headers()["idempotency-key"]);
    if (fail-- > 0) return route.abort("connectionfailed");
    return route.fallback();
  });
  await form.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect(form.getByRole("button", { name: "Gửi lại" })).toBeVisible();
  await expect(form.getByLabel(/Tên hiển thị/)).toHaveValue("Nhà thử");
  await form.getByRole("button", { name: "Gửi lại" }).click();
  await expect(form).toHaveCount(0);
  expect(new Set(keys).size).toBe(1);
  expect(keys.length).toBeGreaterThanOrEqual(4);

  // 409 trên tuyến
  await script(page, { [`PUT ${P}/routes`]: [err(409, "VERSION_CONFLICT", { details: { current_version: 9 } }), { body: { task: "CHAT", lane: "INTERACTIVE", chain: CHAIN3.slice().reverse(), params: {}, version: 10, reindex_required: false } }] });
  const row = page.locator("[data-part=route-row][data-task=CHAT]");
  await row.getByLabel("Mô hình cho Trả lời chat riêng").selectOption(IDS.m3);
  await expect(row.getByText("Cài đặt này vừa được người khác đổi. Giữ bản của bạn hay dùng bản mới?")).toBeVisible();
  await row.getByRole("button", { name: "Giữ bản của tôi" }).click();
  await expect.poll(async () => (await putRoutes(page)).length).toBe(2);
  expect(JSON.parse((await putRoutes(page))[1].body).version).toBe(9);
});

test("roles: SV / TA bị chặn và 0 request admin/llm; GV chỉ xem; Admin đủ", async ({ page, context }) => {
  const hits: string[] = [];
  page.on("request", (r) => /admin\/llm/.test(r.url()) && hits.push(r.url()));
  await reset(page);
  await script(page, base());
  for (const role of ["student", "ta"] as DemoRole[]) {
    await context.clearCookies();
    await asDemo(context, role);
    await page.goto("/settings/llm");
    await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
    await expect(page.getByText("Trang này dành cho giảng viên và quản trị viên.")).toBeVisible();
  }
  expect(hits).toEqual([]);

  await context.clearCookies();
  await openLlm(page, context, "TEACHER");
  await expect(page.getByRole("heading", { name: "Kết nối nhà cung cấp", level: 2 })).toBeVisible();
  await expect(page.getByText("Chỉ quản trị viên được thay đổi cấu hình.")).toBeVisible();
  await expect(page.locator("main").getByRole("button", { name: /Test kết nối|Xoá|Lưu|Thêm nhà cung cấp|Đổi hạn mức/ })).toHaveCount(0);
  await expect(page.locator("main [role=switch]")).toHaveCount(0);
  for (const sel of await page.locator("main select").all()) await expect(sel).toBeDisabled();
  await expect(page.getByRole("table", { name: "Mức dùng theo tác vụ" })).toBeVisible();

  await context.clearCookies();
  await openLlm(page, context, "ADMIN");
  await expect(page.getByRole("button", { name: "Test kết nối" })).toHaveCount(2);
  await expect(page.locator("main [role=switch], main input[type=checkbox]").first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Đổi hạn mức" })).toBeVisible();
});

test("copy: không có từ kỹ thuật, không toast Thành công (admin và giảng viên)", async ({ page, context }) => {
  for (const role of ["ADMIN", "TEACHER"] as const) {
    await context.clearCookies();
    await openLlm(page, context, role);
    await expect(page.getByRole("heading", { name: "Mô hình tìm kiếm tài liệu", level: 2 })).toBeVisible();
    await page.waitForTimeout(300);
    // tên mô hình là DỮ LIỆU của nhà cung cấp (vd. text-embedding-3-small), không phải lời của màn
    const txt = (await page.locator("main").innerText()).replace(/text-embedding-3-small|gemini-embedding/g, "");
    expect(txt).not.toMatch(/\b(RAG|PII|trace|fallback|embedding|prompt)\b/i);
    await expect(page.locator('[role="status"]:has-text("Thành công")')).toHaveCount(0);
  }
});

test("mobile + keyboard + axe: 375/390 không tràn ngang, vùng chạm, thêm nhà chỉ bằng phím, axe 0 serious", async ({ browser }) => {
  test.setTimeout(120_000);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  for (const w of [375, 390]) {
    const context = await browser.newContext({ viewport: { width: w, height: 844 }, hasTouch: true, isMobile: true, locale: "vi-VN", baseURL: BASE_URL });
    const page = await context.newPage();
    // trên điện thoại chỉ Admin có "Thêm" ở thanh dưới: vào thẳng URL
    await openLlm(page, context, "ADMIN");
    await expect(page.getByRole("heading", { name: "Kết nối nhà cung cấp", level: 2 })).toBeVisible();
    await page.waitForTimeout(300);
    expect(await page.evaluate(TOUCH_SRC), `TOUCH @${w}`).toEqual([]);
    expect(await runAudit(page, AUDIT_SRC), `AUDIT @${w}`).toMatchObject({ ox: 0, cut: [], ell: [] });
    await context.close();
  }
  // bàn phím: mở form thêm, gõ, đi tới Lưu
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "vi-VN", baseURL: BASE_URL });
  const page = await ctx.newPage();
  await openLlm(page, ctx, "ADMIN", { [`POST ${P}/providers`]: [{ status: 201, body: provider({ id: "00000000-0000-7000-8000-000000000096", name: "Bàn phím" }) }] });
  const add = page.getByRole("button", { name: "Thêm nhà cung cấp" });
  await add.focus();
  await page.keyboard.press("Enter");
  const form = page.locator("[data-part=provider-form]");
  await form.getByLabel(/Loại nhà cung cấp/).focus();
  await page.keyboard.press("Tab");
  await page.keyboard.type("Bàn phím");
  await page.keyboard.press("Tab");
  await page.keyboard.type("good-key");
  await form.getByRole("button", { name: "Lưu", exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(form).toHaveCount(0);
  expect((await getLog(page, `${P}/providers`)).filter((l) => l.method === "POST")).toHaveLength(1);
  await ctx.close();
});

test("axe: màn thật ở Admin và Giảng viên (1440 và 390) không có lỗi critical / serious", async ({ page, context }) => {
  test.setTimeout(120_000);
  const bad: string[] = [];
  for (const role of ["ADMIN", "TEACHER"] as const) {
    for (const w of [1440, 390]) {
      await context.clearCookies();
      await page.setViewportSize({ width: w, height: w > 700 ? 900 : 844 });
      await openLlm(page, context, role);
      await expect(page.getByRole("heading", { name: "Mô hình tìm kiếm tài liệu", level: 2 })).toBeVisible();
      await page.waitForTimeout(300);
      const r = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
      for (const v of r.violations) if (v.impact === "critical" || v.impact === "serious") bad.push(`${role}@${w}: ${v.id} ×${v.nodes.length} ${v.nodes[0]?.target.join(" ")}`);
    }
  }
  expect(bad).toEqual([]);
});

// Ca cần gateway thật (không chạy ở CI): hai tab, khoá sai với fake FAKE_LLM_VALID_KEY, 403 khi ép bằng curl.
test("@real two tabs / wrong key / 403", async () => {
  test.skip(true, "@real: cần stack Go (docker-compose.test.yml) — QC chạy tay");
});

