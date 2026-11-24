import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { call, courseId, login, needReal, uiLogin, uniq } from "./qc-real-lib";

// TC-10-09 (DEF-R10S-13 / P-11-111, US-P9-10 AC3): API THẬT, không route.fulfill. GV giao bài thi PE → SV vào lượt → chat riêng bị khoá VÀ luyện đề vẫn bắt đầu được →
// lượt kết thúc (nộp / hết giờ) → chat mở lại ở chính màn /chat. Spec mock cũ (`quiz-integrity.spec.ts`) giữ nguyên.
// Chạy: xem playwright.real.config.ts. Dùng seed chuẩn: GV `teacher`, lớp 761987, mỗi ca MỘT SV riêng không có lượt nào ở seed (khoá chat theo SV nên các ca chạy song song không đạp nhau):
// sv21 nộp / desktop, sv22 nộp / mobile, sv23 hết giờ.
const TEACHER = "teacher";
const STUDENT_OF = { submitDesktop: "sv21", submitMobile: "sv22", expire: "sv23" } as const;
const LEAD_S = Number(process.env.E2E_EXAM_LEAD_S ?? 14); // = seed.mjs
const DURATION_MIN = Number(process.env.E2E_EXAM_MIN ?? 2); // = seed.mjs

test.beforeEach(() => needReal());

const composer = (page: Page) => page.getByRole("textbox", { name: "Câu hỏi của bạn" });

/** GV tạo + lên lịch bài thi 2 câu trắc nghiệm; trả id bài. */
async function giveExam(request: APIRequestContext, course: string): Promise<string> {
  const t = await login(request, TEACHER);
  const qs = await call(request, t, "GET", `/courses/${course}/questions?review_status=APPROVED&type=MCQ_SINGLE&topic=${encodeURIComponent("Mật mã đối xứng")}&limit=2`);
  const ids = ((qs.json.items ?? []) as { id: string }[]).map((q) => q.id);
  expect(ids, `ngân hàng seed thiếu câu MCQ_SINGLE «Mật mã đối xứng»: ${qs.text.slice(0, 120)}`).toHaveLength(2);
  const now = Date.now();
  const created = await call(request, t, "POST", `/courses/${course}/exams`, {
    title: `E2E khoá chat ${uniq()}`, instructions: "", duration_minutes: DURATION_MIN, multi_scoring: "PARTIAL",
    opens_at: new Date(now + LEAD_S * 1000).toISOString(), closes_at: new Date(now + (LEAD_S + DURATION_MIN * 60 + 120) * 1000).toISOString(),
  });
  expect(created.status, created.text.slice(0, 200)).toBe(201);
  const exam = created.json as { id: string; version: number };
  const put = await call(request, t, "PUT", `/courses/${course}/exams/${exam.id}/items`, { items: ids.map((id) => ({ question_id: id, points: "1" })), version: exam.version });
  expect(put.status, put.text.slice(0, 200)).toBe(200);
  const sched = await call(request, t, "POST", `/courses/${course}/exams/${exam.id}/schedule`, {});
  expect(sched.status, sched.text.slice(0, 200)).toBe(200);
  return exam.id;
}

/** SV vào lượt (thử lại tới giờ mở); trả id lượt + tab. */
async function startAttempt(request: APIRequestContext, student: string, course: string, exam: string): Promise<{ id: string; tab: string }> {
  const t = await login(request, student);
  const tab = crypto.randomUUID();
  await expect.poll(async () => (await call(request, t, "POST", `/courses/${course}/exams/${exam}/attempts`, undefined, { "X-Exam-Tab": tab })).status, { timeout: (LEAD_S + 30) * 1000, intervals: [2000] }).toBeLessThan(300);
  const r = await call(request, t, "POST", `/courses/${course}/exams/${exam}/attempts`, undefined, { "X-Exam-Tab": tab }); // cùng SV: trả lại lượt đang làm
  const id = ((r.json.attempt ?? {}) as { id?: string }).id;
  expect(id, r.text.slice(0, 200)).toBeTruthy();
  return { id: id!, tab };
}

async function lockedNow(request: APIRequestContext, student: string): Promise<boolean> {
  const r = await call(request, await login(request, student), "GET", "/me/exam-lock");
  return r.json.locked === true;
}

async function chatLockedAndPracticeOpen(page: Page) {
  await page.goto("/chat");
  await expect(composer(page)).toBeDisabled();
  await page.goto("/practice");
  await page.getByRole("tab", { name: /Theo chủ đề/ }).click();
  await page.getByRole("listitem").filter({ hasText: "Mật mã khối" }).getByRole("button").click(); // luyện đề KHÔNG bị khoá bởi bài thi
  await page.waitForURL(/\/practice\/[0-9a-f-]{36}/);
  await expect(page.getByRole("heading", { name: "Mật mã khối", level: 1 })).toBeVisible();
}

test("@real nộp bài: chat khoá khi đang trong lượt, luyện đề vẫn bắt đầu được, nộp xong chat mở lại", async ({ page, request }, info) => {
  const STUDENT = info.project.name === "mobile" ? STUDENT_OF.submitMobile : STUDENT_OF.submitDesktop;
  const course = await courseId(request, STUDENT);
  const exam = await giveExam(request, course);
  const { id, tab } = await startAttempt(request, STUDENT, course, exam);
  expect(await lockedNow(request, STUDENT)).toBe(true);

  await uiLogin(page, STUDENT);
  await chatLockedAndPracticeOpen(page);

  const sub = await call(request, await login(request, STUDENT), "POST", `/courses/${course}/exams/${exam}/attempts/${id}/submit`, undefined, { "X-Exam-Tab": tab });
  expect(sub.status, sub.text.slice(0, 200)).toBeLessThan(300);
  await page.goto("/chat");
  await expect(composer(page)).toBeEnabled();
});

test("@real hết giờ: tick tự nộp lượt, chat mở lại ở chính màn /chat", async ({ page, request }, info) => {
  test.skip(info.project.name !== "desktop", "đường hết giờ chạy một lần (≈ 3 phút)");
  const STUDENT = STUDENT_OF.expire;
  const course = await courseId(request, STUDENT);
  const exam = await giveExam(request, course);
  await startAttempt(request, STUDENT, course, exam);
  await uiLogin(page, STUDENT);
  await page.goto("/chat");
  await expect(composer(page)).toBeDisabled();

  await expect.poll(() => lockedNow(request, STUDENT), { timeout: (DURATION_MIN * 60 + 90) * 1000, intervals: [5000] }).toBe(false); // hạn + EXAM_GRACE_SECONDS + một nhịp tick
  await page.reload();
  await expect(composer(page)).toBeEnabled();
});
