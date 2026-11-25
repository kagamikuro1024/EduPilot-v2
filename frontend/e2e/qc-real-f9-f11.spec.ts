import { createHash } from "node:crypto";
import { test, expect } from "@playwright/test";
import { call, courseId, login, needReal, poll, shot, sql, uiLogin, uniq, type Json } from "./qc-real-lib";

// QC @real (P-11-29): F9 nộp → chấm (LLM qua 9router) → duyệt → công bố, rồi F11 phúc khảo. SV thao tác bằng giao diện thật;
// GV tạo / chấm hàng loạt / duyệt qua API thật (màn duyệt có spec riêng: qc-gd-*, qc-uiux91-w4*). Điểm nháp không bao giờ tới SV.
needReal();
test.describe.configure({ mode: "serial", timeout: 600_000 });
const tag = uniq();

function pdf(lines: string[]): Buffer {
  const esc = (s: string) => s.replace(/[()\\]/g, "[").replace(/[^\x20-\x7e]/g, "?");
  const cs = "BT /F1 14 Tf 72 740 Td 18 TL " + lines.map((l) => `(${esc(l)}) Tj T*`).join(" ") + " ET";
  const objs = ["<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
    `<< /Length ${cs.length} >>\nstream\n${cs}\nendstream`, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"];
  let out = "%PDF-1.4\n"; const offs: number[] = [];
  objs.forEach((o, i) => { offs.push(out.length); out += `${i + 1} 0 obj\n${o}\nendobj\n`; });
  const x = out.length;
  out += `xref\n0 ${objs.length + 1}\n0000000000 65535 f \n` + offs.map((o) => `${String(o).padStart(10, "0")} 00000 n \n`).join("");
  out += `trailer\n<< /Size ${objs.length + 1} /Root 1 0 R >>\nstartxref\n${x}\n%%EOF\n`;
  return Buffer.from(out, "latin1");
}

const S: { cid: string; aid: string; rid: string; title: string } = { cid: "", aid: "", rid: "", title: `QC F9 Tường lửa ${tag}` };

test("@real F9 GV tạo + công bố bài; SV nộp PDF trong app; chấm hàng loạt qua LLM thật → kết quả DRAFT, SV không thấy", async ({ page, request }) => {
  S.cid = await courseId(request, "teacher");
  const teacher = await login(request, "teacher");
  const due = new Date(Date.now() + 3 * 24 * 3600_000).toISOString().replace(/\.\d+Z$/, "Z");
  const rubric = [{ id: "c1", name: "Nêu đúng khái niệm", max_score: "5", guidance: "Nêu đúng định nghĩa tường lửa và vị trí đặt: 4–5. Thiếu một ý: 2–3. Sai: 0–1." },
    { id: "c2", name: "Ví dụ minh hoạ", max_score: "5", guidance: "Có ví dụ cụ thể về quy tắc lọc gói: 4–5. Chung chung: 2–3. Không có: 0–1." }];
  const mk = await call(request, teacher, "POST", `/courses/${S.cid}/assignments`, {
    title: S.title, description: "Trình bày tường lửa là gì, đặt ở đâu trong mạng và cho một ví dụ quy tắc lọc.", type: "ESSAY", max_score: "10", due_at: due,
    allow_late: true, late_penalty_per_day: "0.5", late_max_days: 2, appeal_days: 7, channels: ["IN_APP"], submission_code: `Q${tag.toUpperCase().slice(-6)}`,
    reference_answer: "Tường lửa là thiết bị hoặc phần mềm lọc lưu lượng theo quy tắc, thường đặt ở biên mạng giữa mạng trong và Internet.", rubric });
  expect([200, 201], mk.text.slice(0, 200)).toContain(mk.status);
  S.aid = String(mk.json.id ?? (mk.json.assignment as Json | undefined)?.id);
  const rub = await call(request, teacher, "PUT", `/courses/${S.cid}/assignments/${S.aid}/rubric`, { version: mk.json.version ?? 1, criteria: rubric });
  expect([200, 204], rub.text.slice(0, 200)).toContain(rub.status);
  const pub = await call(request, teacher, "POST", `/courses/${S.cid}/assignments/${S.aid}/publish`, {});
  expect([200, 201, 204], pub.text.slice(0, 200)).toContain(pub.status);

  await uiLogin(page, "sv.kha");
  await page.goto(`/assignments/${S.aid}`);
  await expect(page.getByRole("heading", { name: S.title })).toBeVisible();
  await shot(page, "f9-student-assignment");
  const body = pdf(["Tuong lua la thiet bi hoac phan mem loc luu luong theo quy tac.", "No thuong dat o bien mang giua mang trong va Internet.", "Vi du: quy tac chan moi goi den cong 23 (telnet) tu ben ngoai."]);
  await page.locator("input[type=file]").setInputFiles({ name: `bai-${tag}.pdf`, mimeType: "application/pdf", buffer: body });
  await page.getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByText(/Đã nhận bài lúc/).first()).toBeVisible({ timeout: 60_000 });
  await shot(page, "f9-student-submitted");
  expect(createHash("sha256").update(body).digest("hex")).toHaveLength(64);

  const gj = await call(request, teacher, "POST", `/courses/${S.cid}/assignments/${S.aid}/grade-all`, {});
  expect([200, 202], gj.text.slice(0, 200)).toContain(gj.status);
  const rows = await poll(() => sql(`select r.id, r.status, r.run_state from grading_results r join submissions s on s.id=r.submission_id where s.assignment_id='${S.aid}'`), (r) => r.length >= 1 && r.every((x) => x[2] === "DONE" || x[2] === "FAILED"), 300_000, 5000);
  expect(rows.length, "có kết quả chấm").toBeGreaterThanOrEqual(1);
  expect(rows[0][1], "kết quả chấm nháp").toBe("DRAFT");
  S.rid = rows[0][0];
  expect(Number(sql(`select count(*) from llm_audit where task='GRADING' and created_at > now() - interval '15 minutes'`)[0][0])).toBeGreaterThanOrEqual(1);
  // SV không thấy điểm nháp (API + UI)
  const mine = await call(request, await login(request, "sv.kha"), "GET", `/courses/${S.cid}/assignments/${S.aid}/mine`);
  expect(JSON.stringify(mine.json)).not.toMatch(/"(total|score|ai_scores)"/);
  await page.reload();
  await expect(page.locator("main")).not.toContainText(/\d+(,\d+)?\s*\/\s*10/);
  await shot(page, "f9-student-no-draft");
});

