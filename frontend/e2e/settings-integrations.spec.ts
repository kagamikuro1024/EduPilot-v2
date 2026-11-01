import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";
import { imapView, integrationsApi, teamsView } from "./support/integration-fixtures";

// US-P7-05 + US-P7-11: màn `/settings/integrations` THẬT (gateway giả theo hợp đồng: support/integration-fixtures.ts).
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });

const block = (page: Page, kind: "smtp" | "imap" | "teams") => page.locator(`[data-part=integration-block][data-kind=${kind}]`);
const section = (page: Page, kind: "smtp" | "imap" | "teams") => page.locator("[data-ep-panel-section]").filter({ has: block(page, kind) });

async function open(page: Page, context: Parameters<typeof asDemo>[0], role: "admin" | "teacher", opts: Parameters<typeof integrationsApi>[1] = {}) {
  await asDemo(context, role);
  const api = await integrationsApi(page, opts);
  await page.goto("/settings/integrations");
  await expect(page.getByRole("heading", { name: "Tích hợp", level: 1 })).toBeVisible();
  await expect(block(page, "imap")).toBeVisible();
  return api;
}

test("imap key write-only: mật khẩu không vào DOM / storage / console, ô trống khi mở lại", async ({ page, context }, info) => {
  desktopOnly(info);
  const CANARY = "mk-CANARY-abcd1234wxyz";
  const logs: string[] = [];
  page.on("console", (m) => logs.push(m.text()));
  const api = await open(page, context, "admin");

  const pw = block(page, "imap").getByLabel("Mật khẩu", { exact: true });
  await expect(pw).toHaveAttribute("type", "password");
  await expect(pw).toHaveAttribute("autocomplete", "new-password");
  await expect(pw).toHaveValue("");
  await expect(pw).toHaveAttribute("placeholder", "Để trống = giữ cũ");

  await pw.fill(CANARY);
  await block(page, "imap").getByRole("button", { name: "Lưu" }).click();
  await expect.poll(() => api.of("PUT", "/integrations/imap").length).toBe(1);
  const sent = JSON.parse(api.of("PUT", "/integrations/imap")[0].body) as { password?: string; host: string; version: number };
  expect(sent.password).toBe(CANARY);
  expect(sent.host).toBe("imap.ptit.edu.vn");

  await expect(pw).toHaveValue("");
  const dump = await page.evaluate(() => document.body.innerHTML + JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage }) + location.href + document.cookie);
  expect(dump).not.toContain(CANARY);
  expect(dump).not.toContain("wxyz");
  expect(logs.join("\n")).not.toContain("wxyz");
  expect(JSON.stringify(api.calls.filter((c) => c.method === "GET"))).not.toContain("wxyz");

  // lưu lần hai không nhập gì: giữ mật khẩu cũ (không gửi trường `password`)
  await block(page, "imap").getByLabel("Hộp thư").fill("BAIVE");
  await block(page, "imap").getByRole("button", { name: "Lưu" }).click();
  await expect.poll(() => api.of("PUT", "/integrations/imap").length).toBe(2);
  expect(JSON.parse(api.of("PUT", "/integrations/imap")[1].body).password).toBeUndefined();

  await page.reload();
  await expect(block(page, "imap").getByLabel("Mật khẩu", { exact: true })).toHaveValue("");
  await expect(block(page, "imap").getByLabel("Hộp thư")).toHaveValue("BAIVE");
});

test("flat sections: ba khối phẳng trong một panel, không thẻ lồng thẻ", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, "admin");
  await expect(page.locator("main [data-ep-panel]")).toHaveCount(1);
  await expect(page.locator("main [data-ep-panel] [data-ep-panel]")).toHaveCount(0);
  await expect(page.locator("[data-part=integration-block]")).toHaveCount(3);
  const heads = await page.locator("main [data-ep-panel-section] h3:visible").allTextContents();
  expect(heads.map((h) => h.replace(/Môi trường giả lập/, "").trim())).toEqual(["Thư gửi (SMTP)", "Thư nhận (IMAP)", "Teams"]);
  // các khối cách nhau đều nhau (khoảng trắng + kẻ 1 px), không ô vuông riêng
  const gaps = await page.locator("main [data-ep-panel-section]").evaluateAll((els) => els.slice(1).map((e, i) => Math.round(e.getBoundingClientRect().top - els[i].getBoundingClientRect().bottom)));
  expect(Math.max(...gaps) - Math.min(...gaps)).toBeLessThanOrEqual(1);
  expect(await page.locator("main [data-ep-panel-section]").evaluateAll((els) => els.filter((e) => getComputedStyle(e).borderTopWidth !== "0px" && getComputedStyle(e).borderTopWidth !== "1px").length)).toBe(0);
});

