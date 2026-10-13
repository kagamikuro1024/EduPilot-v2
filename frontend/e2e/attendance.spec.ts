import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { NOW, attApi, conflict, err, gridBody, json, sessionRow, sid, student, students, COURSE } from "./support/att-fixtures";
import { asDemo } from "./support/session";

// US-P5-02: điểm danh THẬT. Gateway giả theo hợp đồng thật; `attApi` giữ trạng thái như máy chủ (đổi trạng thái → version + 1).
const SUB = "00000000-0000-7000-8000-0000000000a0"; // `sub` của JWT giả (support/session.ts)
const QKEY = (sessionNo = 2) => `ep:draft:${SUB}:attendance:${COURSE}:00000000-0000-7000-8000-0000000005a${sessionNo}`;

async function open(page: Page, context: BrowserContext, o: Parameters<typeof attApi>[1] = {}, url = "/attendance") {
  await asDemo(context, "teacher");
  await page.clock.setFixedTime(NOW);
  const api = await attApi(page, o);
  await page.goto(url);
  await expect(page.locator("[data-part=student-row]")).not.toHaveCount(0);
  return api;
}
const row = (page: Page, i: number) => page.locator(`[data-part=student-row][data-student-id="${sid(i)}"]`);
const radio = (page: Page, i: number, label: string) => row(page, i).getByRole("radio", { name: `${label} — Sinh Viên ${String(i).padStart(2, "0")}`, exact: true });
const bodyOf = (c: { body: string }) => JSON.parse(c.body) as { records: Array<{ student_id: string; status: string; version?: number }>; complete: boolean };
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");

test("attendance session picker: mặc định buổi đang diễn ra, gần nhất trước, nhãn Đã điểm danh / Chưa, ?session= chọn buổi", async ({ page, context }, info) => {
  desktopOnly(info);
  const list = [sessionRow(1, { attendance: { done_at: "2026-10-05T04:40:00Z", has_records: true } }), sessionRow(2), sessionRow(3)];
  await open(page, context, { sessions: list });
  const sel = page.locator("[data-part=att-session-select]");
  await expect(sel).toHaveValue(sessionRow(2).id);
  await expect(sel.locator("option")).toHaveText(["Buổi 3 · 09/10 · Chưa", "Buổi 2 · 07/10 · Chưa", "Buổi 1 · 05/10 · Đã điểm danh"]);
  await page.goto(`/attendance?session=${sessionRow(1).id}`);
  await expect(page.locator("[data-part=att-session-select]")).toHaveValue(sessionRow(1).id);
});

test("attendance empty no sessions: 'Lớp chưa có lịch buổi học.' và nút về /calendar", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  await attApi(page, { sessions: [] });
  await page.goto("/attendance");
  await expect(page.getByText("Lớp chưa có lịch buổi học.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Tạo lịch buổi học" })).toHaveAttribute("href", "/calendar");
});

test("attendance states: tải lưới lỗi nói chuyện gì xảy ra và có Thử lại", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  let n = 0;
  await attApi(page, { h: { "GET attendance": (r) => (++n === 1 ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json(gridBody()))) } });
  await page.goto("/attendance");
  const retry = page.getByRole("button", { name: "Thử lại" });
  await expect(retry).toBeVisible();
  await retry.click();
  await expect(page.locator("[data-part=student-row]")).toHaveCount(30);
});