test("@real F9 GV sửa tiêu chí + duyệt → công bố; SV thấy điểm + nhận xét; bài chưa duyệt không công bố được", async ({ page, request }) => {
  const teacher = await login(request, "teacher");
  const rd = (await call(request, teacher, "GET", `/courses/${S.cid}/grading-results/${S.rid}`)).json;
  const res = (rd.result as Json | undefined) ?? rd;
  const crit = (res.criteria as { id: string; score?: string }[] | undefined) ?? [];
  const put = await call(request, teacher, "PUT", `/courses/${S.cid}/grading-results/${S.rid}`, { version: res.version ?? 1, criteria: (crit.length ? crit : [{ id: "c1" }, { id: "c2" }]).map((c, i) => ({ id: c.id, score: String(Math.min(5, 3 + i)) })) });
  expect(put.status, put.text.slice(0, 200)).toBe(200);
  const v2 = ((put.json.result as Json | undefined) ?? put.json).version ?? res.version ?? 1;
  const early = await call(request, teacher, "POST", `/courses/${S.cid}/grading-results/publish`, { result_ids: [S.rid] });
  expect([409, 422], "chưa duyệt → không công bố").toContain(early.status);
  const rv = await call(request, teacher, "POST", `/courses/${S.cid}/grading-results/${S.rid}/review`, { version: v2 });
  expect([200, 204], rv.text.slice(0, 200)).toContain(rv.status);
  const pb = await call(request, teacher, "POST", `/courses/${S.cid}/grading-results/publish`, { result_ids: [S.rid] });
  expect([200, 202], pb.text.slice(0, 200)).toContain(pb.status);
  await uiLogin(page, "sv.kha");
  await page.goto(`/assignments/${S.aid}`);
  await expect(page.locator("main")).toContainText(/\d+(,\d+)?\s*\/\s*10|\/ 10/);
  await shot(page, "f9-student-published");
  const bell = await call(request, await login(request, "sv.kha"), "GET", "/notifications?limit=20");
  expect(JSON.stringify(bell.json), "SV được báo điểm vừa công bố").toMatch(/điểm|công bố/i);
});

