import { writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, DID, chunk, doc, docApi, err, json, noContent, stats } from "./support/doc-fixtures";

// US-P8-02: /documents THẬT (gateway giả theo hợp đồng thật, support/doc-fixtures.ts).
const D = `/courses/${COURSE}/documents`;
const list = (items: unknown[] = [doc()]) => json({ items, next_cursor: null });
const pdf = (name = "bai-giang.pdf") => ({ name, mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.4 x") });

async function open(page: Page, context: BrowserContext, h: Parameters<typeof docApi>[1] = {}, role: "teacher" | "ta" = "teacher") {
  await asDemo(context, role);
  const calls = await docApi(page, { [`GET ${D}`]: list(), [`GET ${D}/stats`]: json(stats()), ...h });
  await page.goto("/documents");
  await page.locator("[data-part=dropzone]").waitFor();
  return calls;
}

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca 375 không thuộc story này (≥ 720 px)");
});

test("table: tên, loại, tuần, hai cờ, trạng thái, cập nhật; dải thống kê một dòng", async ({ page, context }) => {
  await open(page, context, { [`GET ${D}`]: list([doc(), doc({ id: "d2", title: "Slide TLS", status: "FAILED", error: "Không đọc được", use_for_rag: false }), doc({ id: "d3", title: "Từ lớp khác", shared_from: "761988" })]), [`GET ${D}/stats`]: json(stats({ total: 3, by_status: { READY: 1, FAILED: 1, PROCESSING: 1 } })) });
  const rows = page.locator("[data-part=doc-row]");
  await expect(rows).toHaveCount(3);
  await expect(page.locator("[data-part=doc-stats]")).toHaveText("3 tài liệu · 1 đang xử lý · 1 lỗi");
  await expect(rows.getByText("Lỗi — Không đọc được")).toBeVisible();
  await expect(rows.getByText("Chia sẻ từ 761988")).toBeVisible();
  await expect(rows.getByRole("switch", { name: /Dùng cho AI: Từ lớp khác/ })).toBeDisabled();
  await expect(page.locator('main [data-variant="primary"]:visible')).toHaveCount(1);
});

test("states: rỗng có Tải tài liệu lên; lỗi có Thử lại", async ({ page, context }) => {
  await open(page, context, { [`GET ${D}`]: list([]), [`GET ${D}/stats`]: json(stats({ total: 0, has_course_policy: false })) });
  await expect(page.getByText("Lớp chưa có tài liệu.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Tải tài liệu lên" })).toHaveCount(2);
  await page.unroute(`**/api/v1/courses/${COURSE}/documents**`);
  let n = 0;
  await docApi(page, { [`GET ${D}`]: (r) => (++n === 1 ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(list())), [`GET ${D}/stats`]: json(stats()) });
  await page.reload();
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.locator("[data-part=doc-row]")).toHaveCount(1);
});

test("answer key labels: đúng hai dòng, công tắc Hiện cho sinh viên khoá", async ({ page, context }) => {
  await open(page, context, { [`GET ${D}`]: list([doc({ id: "k1", title: "Đáp án tuần 1", type: "ANSWER_KEY", visible_to_students: false })]) });
  const row = page.locator("[data-part=doc-row]");
  await expect(row.getByText("Không hiển thị cho sinh viên")).toBeVisible();
  await expect(row.getByText("Không dùng cho AI của sinh viên")).toBeVisible();
  await expect(row.getByRole("switch", { name: /Hiện cho sinh viên/ })).toBeDisabled();
});

test("course policy reminder: một dòng khi thiếu quy chế, nút mở chọn tệp; có rồi thì biến mất", async ({ page, context }) => {
  await open(page, context, { [`GET ${D}/stats`]: json(stats({ has_course_policy: false })) });
  await expect(page.getByText("Lớp chưa có quy chế môn học.")).toHaveCount(1);
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Tải quy chế môn học" }).click();
  await chooser;
  await page.unroute(`**/api/v1/courses/${COURSE}/documents**`);
  await docApi(page, { [`GET ${D}`]: list(), [`GET ${D}/stats`]: json(stats()) });
  await page.reload();
  await page.locator("[data-part=dropzone]").waitFor();
  await expect(page.getByText("Lớp chưa có quy chế môn học.")).toHaveCount(0);
});