test("attendance grid keyboard: 1–4 đặt trạng thái rồi xuống hàng kế, ↑↓ đổi hàng, Shift+số giữ hàng, P ghi phát biểu; 30 hàng gửi MỘT lần", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  await page.getByLabel(/Bảng điểm danh/).focus();
  await page.keyboard.press("4"); // hàng 1 vắng → xuống hàng 2
  await page.keyboard.press("2"); // hàng 2 muộn → xuống hàng 3
  await page.keyboard.press("ArrowUp"); // về hàng 2
  await page.keyboard.press("Shift+3"); // hàng 2 vắng phép, giữ hàng
  await page.keyboard.press("p"); // phát biểu hàng 2
  await expect(radio(page, 1, "Vắng")).toBeChecked();
  await expect(radio(page, 2, "Vắng phép")).toBeChecked();
  for (let i = 3; i <= 30; i++) await page.keyboard.press("ArrowDown").then(() => page.keyboard.press("1"));
  await expect.poll(() => api.puts().length, { timeout: 8000 }).toBe(1);
  const b = bodyOf(api.puts()[0]);
  expect(b.complete).toBe(false);
  expect(b.records.find((r) => r.student_id === sid(1))?.status).toBe("ABSENT");
  expect(b.records.find((r) => r.student_id === sid(2))?.status).toBe("EXCUSED");
  expect(api.calls.filter((c) => c.path.endsWith("/participation") && c.method === "POST")).toHaveLength(1);
});

test("attendance optimistic undo: đổi ngay, dòng 'Đã đánh Vắng · … Hoàn tác' tại chỗ, không toast, không hộp thoại; Hoàn tác trả trạng thái trước", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context);
  await radio(page, 3, "Vắng").check();
  await expect(radio(page, 3, "Vắng")).toBeChecked(); // ≤ 100 ms: không chờ mạng
  await expect(page.locator("[data-part=undo-line], main").getByText("Đã đánh Vắng · Sinh Viên 03")).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(radio(page, 3, "Có mặt")).toBeChecked();
});

test("attendance save indicator: Đang lưu… rồi Đã lưu HH:MM sau 2 s yên lặng, một PUT complete:false với version đúng", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  const state = page.locator("[data-part=att-save-state]");
  await expect(state).toHaveText("Chưa có thay đổi");
  await radio(page, 4, "Vắng").check();
  await expect(state).toHaveText("Đang lưu…");
  await expect(state).toHaveText(/^Đã lưu \d{2}:\d{2}$/, { timeout: 8000 });
  expect(api.puts()).toHaveLength(1);
  const b = bodyOf(api.puts()[0]);
  expect(b.complete).toBe(false);
  expect(b.records).toEqual([{ student_id: sid(4), status: "ABSENT" }]); // hàng chưa có bản ghi: không gửi version
  await radio(page, 4, "Muộn").check(); // lần hai: version lấy từ phản hồi lần một
  await expect.poll(() => api.puts().length, { timeout: 8000 }).toBe(2);
  expect(bodyOf(api.puts()[1]).records).toEqual([{ student_id: sid(4), status: "LATE", version: 1 }]);
});

test("attendance 5xx keeps changes: PUT lỗi giữ lựa chọn, 'Chưa lưu được. Thử lại' rồi lưu thành công", async ({ page, context }, info) => {
  desktopOnly(info);
  let fail = true;
  const api = await open(page, context, { h: { "PUT attendance": (r, dflt) => (fail ? r.fulfill(err(503, "SERVICE_UNAVAILABLE")) : dflt(r)) } });
  await radio(page, 5, "Vắng").check();
  const notice = page.getByText("Chưa lưu được. Thử lại");
  await expect(notice).toBeVisible({ timeout: 8000 });
  await expect(radio(page, 5, "Vắng")).toBeChecked();
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.locator("[data-part=att-save-state]")).toHaveText(/^Đã lưu/, { timeout: 8000 });
  await expect(radio(page, 5, "Vắng")).toBeChecked();
  expect(api.puts().length).toBeGreaterThanOrEqual(2);
});