test("states: bốn trạng thái, lời test kết nối, đồng bộ ngay 202 rồi 429", async ({ page, context }, info) => {
  desktopOnly(info);
  // 1) Đã kết nối / Chưa kết nối / Lỗi
  await open(page, context, "admin", {
    imap: imapView({ status: "ERROR", last_error: "UNREACHABLE", error_since: "2026-10-23T02:12:00.000Z" }),
    teams: teamsView({ status: "NOT_CONNECTED", configured: false, config: null }),
    imapTest: { ok: false, kind: "AUTH" },
  });
  await expect(section(page, "smtp").locator("[data-part=integration-status]")).toHaveText("Đã kết nối");
  await expect(section(page, "imap").locator("[data-part=integration-status]")).toHaveText("Lỗi");
  await expect(section(page, "teams").locator("[data-part=integration-status]")).toHaveText("Chưa kết nối");
  await expect(block(page, "imap").locator("[data-part=integration-error]")).toHaveText("Không kết nối được từ 09:12 23/10");
  await expect(block(page, "imap").locator("[data-part=imap-last-sync]")).toHaveText("Lần đồng bộ cuối 08:10 23/10");

  await block(page, "imap").getByRole("button", { name: "Test kết nối" }).click();
  await expect(page.locator("[data-part=imap-test]")).toHaveText("Tài khoản hoặc mật khẩu không đúng");
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);

  // 2) Tạm dừng: nói việc cần làm, không câu dài
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await integrationsApi(page, { imap: imapView({ status: "PAUSED", last_error: "AUTH", error_since: "2026-10-23T02:12:00.000Z" }) });
  await page.reload();
  await expect(section(page, "imap").locator("[data-part=integration-status]")).toHaveText("Tạm dừng");
  await expect(block(page, "imap").locator("[data-part=integration-error]")).toHaveText("Cần nhập lại mật khẩu");

  // 3) Giảng viên: chỉ xem, chỉ có `Đồng bộ ngay`; 202 → kết quả, lần hai 429 → nút khoá kèm lý do
  const p2 = await context.newPage();
  await asDemo(context, "teacher");
  let throttled = false;
  const api = await integrationsApi(p2, {
    h: {
      "POST integrations/imap/sync-now": (r, dflt) =>
        throttled ? r.fulfill(json({ code: "RATE_LIMITED", message: "x", trace_id: "t", retry_after: 24 }, 429)) : dflt(r),
    },
  });
  await p2.goto("/settings/integrations");
  await expect(block(p2, "imap")).toBeVisible();
  await expect(block(p2, "imap").getByLabel("Mật khẩu", { exact: true })).toHaveCount(0);
  await expect(block(p2, "imap").getByRole("button", { name: "Lưu" })).toHaveCount(0);
  await expect(block(p2, "imap").getByRole("button", { name: "Test kết nối" })).toHaveCount(0);
  await p2.getByRole("button", { name: "Đồng bộ ngay" }).click();
  await expect(p2.locator("[data-part=sync-result]")).toHaveText("Nhận 5 · tạo 3 · khớp 2 · chưa khớp 1 · bỏ qua 1 · lỗi 0");
  expect(api.of("POST", "/integrations/imap/sync-now")[0].headers["idempotency-key"]).toBeTruthy();

  throttled = true;
  await p2.getByRole("button", { name: "Đồng bộ ngay" }).click();
  const sync = p2.getByRole("button", { name: "Đồng bộ ngay" });
  await expect(sync).toBeDisabled();
  await expect(sync).toHaveAttribute("title", "Chờ 24 giây nữa");
  await p2.close();
});

