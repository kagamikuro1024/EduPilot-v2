import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, DID, doc, docApi, err, json, libDetail, libItem } from "./support/doc-fixtures";
import { SID } from "./support/chat-fixtures";

// US-P8-02: /library THẬT (gateway giả theo hợp đồng thật).
const L = `/courses/${COURSE}/library`;
const list = (items: unknown[] = [libItem()]) => json({ items, next_cursor: null });

async function open(page: Page, context: BrowserContext, h: Parameters<typeof docApi>[1] = {}) {
  await asDemo(context, "student", { person: "sv-2" });
  const calls = await docApi(page, { [`GET ${L}`]: list(), ...h });
  await page.goto("/library");
  await page.getByRole("search").waitFor();
  return calls;
}

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca 375 riêng");
});
void doc;

test("layout: ô tìm kiếm trước, dòng gọn có dấu loại tệp bằng chữ, hành động Xem, không nút Luyện đề này", async ({ page, context }) => {
  await open(page, context, { [`GET ${L}`]: list([libItem(), libItem({ id: "d2", title: "Slide TLS", file_kind: "PPTX", can_ask_ai: false })]) });
  await expect(page.getByLabel("Tìm tài liệu")).toBeFocused();
  await expect(page.getByText("PDF", { exact: true })).toBeVisible();
  await expect(page.getByText("PPTX", { exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Xem" })).toHaveCount(2);
  await expect(page.getByRole("button", { name: /Luyện đề này/ })).toHaveCount(0);
  await expect(page.getByRole("link", { name: /Luyện đề này/ })).toHaveCount(0);
  const body = await page.locator("main").innerText();
  expect(body).not.toMatch(/\b(RAG|chunk|embedding|ANSWER_KEY)\b/i);
});

test("states: rỗng, bộ lọc không khớp, lỗi có Thử lại", async ({ page, context }) => {
  await open(page, context, { [`GET ${L}`]: list([]) });
  await expect(page.getByText("Chưa có tài liệu nào")).toBeVisible();
  await page.unroute(`**/api/v1/courses/${COURSE}/library**`);
  let n = 0;
  await docApi(page, { [`GET ${L}`]: (r) => (++n === 1 ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(list())) });
  await page.reload();
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Quy chế học vụ")).toBeVisible();
});

test("search: q ≥ 2 ký tự mới gửi; một ký tự bị bỏ qua; error keeps query", async ({ page, context }) => {
  let fail = false;
  const calls = await open(page, context, { [`GET ${L}`]: (r) => (fail ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(list())) });
  const q = page.getByLabel("Tìm tài liệu");
  await q.fill("a");
  await page.waitForTimeout(400);
  expect(calls.some((c) => c.path.includes("q=a"))).toBe(false);
  fail = true;
  await q.fill("quy chế");
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  await expect(q).toHaveValue("quy chế");
  expect(calls.some((c) => /q=quy\+ch%E1%BA%BF|q=quy%20ch/.test(c.path))).toBe(true);
});

for (const [w, h] of [[1440, 900], [375, 812]] as const) {
  test(`topic filter @${w} (DEF-R6-U9, US-P8-02 AC9): bộ lọc «Chủ đề» (cạnh Loại, Tuần) gửi category và lọc danh sách`, async ({ page, context }) => {
    await page.setViewportSize({ width: w, height: h });
    const calls = await open(page, context, {
      [`GET ${L}`]: (r) => {
        const cat = new URL(r.request().url()).searchParams.get("category");
        const all = [libItem({ id: "d1", title: "Quy chế học vụ", category: "Quy chế" }), libItem({ id: "d2", title: "Slide AES", category: "Mật mã" })];
        return r.fulfill(list(cat ? all.filter((d) => d.category === cat) : all));
      },
    });
    if (w < 720) await page.getByRole("button", { name: /^Bộ lọc/ }).click(); // di động: Loại / Chủ đề / Tuần nằm trong bảng «Bộ lọc» (M4)
    await page.getByLabel("Chủ đề").selectOption("Mật mã");
    await expect(page.getByText("Slide AES")).toBeVisible();
    await expect(page.getByText("Quy chế học vụ")).toHaveCount(0);
    expect(calls.some((c) => c.path.includes("category=M"))).toBe(true);
  });
}

// M4 (techlead-ui 2026-11-25): di động không tự focus ô tìm (bàn phím che nửa màn), Loại / Chủ đề / Tuần gom vào một nút «Bộ lọc», hàng tài liệu gọn.
test("M4 @375: ô tìm không tự focus, bộ lọc gom một nút, danh sách bắt đầu cao, mỗi hàng ≤ 110 px, cả hàng là liên kết", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await open(page, context, { [`GET ${L}`]: list([libItem({ category: "Quy chế" }), libItem({ id: "d2", title: "Slide TLS", file_kind: "PPTX", category: "Mật mã" })]) });
  await expect(page.getByLabel("Tìm tài liệu")).not.toBeFocused();
  const toggle = page.getByRole("button", { name: /^Bộ lọc/ });
  await expect(toggle).toBeVisible();
  for (const f of ["Loại", "Chủ đề", "Tuần"]) await expect(page.getByLabel(f, { exact: true }), f).toBeHidden();
  const rows = page.locator("[data-ep-action-list] > li");
  await expect(rows).toHaveCount(2);
  const first = (await rows.first().boundingBox())!;
  expect(first.y).toBeLessThanOrEqual(260);
  for (const h of await rows.evaluateAll((els) => els.map((e) => e.getBoundingClientRect().height))) expect(h).toBeLessThanOrEqual(110);
  await expect(page.getByRole("link", { name: /Quy chế học vụ/ })).toHaveCount(1); // cả hàng là liên kết
  await toggle.click();
  for (const f of ["Loại", "Chủ đề", "Tuần"]) await expect(page.getByLabel(f)).toBeVisible();
});

// techlead-ui 2026-11-25 (65fe5fdb): 1440 — cột hành động thẳng hàng dù có hàng không có «⋯»; ô «Tuần» không rơi xuống hàng riêng.
test("layout @1440: nút «Xem» thẳng cột dù hàng không có «⋯»; Loại / Chủ đề / Tuần cùng hàng", async ({ page, context }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await open(page, context, { [`GET ${L}`]: list([libItem({ category: "Quy chế" }), libItem({ id: "d2", title: "Slide TLS", file_kind: "PPTX", can_ask_ai: false, category: "Mật mã" }), libItem({ id: "d3", title: "Đề cương", can_ask_ai: true })]) });
  const xs = await page.getByRole("link", { name: "Xem" }).evaluateAll((els) => els.map((e) => Math.round(e.getBoundingClientRect().right)));
  expect(xs).toHaveLength(3);
  expect(new Set(xs).size, `mép phải nút Xem ${xs}`).toBe(1);
  const ys = await Promise.all(["Loại", "Chủ đề", "Tuần"].map(async (f) => Math.round((await page.getByLabel(f, { exact: true }).boundingBox())!.y)));
  expect(new Set(ys).size, `y ${ys}`).toBe(1);
});

test("ask about document: tạo phiên có document_id rồi sang /chat?session=", async ({ page, context }) => {
  await context.route("**/api/v1/chat/sessions?**", (r) => r.fulfill(json({ items: [], next_cursor: null })));
  await context.route(`**/api/v1/chat/sessions/${SID}/messages**`, (r) => r.fulfill(json({ items: [], next_cursor: null })));
  const calls = await open(page, context, { "POST /chat/sessions": json({ id: SID, title: null, last_message_at: "2026-10-11T02:00:00Z", document_id: DID }, 201) });
  await page.getByRole("button", { name: /Thêm với Quy chế học vụ/ }).click();
  await page.getByRole("menuitem", { name: "Hỏi AI về tài liệu" }).click();
  await expect(page).toHaveURL(new RegExp(`/chat\\?session=${SID}`));
  expect(JSON.parse(calls.find((c) => c.method === "POST" && c.path === "/chat/sessions")!.body)).toEqual({ course_id: COURSE, document_id: DID });
});

test("detail: PDF xem trước trong trang + Hỏi AI + Tải xuống; DOCX chỉ Tải xuống", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await docApi(page, { [`GET ${L}/${DID}`]: json(libDetail()), [`GET ${L}/${DID}/download`]: json({ url: "http://localhost:9/f.pdf" }) });
  await page.goto(`/library/${DID}`);
  await expect(page.locator("[data-part=pdf-preview]")).toHaveAttribute("src", "http://localhost:9/preview.pdf");
  await expect(page.getByRole("button", { name: "Hỏi AI về tài liệu" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Tải xuống" })).toBeVisible();
  await page.unroute(`**/api/v1/courses/${COURSE}/library**`);
  await docApi(page, { [`GET ${L}/${DID}`]: json(libDetail({ file_kind: "DOCX", preview_url: null, can_ask_ai: false })) });
  await page.reload();
  await expect(page.locator("[data-part=pdf-preview]")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Hỏi AI về tài liệu" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Tải xuống" })).toBeVisible();
});

test("detail: không tìm thấy → câu báo; tệp mất → Tệp này không còn nữa.", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await docApi(page, { [`GET ${L}/${DID}`]: json(libDetail()), [`GET ${L}/${DID}/download`]: err(404, "FILE_GONE"), [`GET ${L}/00000000-0000-7000-8000-0000000000ff`]: err(404, "NOT_FOUND") });
  await page.goto(`/library/${DID}`);
  await page.getByRole("button", { name: "Tải xuống" }).click();
  await expect(page.getByText("Tệp này không còn nữa.")).toBeVisible();
  await page.goto("/library/00000000-0000-7000-8000-0000000000ff");
  await expect(page.getByText("Không tìm thấy tài liệu")).toBeVisible();
});

test("375: danh sách không tràn ngang, vùng chạm ≥ 44 px", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await open(page, context);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0);
  const small = await page.locator("main a, main button, main input, main select").evaluateAll((els) => els.filter((e) => { const b = e.getBoundingClientRect(); return b.width > 0 && b.height > 0 && b.height < 44; }).map((e) => e.outerHTML.slice(0, 80)));
  expect(small).toEqual([]);
});