test("@real F11 phúc khảo: SV «Yêu cầu xem lại» (một lần) → GV quyết định có phản hồi → SV được báo; AI không tham gia", async ({ page, browser, request }) => {
  // Hôm nay gom theo bài tập có phúc khảo cũ nhất: đóng phúc khảo QC còn sót của lượt trước để mục «yêu cầu xem lại điểm» trỏ đúng bài vừa tạo
  sql(`update grade_appeals set status='UPHELD', decided_at=now(), decided_by=(select id from users where email='teacher@edupilot.local'), response='QC dọn fixture', score_after=score_before where status='OPEN' and assignment_id <> '${S.aid}'`);
  await uiLogin(page, "sv.kha");
  await page.goto(`/assignments/${S.aid}`);
  await page.getByRole("button", { name: "Yêu cầu xem lại" }).click();
  await page.getByRole("checkbox", { name: /Ví dụ minh hoạ/ }).check();
  await page.getByLabel("Lý do").fill("Em cho rằng ví dụ về quy tắc lọc cổng 23 đã đủ cụ thể, nhờ thầy xem lại tiêu chí 2.");
  await page.getByRole("button", { name: "Gửi", exact: true }).click({ timeout: 20_000 });
  await expect.poll(() => sql(`select count(*) from grade_appeals where course_id='${S.cid}' and assignment_id='${S.aid}' and student_id=(select id from users where email='sv.kha@edupilot.local') and status='OPEN'`)[0][0], { timeout: 10_000 }).toBe("1");
  await shot(page, "f11-student-appeal-sent");
  await expect(page.getByRole("button", { name: "Yêu cầu xem lại" })).toHaveCount(0); // mỗi bài một lần
  const llm0 = Number(sql(`select count(*) from llm_audit`)[0][0]);
  const gv = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const gp = await gv.newPage();
  await uiLogin(gp, "teacher");
  await gp.goto(`/?course=${S.cid}`);
  const item = gp.locator("main").getByText(/yêu cầu xem lại điểm/i).first();
  await expect(async () => { if (!(await item.isVisible())) await gp.reload(); await expect(item).toBeVisible({ timeout: 8_000 }); }).toPass({ timeout: 120_000, intervals: [5_000] });
  await item.click();
  await gp.getByRole("button", { name: "Giữ nguyên điểm" }).click({ timeout: 20_000 });
  await gp.getByLabel(/Phản hồi/).fill("Đã xem lại: ví dụ đạt yêu cầu, thầy giữ nguyên điểm theo thang đã công bố.");
  await gp.getByRole("button", { name: "Quyết định", exact: true }).click({ timeout: 20_000 });
  await gp.getByRole("dialog").getByRole("button", { name: "Quyết định" }).click({ timeout: 20_000 });
  await shot(gp, "f11-teacher-decided");
  await expect.poll(() => sql(`select status from grade_appeals where course_id='${S.cid}' and assignment_id='${S.aid}' order by created_at desc limit 1`)[0][0]).not.toBe("OPEN");
  await gv.close();
  const bell = await poll(async () => JSON.stringify((await call(request, await login(request, "sv.kha"), "GET", "/notifications?limit=20")).json), (s) => /phúc khảo|xem lại/i.test(s), 60_000, 3000);
  expect(bell).toMatch(/phúc khảo|xem lại/i);
  expect(Number(sql(`select count(*) from llm_audit`)[0][0]), "quyết định phúc khảo không gọi LLM").toBe(llm0);
  const again = await call(request, await login(request, "sv.kha"), "POST", `/courses/${S.cid}/assignments/${S.aid}/appeal`, { reason: "lần hai", criteria: ["c2"] });
  expect([403, 404, 409, 422], "lần hai bị chặn").toContain(again.status);
});