test("reject before upload: sai loại / quá 50 MB báo ngay, không gọi presign", async ({ page, context }) => {
  const calls = await open(page, context);
  const big = join(tmpdir(), "lon.pdf");
  writeFileSync(big, Buffer.alloc(50 * 1024 * 1024 + 1));
  const txt = join(tmpdir(), "ghi-chu.txt");
  writeFileSync(txt, "x");
  await page.locator("[data-part=file-input]").setInputFiles([txt, big]);
  await expect(page.getByText("ghi-chu.txt: chỉ nhận PDF, DOCX hoặc PPTX. Hãy chọn tệp khác.")).toBeVisible();
  await expect(page.getByText("lon.pdf: quá 50 MB. Hãy nén hoặc tách tệp.")).toBeVisible();
  expect(calls.some((c) => c.path.includes("/uploads/"))).toBe(false);
});

const UP = `/courses/${COURSE}/uploads`;
const presignOk = (c: { put: string }) => json({ upload_id: "00000000-0000-7000-8000-0000000000u1", method: "PUT", url: c.put, headers: { "Content-Type": "application/pdf" }, blob_key: "k", expires_in: 600 });

test("dropzone + progress: hàng Đang tải → Đang xử lý (job) → biến khi xong (job đã xong ngay ở lần đọc đầu: không phụ thuộc SSE / thăm dò 2 s); hàng mới ở bảng", async ({ page, context }) => {
  await page.route("http://blob.test/put", (r) => r.fulfill({ status: 200, headers: { "Access-Control-Allow-Origin": "*" }, body: "" }));
  let job = 0;
  let uploaded = false;
  const calls = await open(page, context, {
    [`GET ${D}`]: (r) => r.fulfill(list(uploaded ? [doc(), doc({ id: "n1", title: "bai-giang", status: "PROCESSING" })] : [doc()])),
    [`POST ${UP}/presign`]: presignOk({ put: "http://blob.test/put" }),
    [`POST ${UP}/complete`]: (r) => { uploaded = true; return r.fulfill(json({ job_id: "00000000-0000-7000-8000-0000000000j1", document: doc({ id: "n1", title: "bai-giang", status: "QUEUED" }) }, 202)); },
    "GET /jobs/00000000-0000-7000-8000-0000000000j1": (r) => r.fulfill(json({ id: "00000000-0000-7000-8000-0000000000j1", status: ++job < 1 ? "RUNNING" : "SUCCEEDED", progress: 100 })),
  });
  await page.route("**/api/v1/jobs/**", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204 }) : r.fallback()));
  await page.locator("[data-part=file-input]").setInputFiles(pdf());
  const row = page.locator("[data-part=upload-row]");
  await expect(row).toContainText("bai-giang.pdf");
  await expect(row).toContainText(/Đang tải|Đang xử lý/);
  await expect(row).toHaveCount(0, { timeout: 15_000 });
  const complete = calls.find((c) => c.path === `${UP}/complete`)!;
  expect(JSON.parse(complete.body)).toMatchObject({ title: "bai-giang", type: "LECTURE" });
  expect(complete.headers["idempotency-key"]).toBeTruthy();
});

test("offline upload retry: mất mạng giữa chừng → Lỗi — Tải lại giữ tên tệp, tải lại đúng tệp đó, một lần complete", async ({ page, context }) => {
  let offline = true;
  await page.route("http://blob.test/put", (r) => (offline ? r.abort("failed") : r.fulfill({ status: 200, headers: { "Access-Control-Allow-Origin": "*" }, body: "" })));
  const calls = await open(page, context, {
    [`POST ${UP}/presign`]: presignOk({ put: "http://blob.test/put" }),
    [`POST ${UP}/complete`]: json({ job_id: "00000000-0000-7000-8000-0000000000j1", document: doc({ id: "n1", status: "QUEUED" }) }, 202),
    "GET /jobs/00000000-0000-7000-8000-0000000000j1": json({ id: "j", status: "SUCCEEDED", progress: 100 }),
  });
  await page.locator("[data-part=file-input]").setInputFiles(pdf("bai-2.pdf"));
  const row = page.locator("[data-part=upload-row]");
  await expect(row).toContainText("Lỗi");
  await expect(row).toContainText("bai-2.pdf");
  offline = false;
  await row.getByRole("button", { name: "Tải lại" }).click();
  await expect(row).toHaveCount(0, { timeout: 15_000 });
  expect(calls.filter((c) => c.path === `${UP}/complete`)).toHaveLength(1);
});