test("attendance complete inline confirm: Lưu điểm danh gửi complete:true một lần, dòng 'Đã lưu điểm danh · 30 sinh viên · vắng 3', nút thành Đã lưu, không toast / hộp thoại", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  for (const i of [3, 7, 9]) await radio(page, i, "Vắng").check();
  await radio(page, 11, "Muộn").check();
  await page.getByRole("button", { name: "Lưu điểm danh" }).click();
  await expect(page.getByText("Đã lưu điểm danh · 30 sinh viên · vắng 3")).toBeVisible();
  await expect(page.getByRole("button", { name: "Đã lưu" })).toBeDisabled();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(api.puts()).toHaveLength(1); // một lời gọi ghi cả lưới
  const b = bodyOf(api.puts()[0]);
  expect(b.complete).toBe(true);
  expect(b.records).toHaveLength(30);
  expect(b.records.filter((r) => r.status !== "PRESENT").map((r) => [r.student_id, r.status])).toEqual([[sid(3), "ABSENT"], [sid(7), "ABSENT"], [sid(9), "ABSENT"], [sid(11), "LATE"]]);
  await radio(page, 12, "Vắng").check(); // thay đổi mới mở lại nút
  await expect(page.getByRole("button", { name: "Lưu điểm danh" })).toBeEnabled();
});

test("attendance complete không có thay đổi vẫn gửi complete:true cho cả lưới (mặc định mọi người có mặt)", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  await page.getByRole("button", { name: "Lưu điểm danh" }).click();
  await expect(page.getByText("Đã lưu điểm danh · 30 sinh viên · vắng 0")).toBeVisible();
  expect(bodyOf(api.puts()[0]).records.every((r) => r.status === "PRESENT")).toBe(true);
  expect(bodyOf(api.puts()[0]).records).toHaveLength(30);
});

test("attendance speak plus undo: + phát biểu · +0,25 tăng đếm ngay, POST có client_event_id, ba lần bấm = ba sự kiện, Hoàn tác = DELETE", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  const speak = row(page, 6).getByRole("button", { name: "+ phát biểu · +0,25" });
  await speak.click();
  await expect(row(page, 6).locator("[data-part=speak-count]")).toHaveText("1");
  await speak.click();
  await speak.click();
  await expect(row(page, 6).locator("[data-part=speak-count]")).toHaveText("3");
  await expect.poll(() => api.calls.filter((c) => c.method === "POST").length).toBe(3);
  const ids = api.calls.filter((c) => c.method === "POST").map((c) => (JSON.parse(c.body) as { client_event_id: string; type: string }).client_event_id);
  expect(new Set(ids).size).toBe(3);
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(row(page, 6).locator("[data-part=speak-count]")).toHaveText("2");
  await expect.poll(() => api.calls.filter((c) => c.method === "DELETE").length).toBe(1);
});

test("attendance two teachers conflict: 409 → 'Thầy Nam vừa đổi 1 dòng', chọn từng dòng, giữ phần còn lại; Lấy của Thầy Nam bỏ bản của mình, Giữ của tôi gửi lại với version mới", async ({ page, context }, info) => {
  desktopOnly(info);
  let calls = 0;
  const api = await open(page, context, {
    h: {
      "PUT attendance": (r, dflt) => {
        calls++;
        if (calls === 1) return r.fulfill(json({ code: "VERSION_CONFLICT", message: "m", trace_id: "b".repeat(32), details: { conflicts: [conflict(3, "ABSENT", "LATE"), conflict(8, "ABSENT", "EXCUSED")] } }, 409));
        return dflt(r);
      },
    },
  });
  for (const i of [3, 8, 9]) await radio(page, i, "Vắng").check();
  await expect(page.getByText("Thầy Nam vừa đổi 2 dòng")).toBeVisible({ timeout: 8000 });
  await expect(page.getByRole("button", { name: "Lưu điểm danh" })).toBeDisabled();
  await row(page, 3).getByRole("button", { name: "Lấy của Thầy Nam" }).click();
  await expect(radio(page, 3, "Muộn")).toBeChecked();
  await row(page, 8).getByRole("button", { name: "Giữ của tôi" }).click();
  await expect(radio(page, 8, "Vắng")).toBeChecked();
  await expect(radio(page, 9, "Vắng")).toBeChecked(); // dòng không xung đột giữ nguyên
  await expect.poll(() => api.puts().length, { timeout: 8000 }).toBe(2);
  const second = bodyOf(api.puts()[1]);
  expect(second.records.map((r) => r.student_id).sort()).toEqual([sid(8), sid(9)].sort());
  expect(second.records.find((r) => r.student_id === sid(8))?.version).toBe(2);
});

