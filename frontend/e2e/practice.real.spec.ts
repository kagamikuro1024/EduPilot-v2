import { readFileSync } from "node:fs";
import { join } from "node:path";
import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";
import { QUESTIONS as PE_QUESTIONS } from "../../scripts/exam-seed-data.mjs";
import { APPROVED as PRACTICE_QUESTIONS } from "../../scripts/practice-seed-data.mjs";
import { call, courseId, login, needReal, uiLogin } from "./qc-real-lib";

// TC-12-10 (DEF-R10S-13 / P-11-111, US-P9-12 AC3): thi thử 20 câu trên API THẬT, không route.fulfill. Tải lại ở câu 8 giữ vị trí + lựa chọn; mất mạng ở câu 12–14 không mất câu trả lời;
// nộp → «đúng k/20» khớp ĐÁP ÁN TÍNH TAY: khoá đáp án lấy từ DỮ LIỆU SEED (scripts/*-seed-data.mjs), không đọc từ API. Spec mock cũ (`practice.spec.ts`) giữ nguyên.
// Chạy: xem playwright.real.config.ts. Dùng seed chuẩn: SV `sv20` (không có lượt luyện đề ở seed), lớp 761987.
const STUDENT = "sv20";
const N = 20;

interface Key { type: string; options?: string[]; correct?: number[]; value?: boolean; accepted?: string[] }
// Ba nguồn của ngân hàng seed: 80 câu luyện đề, 20 câu PE, và các câu GENERATED (bản ghi `practice-generate-seed` của nhà cung cấp giả: seed/demo/fake_llm.json).
interface Rec { name: string; response: { questions?: { type: string; stem: string; options: { body: string; correct: boolean }[] }[] } }
const GENERATED: (Key & { stem: string })[] = (JSON.parse(readFileSync(join(__dirname, "../../seed/demo/fake_llm.json"), "utf8")).recordings as Rec[])
  .filter((r) => r.name === "practice-generate-seed").flatMap((r) => r.response.questions ?? [])
  .map((q) => ({ type: q.type, stem: q.stem, options: q.options.map((o) => o.body), correct: q.options.flatMap((o, i) => (o.correct ? [i] : [])) }));
const KEYS = new Map<string, Key>([...PRACTICE_QUESTIONS, ...PE_QUESTIONS, ...GENERATED].map((q: Key & { stem: string }) => [q.stem, q]));

// Kế hoạch tính tay: câu 4, 8, 12, 16 trả lời SAI có chủ đích, câu 20 bỏ trống, 15 câu còn lại đúng ⇒ 15/20 (quy đổi 7,5/10).
const wrongAt = (n: number) => n % 4 === 0 && n < N;
const blankAt = (n: number) => n === N;
const EXPECTED = Array.from({ length: N }, (_, i) => i + 1).filter((n) => !wrongAt(n) && !blankAt(n)).length;

test.beforeEach(() => needReal());

const mock = (page: Page) => page.locator("[data-part=practice-mock]");

/** Hiện câu `n` (1-based) đang mở trên màn: trả khoá đáp án theo đề trên màn. */
async function keyOnScreen(page: Page): Promise<Key> {
  const text = (await mock(page).getByRole("region", { name: "Câu hỏi" }).innerText()).replace(/\s+/g, " ");
  const hit = [...KEYS.entries()].filter(([stem]) => text.includes(stem.replace(/\s+/g, " "))).sort((a, b) => b[0].length - a[0].length)[0];
  expect(hit, `câu trên màn không có trong dữ liệu seed: ${text.slice(0, 160)}`).toBeTruthy();
  return hit![1];
}

const option = (page: Page, body: string): Locator => mock(page).getByRole("list", { name: "Các đáp án" }).getByRole("button", { name: body, exact: true });

/** Trả lời câu đang mở: `right` đúng theo khoá tính tay, ngược lại sai có chủ đích. Trả hàm kiểm lựa chọn còn nguyên sau tải lại. */
async function answer(page: Page, right: boolean): Promise<() => Promise<void>> {
  const k = await keyOnScreen(page);
  if (k.type === "TRUE_FALSE") {
    const v = right ? k.value! : !k.value;
    const b = mock(page).getByRole("list", { name: "Đúng hoặc sai" }).getByRole("button", { name: v ? "Đúng" : "Sai", exact: true });
    await b.click();
    return () => expect(b).toHaveAttribute("aria-pressed", "true");
  }
  if (k.type === "SHORT") {
    const text = right ? k.accepted![0] : "không đúng";
    const input = mock(page).getByLabel("Câu trả lời ngắn");
    await input.fill(text);
    return () => expect(input).toHaveValue(text);
  }
  const all = k.options!.map((_, i) => i);
  const picks = right ? k.correct! : [all.find((i) => !k.correct!.includes(i))!]; // sai = chỉ chọn MỘT phương án không thuộc đáp án đúng
  for (const i of picks) await option(page, k.options![i]).click();
  return async () => { for (const i of picks) await expect(option(page, k.options![i])).toHaveAttribute("aria-pressed", "true"); };
}