test("keyboard: Enter trên vùng thả mở hộp chọn tệp; công tắc đổi bằng phím cách", async ({ page, context }) => {
  await open(page, context, { [`PATCH ${D}/${DID}`]: (r) => r.fulfill(json(doc({ version: 2, use_for_rag: false }))) });
  await page.locator("[data-part=dropzone]").focus();
  const chooser = page.waitForEvent("filechooser");
  await page.keyboard.press("Enter");
  await chooser;
  const sw = page.getByRole("switch", { name: /Dùng cho AI/ });
  await sw.focus();
  await page.keyboard.press("Space");
  await expect(sw).toHaveAttribute("aria-checked", "false");
});

test("patch: đổi cờ lạc quan + Đã đổi · Hoàn tác; version trong thân; patch rollback khi 500", async ({ page, context }) => {
  let n = 0;
  const calls = await open(page, context, {
    [`PATCH ${D}/${DID}`]: (r) => (++n === 2 ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json(doc({ version: 2, visible_to_students: false })))),
  });
  const vis = page.getByRole("switch", { name: /Hiện cho sinh viên/ });
  await vis.click();
  await expect(vis).toHaveAttribute("aria-checked", "false");
  await expect(page.getByText("Đã đổi cờ Hiện cho sinh viên")).toBeVisible();
  expect(JSON.parse(calls.find((c) => c.method === "PATCH")!.body)).toEqual({ visible_to_students: false, version: 1 });
  await vis.click(); // lần 2: máy chủ lỗi 500 → hoàn lại và báo
  await expect(page.getByText("Chưa lưu được. Thử lại.")).toBeVisible();
  await expect(vis).toHaveAttribute("aria-checked", "false");
});

test("chunks: Drawer liệt kê đoạn, sửa đoạn và lưu; nhúng lỗi báo Chưa lưu được", async ({ page, context }) => {
  let fail = true;
  const calls = await open(page, context, {
    [`GET ${D}/${DID}/chunks`]: json({ items: [chunk(0), chunk(1)], next_cursor: null }),
    [`PATCH ${D}/${DID}/chunks/${chunk(0).id}`]: (r) => (fail ? r.fulfill(err(503, "CHAT_UNAVAILABLE")) : r.fulfill(json(chunk(0, { text: "Chữ mới" })))),
  });
  await page.locator("[data-part=doc-row]").getByRole("button", { name: "Quy chế học vụ" }).first().click();
  await expect(page.getByText("Nội dung đoạn 2")).toBeVisible();
  await page.getByRole("button", { name: "Sửa đoạn" }).first().click();
  await page.getByLabel("Nội dung đoạn").fill("Chữ mới");
  await page.getByRole("button", { name: "Lưu đoạn" }).click();
  await expect(page.getByText("Chưa lưu được. Thử lại.")).toBeVisible();
  fail = false;
  await page.getByRole("button", { name: "Lưu đoạn" }).click();
  await expect(page.getByLabel("Nội dung đoạn")).toHaveCount(0);
  expect(calls.filter((c) => c.method === "PATCH")).toHaveLength(2);
});

test("delete: Giảng viên xoá có xác nhận nêu số cụ thể; TA không có mục Xoá", async ({ page, context }) => {
  const calls = await open(page, context, {
    [`GET ${D}/${DID}/impact`]: json({ chunks: 12, courses: 1 }),
    [`DELETE ${D}/${DID}`]: noContent,
  });
  await page.locator("[data-part=doc-row]").getByRole("button", { name: /Thao tác với Quy chế học vụ/ }).first().click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect(page.getByText("Xoá Quy chế học vụ?")).toBeVisible();
  await expect(page.getByText("12 đoạn và 1 lớp đang dùng tài liệu này sẽ mất nó.")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Xoá" }).click();
  await expect.poll(() => calls.some((c) => c.method === "DELETE")).toBe(true);
});

test("TA: không có mục Xoá", async ({ page, context }) => {
  await open(page, context, {}, "ta");
  await page.locator("[data-part=doc-row]").getByRole("button", { name: /Thao tác với Quy chế học vụ/ }).first().click();
  await expect(page.getByRole("menuitem", { name: "Lập chỉ mục lại" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Xoá" })).toHaveCount(0);
});