test("attendance offline queue sync: mất mạng đánh vắng 3 người → banner 'Chưa có mạng · 3 thay đổi chờ lưu'; có mạng lại tự gửi đủ 3", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  await context.setOffline(true);
  for (const i of [3, 4, 5]) await radio(page, i, "Vắng").check();
  await expect(page.locator("[data-part=offline-banner]")).toContainText("Chưa có mạng · 3 thay đổi chờ lưu");
  const stored = await page.evaluate((k) => localStorage.getItem(k), QKEY());
  expect(Object.keys(JSON.parse(stored ?? "{}").dirty ?? {})).toHaveLength(3);
  expect(api.puts()).toHaveLength(0);
  await context.setOffline(false);
  await expect.poll(() => api.puts().length, { timeout: 15000 }).toBe(1);
  expect(bodyOf(api.puts()[0]).records.map((r) => r.status)).toEqual(["ABSENT", "ABSENT", "ABSENT"]);
  await expect(page.locator("[data-part=offline-banner]")).not.toContainText("thay đổi chờ lưu");
});

test("attendance offline reload keeps queue: tải lại giữa chừng vẫn thấy lựa chọn và gửi tiếp", async ({ page, context }, info) => {
  desktopOnly(info);
  let down = true;
  const api = await open(page, context, { h: { "PUT attendance": (r, dflt) => (down ? r.abort("internetdisconnected") : dflt(r)) } });
  await radio(page, 6, "Vắng").check();
  await expect.poll(() => page.evaluate((k) => localStorage.getItem(k) != null, QKEY())).toBe(true);
  await page.reload();
  await expect(radio(page, 6, "Vắng")).toBeChecked();
  down = false;
  await expect.poll(() => api.puts().length, { timeout: 15000 }).toBeGreaterThanOrEqual(1);
  expect(bodyOf(api.puts().at(-1)!).records).toEqual([{ student_id: sid(6), status: "ABSENT" }]);
});

test("attendance logout asks first: còn thay đổi chưa lưu thì hỏi; Ở lại giữ nguyên, Đăng xuất xoá khoá attendance của người đó ở máy", async ({ page, context }, info) => {
  desktopOnly(info);
  await open(page, context, { h: { "PUT attendance": (r) => r.abort("internetdisconnected") } });
  await radio(page, 2, "Vắng").check();
  await expect.poll(() => page.evaluate((k) => localStorage.getItem(k) != null, QKEY())).toBe(true);
  const menu = async () => {
    await page.getByRole("button", { name: /Tài khoản|teacher|Người Thử/i }).first().click();
    await page.getByRole("menuitem", { name: "Đăng xuất" }).click();
  };
  await menu();
  const dlg = page.getByRole("dialog");
  await expect(dlg.getByText("Còn điểm danh chưa lưu")).toBeVisible();
  await dlg.getByRole("button", { name: "Ở lại" }).click();
  await expect(radio(page, 2, "Vắng")).toBeChecked();
  await menu();
  await page.getByRole("dialog").getByRole("button", { name: "Đăng xuất" }).click();
  await expect(page).toHaveURL(/\/login/);
  const left = await page.evaluate(() => Object.keys(localStorage).filter((k) => k.includes(":attendance:")));
  expect(left).toEqual([]);
});

