import { expect, test } from "@playwright/test";
import { A1, A2, studentApi, studentItem } from "./support/assignment-fixtures";
import { COURSE } from "./support/cal-fixtures";
import { asDemo } from "./support/session";

// P-11-14 / US-P7-03 AC11: `/assignments` của sinh viên — ba nhóm, một hành động mỗi hàng; mục điều hướng «Bài tập».
const A3 = "00000000-0000-7000-8000-0000000a0003";
const A4 = "00000000-0000-7000-8000-0000000a0004";
const items = () => [
  studentItem({ id: A2, title: "Bài tập 04 · Mã độc", due_at: "2026-12-01T07:00:00.000Z" }),
  studentItem({ id: A1, title: "Bài tập 03 · Phân tích nhật ký", due_at: "2026-11-20T07:00:00.000Z" }),
  studentItem({ id: A3, title: "Bài tập 02 · Mật mã", my: { state: "SUBMITTED", late_days: 1, submitted_at: "2026-10-23T01:10:00.000Z" } }),
  studentItem({ id: A4, title: "Bài tập 01 · Mạng", my: { state: "GRADED", late_days: 0, submitted_at: "2026-10-10T01:10:00.000Z" } }),
];

test("danh sách: ba nhóm theo trạng thái, hạn gần nhất trước, một hành động mỗi hàng", async ({ page, context }) => {
  await asDemo(context, "student");
  await studentApi(page, { items: items() });
  await page.goto("/assignments");
  await expect(page.getByRole("heading", { level: 1, name: "Bài tập" })).toBeVisible();
  await expect(page.getByRole("heading", { level: 2 })).toHaveText(["Cần nộp", "Đã nộp, chờ điểm", "Đã có điểm"]);
  const rows = page.locator("[data-part=assignment-row]");
  await expect(rows).toHaveCount(4);
  await expect(rows.nth(0)).toContainText("Bài tập 03"); // hạn 20/11 trước 01/12
  await expect(rows.nth(1)).toContainText("Bài tập 04");
  await expect(rows.nth(0).getByRole("link", { name: "Nộp bài" })).toHaveAttribute("href", `/assignments/${A1}?course=${COURSE}`);
  await expect(rows.nth(2)).toContainText("Nộp muộn 1 ngày");
  await expect(rows.nth(2).getByRole("link", { name: "Nộp lại" })).toBeVisible();
  await expect(rows.nth(3).getByRole("link", { name: "Xem" })).toBeVisible();
  for (const r of await rows.all()) await expect(r.getByRole("link", { name: /^(Nộp bài|Nộp lại|Xem)$/ })).toHaveCount(1);
  await expect(page.locator("main")).not.toContainText(/nháp|AI|cờ|rubric/i);
});

test("danh sách: nhóm trống không hiện; không có bài thì «Chưa có bài tập» và không nút", async ({ page, context }) => {
  await asDemo(context, "student");
  await studentApi(page, { items: [studentItem({ my: { state: "GRADED", late_days: 0, submitted_at: "2026-10-10T01:10:00.000Z" } })] });
  await page.goto("/assignments");
  await expect(page.getByRole("heading", { level: 2 })).toHaveText(["Đã có điểm"]);
  const empty = await context.newPage();
  await studentApi(empty, { items: [] });
  await empty.goto("/assignments");
  await expect(empty.getByText("Chưa có bài tập")).toBeVisible();
  await expect(empty.locator("main").getByRole("button")).toHaveCount(0);
  await expect(empty.locator("main").getByRole("link")).toHaveCount(0);
});

test("điều hướng: «Bài tập» đứng giữa «Hỏi đáp công khai» và «Luyện đề» (desktop); điện thoại: đầu «Thêm»", async ({ page, context }, info) => {
  await asDemo(context, "student");
  await studentApi(page, { items: items() });
  await page.goto("/assignments");
  if (info.project.name === "mobile") {
    await page.getByRole("button", { name: "Thêm" }).click();
    const links = page.getByRole("dialog").getByRole("link");
    await expect(links.first()).toHaveText(/Bài tập/);
    return;
  }
  const names = await page.getByRole("navigation", { name: /điều hướng|chính/i }).first().getByRole("link").allInnerTexts();
  const i = names.findIndex((n) => n.includes("Bài tập"));
  expect(i).toBeGreaterThan(0);
  expect(names[i - 1]).toContain("Hỏi đáp");
  expect(names[i + 1]).toContain("Luyện đề");
});

test("Staff mở /assignments: màn chặn, không gọi API bài tập", async ({ page, context }) => {
  await asDemo(context, "teacher");
  const calls: string[] = [];
  page.on("request", (r) => /\/assignments(\?|$)/.test(new URL(r.url()).pathname) && calls.push(r.url()));
  await page.goto("/assignments");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
  expect(calls).toEqual([]);
});
