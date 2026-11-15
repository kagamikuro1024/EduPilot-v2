import { expect, test } from "@playwright/test";
import { call, login, needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — US-P10-01/02 giao diện thật trên ep-qc (@real): /observability ADMIN + GV, drawer khoá → lý do → nội dung.
needReal();

test("@real OBS-1 ADMIN: dải 24 giờ, bảng 7 ngày, bảng yêu cầu, lọc (TC-01-03/04/07)", async ({ page, request }) => {
  const admin = await login(request, "admin");
  await uiLogin(page, "admin");
  await page.goto("/observability");
  const today = page.locator('[data-part="obs-today"]');
  await expect(today).toHaveCount(1);
  await expect(today).toHaveText(/^24 giờ qua: [\d.,]+ yêu cầu · p95 [\d.,]+ ?\w* · lỗi [\d.,]+ ?% · dự phòng [\d.,]+ ?% · chi phí/);
  await expect(page.getByText("7 ngày qua", { exact: true })).toBeVisible();
  await expect(page.getByText("Chi phí theo tác vụ", { exact: true })).toBeVisible();
  const heads = (await page.getByRole("table", { name: /Số liệu theo ngày/ }).innerText()).toLowerCase();
  for (const h of ["yêu cầu", "p95", "lỗi"]) expect(heads, `cột ${h} ở bảng ngày`).toContain(h);
  await shot(page, "obs-admin-1440");
  // lọc «Có vấn đề»: bảng khớp API
  await page.getByRole("radio", { name: "Có vấn đề" }).click().catch(async () => page.getByText("Có vấn đề", { exact: true }).click());
  await page.waitForTimeout(1500);
  const api = await call(request, admin, "GET", "/admin/observability/requests?filter=problem&limit=50");
  const rows = await page.locator("table").last().locator("tbody tr").count();
  expect(rows, "số hàng sau lọc «Có vấn đề» khớp API (≤ 50, cùng khoảng 24 giờ)").toBeGreaterThan(0);
  expect(((api.json as { items: unknown[] }).items).length).toBeGreaterThanOrEqual(rows);
  const body = await page.locator("table").last().innerText();
  expect(body).not.toMatch(/prompt|nội dung câu hỏi|masked/i);
  await shot(page, "obs-admin-problem");
  await page.getByText("Thông tin cá nhân", { exact: true }).click();
  await page.waitForTimeout(1200);
  await shot(page, "obs-admin-privacy");
});

test("@real OBS-2 GV: chỉ lớp của mình, không bảng yêu cầu / chi phí (TC-01-08, TC-00-RBAC-4)", async ({ page, browser }) => {
  await uiLogin(page, "teacher");
  await page.goto("/observability");
  await expect(page.getByRole("heading", { name: "Quan sát AI" })).toBeVisible();
  await expect(page.getByText("Toàn hệ thống")).toHaveCount(0);
  await expect(page.getByText("Yêu cầu gửi tới model")).toHaveCount(0);
  await expect(page.getByText("Chi phí theo tác vụ")).toHaveCount(0);
  await shot(page, "obs-teacher-1440");
  for (const u of ["ta", "sv.kha"]) {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, baseURL: process.env.QC_BASE ?? "https://localhost:943" });
    const p = await ctx.newPage();
    await uiLogin(p, u);
    await p.goto("/observability");
    await p.waitForTimeout(1500);
    const txt = await p.locator("body").innerText();
    expect(txt, `${u} vào thẳng /observability không thấy số liệu AI`).not.toMatch(/24 giờ qua:|Yêu cầu gửi tới model|Chi phí theo tác vụ/);
    await shot(p, `obs-${u}-direct`);
    await ctx.close();
  }
});