test("attendance switch session keeps changes: đổi buổi khi còn thay đổi chưa lưu thì vẫn gửi phần của buổi cũ", async ({ page, context }, info) => {
  desktopOnly(info);
  const api = await open(page, context);
  await radio(page, 3, "Vắng").check();
  await page.locator("[data-part=att-session-select]").selectOption(sessionRow(1).id);
  await expect.poll(() => api.puts().length, { timeout: 8000 }).toBe(1);
  expect(api.puts()[0].path).toContain(sessionRow(2).id);
  expect(bodyOf(api.puts()[0]).records).toEqual([{ student_id: sid(3), status: "ABSENT" }]);
});

test("attendance 375: hàng chỉ có chữ trạng thái, bấm hàng mở bốn lựa chọn ≥ 44 px, đổi một người ≤ 2 chạm, không tràn ngang", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 375, height: 812 });
  const api = await open(page, context, { grid: gridBody(2, [student(1), student(2, { record: { status: "ABSENT", note: null, version: 1, marked_by: { full_name: "Thầy Nam" }, marked_at: "2026-10-07T02:30:00Z" } }), student(3)]) }, "/attendance");
  await expect(page.locator("[data-part=att-row]").first()).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect(page.locator("[data-part=att-row][data-student-id]").nth(1).locator("[data-part=att-status]")).toHaveText("Vắng");
  let taps = 0;
  await page.locator("[data-part=att-row]").first().getByRole("button").first().click();
  taps++;
  const opts = page.getByRole("radiogroup").first().getByRole("radio");
  await expect(opts).toHaveCount(4);
  for (const o of await opts.all()) expect((await o.boundingBox())!.height).toBeGreaterThanOrEqual(44);
  await opts.nth(3).click();
  taps++;
  expect(taps).toBeLessThanOrEqual(2);
  await expect(page.locator("[data-part=att-row]").first().locator("[data-part=att-status]")).toHaveText("Vắng");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(api.puts()).toHaveLength(0); // chưa tới 2 s
});

test("attendance 375 scripted under 15s: 3 vắng, 1 muộn, 1 phát biểu rồi Lưu điểm danh — đúng một PUT cho cả lưới, mỗi chạm phản hồi ≤ 100 ms", async ({ page, context }, info) => {
  desktopOnly(info);
  await page.setViewportSize({ width: 375, height: 812 });
  const api = await open(page, context, {}, "/attendance");
  await page.locator("[data-part=att-row]").first().waitFor();
  const t0 = Date.now();
  const rows = page.locator("[data-part=att-row]");
  async function mark(i: number, label: string) {
    await rows.nth(i).getByRole("button").first().click();
    const t = Date.now();
    await page.getByRole("radiogroup").getByRole("radio", { name: label, exact: true }).click();
    await expect(rows.nth(i).locator("[data-part=att-status]")).toHaveText(label);
    expect(Date.now() - t).toBeLessThan(1000); // trần rộng cho CI; mốc ≤ 100 ms đo ở TC tay
  }
  for (const i of [2, 5, 8]) await mark(i, "Vắng");
  await mark(10, "Muộn");
  await rows.nth(12).getByRole("button").first().click();
  await page.getByRole("button", { name: "+ phát biểu · +0,25" }).click();
  await page.getByRole("button", { name: "Lưu điểm danh" }).click();
  await expect(page.getByText("Đã lưu điểm danh · 30 sinh viên · vắng 3")).toBeVisible();
  expect(Date.now() - t0).toBeLessThan(15000);
  const last = bodyOf(api.puts().at(-1)!); // tự lưu giữa chừng (nếu có) không tính: lời gọi CUỐI ghi cả lưới
  expect(last.complete).toBe(true);
  expect(last.records).toHaveLength(30);
  expect(last.records.filter((r) => r.status === "ABSENT")).toHaveLength(3);
  expect(api.puts().filter((c) => bodyOf(c).complete)).toHaveLength(1);
  expect(students().length).toBe(30);
});