test("save conflicts: 422 lỗi dưới ô, 409 báo bằng chữ rồi nạp lại giá trị mới", async ({ page, context }, info) => {
  desktopOnly(info);
  // 422: lần đầu lưu mà chưa nhập mật khẩu ⇒ lỗi ngay dưới ô Mật khẩu, các ô khác giữ chữ
  await open(page, context, "admin", { imap: imapView({ configured: false }) });
  await block(page, "imap").getByLabel("Tài khoản").fill("hopthu@ptit.edu.vn");
  await block(page, "imap").getByRole("button", { name: "Lưu" }).click();
  await expect(block(page, "imap").getByText("Lần đầu cần nhập mật khẩu.")).toBeVisible();
  await expect(block(page, "imap").getByLabel("Mật khẩu", { exact: true })).toHaveAttribute("aria-invalid", "true");
  await expect(block(page, "imap").getByLabel("Tài khoản")).toHaveValue("hopthu@ptit.edu.vn");

  // 409: người khác vừa đổi ⇒ dòng báo + nạp lại giá trị mới của máy chủ
  await page.unrouteAll({ behavior: "ignoreErrors" });
  let served = 0;
  await integrationsApi(page, {
    h: {
      "GET integrations/imap": (r) =>
        r.fulfill(json(served++ === 0 ? imapView({ version: 9 }) : imapView({ version: 10, config: { host: "imap-moi.ptit.edu.vn", port: 993, security: "TLS", username: "nopbai@ptit.edu.vn", mailbox: "INBOX", poll_seconds: 60 } }))),
      "PUT integrations/imap": (r) => r.fulfill(json({ code: "VERSION_CONFLICT", message: "x", trace_id: "t", details: { current_version: 10 } }, 409)),
    },
  });
  await page.reload();
  await block(page, "imap").getByLabel("Máy chủ").fill("imap-cua-toi.ptit.edu.vn");
  await block(page, "imap").getByRole("button", { name: "Lưu" }).click();
  await expect(page.getByText("Cấu hình vừa được người khác đổi")).toBeVisible();
  await expect(block(page, "imap").getByLabel("Máy chủ")).toHaveValue("imap-moi.ptit.edu.vn");
});

test("no description lines: không câu mô tả dưới tiêu đề, không icon Thông tin", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, "admin");
  const text = await page.locator("main").innerText();
  expect(text).not.toMatch(/Thư đi, thư đến và Teams|Chỉ quản trị viên hệ thống đổi được|sinh viên và giảng viên nhận được|Bắt đầu từ thư đi/);
  // không đoạn văn dài (câu mô tả) trong các khối
  const long = await page.locator("[data-part=integration-block] p").evaluateAll((ps) => ps.map((p) => p.textContent ?? "").filter((t) => t.trim().length > 60));
  expect(long).toEqual([]);
  await expect(page.locator("main [data-part=integration-block] svg[aria-label*='Thông tin'], main [data-icon=info]")).toHaveCount(0);
});

test("teams card sandbox label: nhãn Môi trường giả lập cạnh tên thẻ", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, "admin", { teams: teamsView({ sandbox: true }) });
  await expect(section(page, "teams").locator("[data-part=teams-sandbox]")).toHaveText("Môi trường giả lập");
  await expect(section(page, "teams").locator("h3")).toContainText("Teams");
  await expect(block(page, "teams").getByLabel("Client secret")).toHaveAttribute("type", "password");
  await expect(block(page, "teams").getByLabel("Client secret")).toHaveValue("");
  await expect(block(page, "teams").getByLabel("Tenant")).toHaveValue("ptit.edu.vn");
});

test("teams consent guidance: lỗi cấp quyền nói bằng lời thường", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, "admin", { teamsTest: { ok: false, kind: "TEAMS_CONSENT_REQUIRED" } });
  await block(page, "teams").getByRole("button", { name: "Kiểm tra kết nối" }).click();
  const msg = page.locator("[data-part=teams-test]");
  await expect(msg).toContainText("Cần quản trị Microsoft 365 của trường cấp quyền đọc bài tập và danh sách lớp");
  await expect(msg).toContainText("EduAssignments");
  expect(await msg.innerText()).not.toMatch(/consent|scope|OAuth|token/i);
});

test("teams disabled: trường chưa bật thì thẻ nói Chưa bật và nút khoá kèm lý do", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, "admin", { teams: teamsView({ available: false, status: "NOT_CONNECTED" }) });
  await expect(section(page, "teams").locator("[data-part=integration-status]")).toHaveText("Chưa bật");
  const test = block(page, "teams").getByRole("button", { name: "Kiểm tra kết nối" });
  await expect(test).toBeDisabled();
  await expect(test).toHaveAttribute("title", "Trường chưa bật Teams");
  await expect(block(page, "teams").getByRole("button", { name: "Lưu" })).toBeDisabled();
});