test("@real OBS-3 drawer: khoá → lý do → nội dung, audit_log, hết hạn (TC-02-01/05/13/14/18)", async ({ page, request }) => {
  const admin = await login(request, "admin");
  await uiLogin(page, "admin");
  const opens: string[] = [];
  page.on("request", (r) => { if (/observability\/requests\/[^/]+\/open/.test(r.url())) opens.push(r.url()); });
  await page.goto("/observability");
  const last = page.getByRole("table", { name: /Yêu cầu gửi tới model/ });
  await expect(last.locator("tbody tr").first()).toBeVisible();
  await last.locator("tbody tr").first().click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toBeVisible();
  expect(opens.length, "mở drawer chưa gọi API open").toBe(0);
  await expect(dlg.getByText("Nội dung đã che")).toHaveCount(0);
  const title = (await dlg.getByRole("heading").first().innerText());
  const rid8 = title.split("·").pop()!.trim();
  // TC-02-05
  const ta = dlg.getByRole("textbox");
  await ta.fill("12345");
  await ta.blur();
  await expect(dlg.getByRole("button", { name: "Xem chi tiết" })).toBeDisabled();
  await expect(dlg.getByText("Ít nhất 10 ký tự.")).toBeVisible();
  expect(opens.length).toBe(0);
  // TC-02-18
  const full = sql(`select id from llm_audit where id::text like '${rid8}%' order by created_at desc limit 1`)[0][0]; // chuỗi 8 ký tự UUIDv7 trùng theo thời gian → lấy hàng mới nhất khớp
  if (sql(`select count(*) from llm_request_payloads where audit_id='${full}'`)[0][0] === "0") sql(`insert into llm_request_payloads(audit_id, course_id, masked_prompt, tools, confidence) select id, course_id, '[STUDENT_1] hỏi về điểm quá trình (QC fixture)', '[]'::jsonb, 0.5 from llm_audit where id='${full}'`); // QC fixture: bảo đảm có payload để kiểm nút sao chép
  const hasPayload = true;
  const before = Number(sql(`select count(*) from audit_log where entity='llm_request' and entity_id='${full}'`)[0][0]);
  const reason = `QC kiểm giao diện ${Date.now()}`;
  await ta.fill(reason);
  await dlg.getByRole("button", { name: "Xem chi tiết" }).click();
  if (hasPayload) {
    await expect(dlg.getByText("Nội dung đã che")).toBeVisible();
    await expect(dlg.getByText("Mã theo dõi")).toBeVisible();
    await shot(page, "obs-drawer-open");
    // TC-02-13: JAEGER_BASE_URL trống → trace_id + nút sao chép, không <a> rỗng/undefined
    const links = await dlg.locator("a").evaluateAll((a) => a.map((x) => (x as HTMLAnchorElement).getAttribute("href")));
    expect(links.filter((h) => !h || /undefined/.test(h)), "không có liên kết rỗng/undefined").toEqual([]);
    await expect.soft(dlg.getByRole("button", { name: /sao chép/i }), "TC-02-13: nút sao chép trace_id").toBeVisible();
  } else {
    await expect(dlg.getByText("Nội dung đã hết hạn lưu.")).toBeVisible();
  }
  expect(opens.length).toBe(1);
  const after = Number(sql(`select count(*) from audit_log where entity='llm_request' and entity_id='${full}'`)[0][0]);
  expect(after, "đúng một dòng audit_log mới").toBe(before + 1);
  const au = await call(request, admin, "GET", `/admin/audit?entity=llm_request&limit=5`);
  const items = (au.json as { items: { entity_id: string; action: string }[] }).items;
  expect(items.some((i) => i.entity_id === full && i.action === "VIEW_DETAIL"), "/admin/audit có dòng VIEW_DETAIL của hàng này").toBe(true);
});

test("@real OBS-4 hết hạn lưu: hàng đã bị dọn → «Nội dung đã hết hạn lưu» + vẫn ghi audit (TC-02-14 giao diện)", async ({ page }) => {
  await uiLogin(page, "admin");
  await page.goto("/observability");
  const last = page.getByRole("table", { name: /Yêu cầu gửi tới model/ });
  await expect(last.locator("tbody tr").first()).toBeVisible();
  await last.locator("tbody tr").first().click();
  const dlg = page.getByRole("dialog");
  const rid8 = (await dlg.getByRole("heading").first().innerText()).split("·").pop()!.trim();
  const full = sql(`select id from llm_audit where id::text like '${rid8}%' order by created_at desc limit 1`)[0][0]; // chuỗi 8 ký tự UUIDv7 trùng theo thời gian → lấy hàng mới nhất khớp
  sql(`delete from llm_request_payloads where audit_id='${full}'`);
  const before = Number(sql(`select count(*) from audit_log where entity='llm_request' and entity_id='${full}'`)[0][0]);
  await dlg.getByRole("textbox").fill("QC kiểm payload đã hết hạn lưu");
  await dlg.getByRole("button", { name: "Xem chi tiết" }).click();
  await expect(dlg.getByText("Nội dung đã hết hạn lưu.")).toBeVisible();
  await shot(page, "obs-drawer-expired");
  expect(Number(sql(`select count(*) from audit_log where entity='llm_request' and entity_id='${full}'`)[0][0])).toBe(before + 1);
});

test("@real OBS-5 lỗi tải + 375 px (TC-01-11, TC-01-14)", async ({ page }) => {
  await uiLogin(page, "admin");
  let fail = true;
  await page.route("**/admin/observability/summary*", (r) => (fail ? r.fulfill({ status: 500, json: { code: "INTERNAL", message: "x" } }) : r.continue()));
  await page.goto("/observability");
  const retry = page.getByRole("button", { name: /Thử lại/ }).first();
  await expect(retry).toBeVisible();
  await shot(page, "obs-error");
  fail = false;
  await retry.click();
  await expect(page.locator('[data-part="obs-today"]')).toBeVisible();
  await page.setViewportSize({ width: 375, height: 812 });
  await page.reload();
  await expect(page.locator('[data-part="obs-today"]')).toBeVisible();
  await noOverflow(page);
  await shot(page, "obs-admin-375");
});