const saved = (page: Page) => expect(mock(page).getByRole("status").first()).toContainText("Đã lưu lúc", { timeout: 20_000 });
const header = (page: Page, n: number) => expect(mock(page).getByText(`Câu ${n}/${N}`, { exact: true })).toBeVisible();

/** Mỗi lần chạy từ trạng thái sạch: nộp nốt lượt thi thử đang dở (nếu lần trước hỏng giữa chừng), vì máy chủ trả lại lượt dở thay vì tạo lượt mới. */
async function closeOpenMocks(request: APIRequestContext, course: string) {
  const t = await login(request, STUDENT);
  const ov = await call(request, t, "GET", `/courses/${course}/practice/overview`);
  for (const a of (ov.json.in_progress ?? []) as { id: string; mode: string }[]) {
    if (a.mode === "MOCK") await call(request, t, "POST", `/courses/${course}/practice/attempts/${a.id}/submit`, undefined, { "X-Practice-Tab": crypto.randomUUID() });
  }
}

test("@real thi thử 20 câu: tải lại ở câu 8, mất mạng ở câu 12–14, nộp → kết quả khớp tính tay", async ({ page, context, request }) => {
  const course = await courseId(request, STUDENT);
  await closeOpenMocks(request, course);
  await uiLogin(page, STUDENT);

  await page.goto("/practice");
  await page.getByRole("tab", { name: /Thi thử/ }).click();
  await page.getByLabel("Số câu").fill(String(N));
  await page.getByLabel("Số phút").fill("30");
  await page.getByRole("button", { name: "Bắt đầu thi thử" }).click();
  await page.waitForURL(/\/practice\/[0-9a-f-]{36}/);
  const attemptId = page.url().split("/").pop()!;
  await expect(mock(page)).toBeVisible();
  await header(page, 1);

  const next = async (n: number) => { await mock(page).getByRole("button", { name: "Câu sau" }).click(); await header(page, n); };
  let check8: () => Promise<void> = async () => undefined;

  for (let n = 1; n <= 8; n++) {
    if (n > 1) await next(n);
    const again = await answer(page, !wrongAt(n));
    if (n === 8) check8 = again;
  }
  await saved(page);

  // tải lại ở câu 8: giữ vị trí + lựa chọn (lượt PATCH vị trí đã tới máy chủ khi bấm «Câu sau» sang câu 8)
  await page.reload();
  await expect(mock(page)).toBeVisible();
  await header(page, 8);
  await check8();

  for (let n = 9; n <= 11; n++) {
    await next(n);
    await answer(page, !wrongAt(n));
  }
  await saved(page);

  // mất mạng ở câu 12–14: lựa chọn giữ trên máy, báo chưa gửi được 3 câu, có mạng lại thì tự gửi, không mất câu nào
  await context.setOffline(true);
  for (let n = 12; n <= 14; n++) {
    await next(n);
    await answer(page, !wrongAt(n));
  }
  await expect(mock(page).getByText(/Chưa gửi được 3 câu/)).toBeVisible();
  await context.setOffline(false);
  await saved(page);
  const t = await login(request, STUDENT);
  const got = await call(request, t, "GET", `/courses/${course}/practice/attempts/${attemptId}`);
  const items = (got.json.items ?? []) as { answer: unknown }[];
  expect(items.slice(11, 14).map((i) => i.answer !== null), "máy chủ nhận đủ câu 12–14 ghi lúc ngoại tuyến").toEqual([true, true, true]);

  for (let n = 15; n <= N; n++) {
    await next(n);
    if (!blankAt(n)) await answer(page, !wrongAt(n));
  }
  await saved(page);

  await mock(page).getByRole("button", { name: "Nộp bài" }).first().click();
  await expect(page.getByRole("dialog")).toContainText(`Bạn đã trả lời ${N - 1}/${N} câu`);
  await page.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByRole("heading", { name: `Bạn đúng ${EXPECTED}/${N} câu`, level: 1 })).toBeVisible();
  await expect(page.getByText(`quy đổi ${(EXPECTED / 2).toFixed(1).replace(".", ",")}/10`)).toBeVisible();
});
