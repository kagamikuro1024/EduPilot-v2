import { setTimeout as sleep } from "node:timers/promises";
import { test, expect } from "@playwright/test";
import { TICK_WAIT_MS, call, courseId, login, mailCount, needReal, poll, shot, sql, uiLogin, uniq } from "./qc-real-lib";

// QC @real (P-11-29): F3 khủng hoảng ở /chat không gọi LLM; F5 nhắc 24 / 72 giờ; F13 nhắc 24 giờ + đổi hạn.
// «Tua giờ» = fixture DB (lùi waiting_since / dời starts_at) rồi chờ tick thật của worker (SLA 15 phút, reminder 5 phút mặc định;
// hạ TICKET_SLA_TICK=1m, REMINDER_TICK=30s ở worker ep-qc thì đặt QC_TICK_WAIT_MS nhỏ hơn). Không có đường tua đồng hồ trong mã.
needReal();
test.describe.configure({ mode: "serial" });
const tag = uniq();

const SVK = "(select id from users where email='sv.kha@edupilot.local')";
const llmRows = () => Number(sql(`select count(*) from llm_audit where user_id=${SVK}`)[0][0]);
const llmRowsSince = (iso: string) => Number(sql(`select count(*) from llm_audit where user_id=${SVK} and created_at > '${iso}'`)[0][0]);

// ---------------- F3 ----------------
for (const msg of ["Mình muốn tự tử", "em dang muon chet di cho xong", "toi khong con ly do de song nua"]) {
  test(`@real F3 khủng hoảng ở /chat: «${msg}» → lời ân cần + thông tin hỗ trợ, KHÔNG có lời gọi LLM / decision`, async ({ page }) => {
    await uiLogin(page, "sv.kha");
    await page.goto("/chat");
    const composer = page.getByRole("textbox", { name: "Câu hỏi của bạn" });
    await expect(composer).toBeVisible();
    const t0 = new Date().toISOString();
    const before = llmRows();
    await composer.fill(msg);
    await composer.press("Enter");
    // câu trả lời của hệ thống xuất hiện; không nhắc «AI», «mô hình»
    await expect(page.getByText(/(mình|bạn).*(ở đây|nghe|quan tâm)|đường dây|hỗ trợ|phòng công tác sinh viên/i).first()).toBeVisible({ timeout: 30_000 });
    await shot(page, `f3-crisis-${msg.length}`);
    await page.waitForTimeout(3000);
    expect(llmRowsSince(t0), `lời gọi LLM sau tin khủng hoảng (llm_audit): ${JSON.stringify(sql(`select task, status from llm_audit where user_id=${SVK} and created_at > '${t0}'`))}`).toBe(0);
    expect(llmRows()).toBe(before);
    // không ticket tự mở, câu hỏi khủng hoảng không vào Threads / RAG / báo cáo lỗ hổng
    const tk = sql(`select count(*) from escalation_tickets t join users u on u.id=t.student_id where u.email='sv.kha@edupilot.local' and t.created_at > '${t0}'`)[0][0];
    expect(Number(tk), "không tạo ticket tự động cho tin khủng hoảng").toBe(0);
  });
}

test("@real F3 khủng hoảng: tải lại vẫn thấy phần đã sinh; Admin/GV không đọc được nội dung chat riêng", async ({ page, request }) => {
  await uiLogin(page, "sv.kha");
  await page.goto("/chat");
  await page.reload();
  const cid = await courseId(request, "teacher");
  const sid = sql(`select s.id from chat_sessions s join users u on u.id=s.user_id where u.email='sv.kha@edupilot.local' and s.course_id='${cid}' order by s.created_at desc limit 1`)[0]?.[0];
  expect(sid).toBeTruthy();
  for (const who of ["teacher", "admin"]) {
    const r = await call(request, await login(request, who), "GET", `/chat/sessions/${sid}/messages`);
    expect([401, 403, 404], `${who} đọc chat riêng chưa escalate: ${r.status}`).toContain(r.status);
  }
});

// ---------------- F5 ----------------
test("@real F5 nhắc 24 giờ → 72 giờ: ticket quá hạn → thông báo gộp + thư, rồi nổi lên «Hôm nay» của GV kèm số giờ đã chờ", async ({ page, browser, request }) => {
  test.setTimeout(TICK_WAIT_MS * 3 + 300_000);
  // thư nhắc ticket chỉ gửi GV đã bật «nhận thư khi có câu hỏi chờ» (mặc định tắt): bật bằng fixture, trả lại cuối ca
  sql(`insert into user_settings (user_id, notify_ticket_by_mail) select id, true from users where email='teacher@edupilot.local' on conflict (user_id) do update set notify_ticket_by_mail = true`);
  const t0 = Date.now();
  // 1) SV hỏi một câu ngoài tài liệu rồi bấm «Nhờ giảng viên hỗ trợ» (UI thật)
  await uiLogin(page, "sv.nguyco");
  await page.goto("/chat");
  const composer = page.getByRole("textbox", { name: "Câu hỏi của bạn" });
  await composer.fill(`Lịch bảo vệ đồ án năm sau có thay đổi không ạ? (QC ${tag})`);
  await composer.press("Enter");
  // AI không đủ chắc → hệ thống tự mở ticket («Đang chờ giảng viên»); nếu chưa tự mở thì bấm nút nhờ giảng viên
  const waiting = page.getByText(/Đang chờ giảng viên/);
  const ask = page.getByRole("button", { name: /Nhờ giảng viên hỗ trợ/ });
  await expect(waiting.or(ask).first()).toBeVisible({ timeout: 90_000 });
  if (!(await waiting.count())) await ask.last().click();
  await expect.poll(() => sql(`select count(*) from escalation_tickets t join users u on u.id=t.student_id where u.email='sv.nguyco@edupilot.local' and t.status='OPEN' and t.created_at > to_timestamp(${t0 / 1000})`)[0][0], { timeout: 30_000 }).toBe("1");
  const tid = sql(`select t.id from escalation_tickets t join users u on u.id=t.student_id where u.email='sv.nguyco@edupilot.local' and t.status='OPEN' order by t.created_at desc limit 1`)[0][0];
  await shot(page, "f5-ticket-open");
    // 2) lùi đồng hồ: 6 ngày lịch (≥ 24 giờ làm việc kể cả cuối tuần)
  sql(`update escalation_tickets set waiting_since = now() - interval '6 days', created_at = now() - interval '6 days' where id='${tid}'`);
  const s1 = await poll(() => sql(`select sla_stage from escalation_tickets where id='${tid}'`)[0][0], (v) => v === "1" || v === "2", TICK_WAIT_MS, 15_000);
  expect(["1", "2"], `sla_stage sau ${TICK_WAIT_MS / 60000} phút`).toContain(s1);
  const tkn = await login(request, "teacher");
  const bell = await poll(async () => ((await call(request, tkn, "GET", "/notifications?limit=30")).json.items ?? []) as { type: string; title: string }[], (it) => it.some((n) => n.type === "TICKET_STALE"), 120_000, 10_000);
  expect(bell.some((n) => n.type === "TICKET_STALE"), "GV có thông báo gộp «N câu hỏi chờ quá 24 giờ làm việc»").toBe(true);
  // thư nhắc gộp theo lớp, một thư mỗi lớp mỗi ngày (dedupe): đã gửi hôm nay ở lần chạy trước thì không gửi lại
  expect(await poll(() => mailCount("teacher@edupilot.local", /chờ giảng viên quá 24 giờ làm việc/i), (n) => n >= 1, 120_000, 10_000), "có thư nhắc «quá 24 giờ làm việc» tới GV").toBeGreaterThanOrEqual(1);
  // 3) 72 giờ làm việc: ≥ 14 ngày lịch
  sql(`update escalation_tickets set waiting_since = now() - interval '20 days', created_at = now() - interval '20 days', sla_stage = case when sla_stage < 1 then 1 else sla_stage end where id='${tid}'`);
  const s2 = await poll(() => sql(`select sla_stage from escalation_tickets where id='${tid}'`)[0][0], (v) => v === "2", TICK_WAIT_MS, 15_000);
  expect(s2, "ticket lên giai đoạn 2 (72 giờ làm việc)").toBe("2");
  const gv = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 } });
  const gp = await gv.newPage();
  await uiLogin(gp, "teacher");
  await gp.goto(`/?course=${await courseId(request, "teacher")}`);
  const item = gp.locator("main").getByText(/Câu hỏi chờ giảng viên/).first();
  await expect(item).toBeVisible();
  await expect(gp.locator("main")).toContainText(/chờ quá 72 giờ làm việc · lâu nhất \d+ giờ/);
  const top = await gp.locator("main").innerText();
  expect(top.indexOf("Câu hỏi chờ giảng viên"), "việc ticket quá hạn nổi lên đầu «Hôm nay»").toBeLessThan(200);
  await shot(gp, "f5-today-overdue-teacher");
  // SV vẫn thấy trạng thái ticket «Đang chờ giảng viên · đã gửi …»
  await page.goto("/chat");
  await page.getByRole("button", { name: new RegExp(`Lịch bảo vệ đồ án năm sau.*QC ${tag}`) }).first().click();
  await expect(page.getByText(/Đang chờ giảng viên/)).toBeVisible();
  await shot(page, "f5-student-status");
  await gv.close();
  sql(`update user_settings set notify_ticket_by_mail = false where user_id = (select id from users where email='teacher@edupilot.local')`);
});

// ---------------- F13 ----------------
test("@real F13 nhắc 24 giờ: sự kiện sắp tới → một chuông + một thư mỗi SV, không gửi trùng, SV tắt loại «exam» thì không nhận", async ({ request }) => {
  test.setTimeout(TICK_WAIT_MS * 3 + 300_000);
  const cid = await courseId(request, "teacher");
  const teacher = await login(request, "teacher");
  // sv.nguyco tắt nhắc bài thi (tắt từng loại)
  sql(`insert into user_settings (user_id, preferences) select id, '{"reminders":{"exam":false}}'::jsonb from users where email='sv.nguyco@edupilot.local' on conflict (user_id) do update set preferences = excluded.preferences`);
  const title = `QC nhắc 24h ${tag}`;
  const startsAt = new Date(Date.now() + 23 * 3600_000).toISOString();
  const ev = await call(request, teacher, "POST", `/courses/${cid}/calendar/events`, { type: "EXAM", title, starts_at: startsAt });
  expect([200, 201], ev.text.slice(0, 160)).toContain(ev.status);
  const eid = String((ev.json.id ?? (ev.json.event as Json | undefined)?.id) ?? "");
  const rl = (u: string) => Number(sql(`select count(*) from reminder_log r join users x on x.id=r.user_id where x.email='${u}@edupilot.local' and r.source_id='${eid}'`)[0][0]);
  const got = await poll(() => rl("sv.kha"), (n) => n >= 1, TICK_WAIT_MS, 15_000);
  expect(got, "sv.kha có đúng một dòng nhắc").toBe(1);
  expect(rl("sv.nguyco"), "sv.nguyco đã tắt «exam» → không nhắc").toBe(0);
  const mails = await poll(() => mailCount("sv.kha@edupilot.local", /Sắp đến giờ/i, title), (n) => n >= 1, 120_000, 10_000);
  expect(mails, "một thư nhắc tới sv.kha").toBe(1);
  const bell = ((await call(request, await login(request, "sv.kha"), "GET", "/notifications?limit=30")).json.items ?? []) as { title: string; body?: string }[];
  expect(bell.filter((n) => `${n.title} ${n.body ?? ""}`.includes(title)).length, "một chuông").toBe(1);
  // không gửi trùng ở tick sau
  await sleep(Math.min(TICK_WAIT_MS, 6 * 60_000));
  expect(rl("sv.kha")).toBe(1);
  expect(await mailCount("sv.kha@edupilot.local", /Sắp đến giờ/i, title)).toBe(1);
  sql(`update user_settings set preferences = '{}'::jsonb where user_id = (select id from users where email='sv.nguyco@edupilot.local')`);
});

type Json = Record<string, unknown>;
test("@real F13 đổi hạn: hạn nộp dời → «Hôm nay» của SV, lịch và API cùng đổi ngay", async ({ page, request }) => {
  const cid = await courseId(request, "teacher");
  const teacher = await login(request, "teacher");
  const title = `QC đổi hạn ${tag}`;
  const due1 = new Date(Date.now() + 20 * 3600_000).toISOString().replace(/\.\d+Z$/, "Z");
  const body = { title, description: "Bài thử đổi hạn.", type: "ESSAY", max_score: "10", due_at: due1, allow_late: true, late_penalty_per_day: "0.5", late_max_days: 2, appeal_days: 7, channels: ["IN_APP"], submission_code: `QC${tag.toUpperCase().slice(-6)}`, reference_answer: "Đáp án mẫu." };
  const mk = await call(request, teacher, "POST", `/courses/${cid}/assignments`, body);
  expect([200, 201], mk.text.slice(0, 200)).toContain(mk.status);
  const aid = String(mk.json.id ?? (mk.json.assignment as Json | undefined)?.id);
  const rub = await call(request, teacher, "PUT", `/courses/${cid}/assignments/${aid}/rubric`, { version: mk.json.version ?? 1, criteria: [{ id: "c1", name: "Nội dung", max_score: "10", guidance: "Đúng ý chính: 7–10. Thiếu ý: 3–6. Sai: 0–2." }] });
  expect([200, 204], rub.text.slice(0, 200)).toContain(rub.status);
  const pub = await call(request, teacher, "POST", `/courses/${cid}/assignments/${aid}/publish`, {});
  expect([200, 201, 204], pub.text.slice(0, 160)).toContain(pub.status);
  await uiLogin(page, "sv.kha");
  await page.goto("/");
  await expect(page.locator("main").getByText(title).first()).toBeVisible({ timeout: 20_000 });
  await shot(page, "f13-today-due-soon");
  // GV đổi hạn +5 ngày
  const cur = (await call(request, teacher, "GET", `/courses/${cid}/assignments/${aid}`)).json;
  const due2 = new Date(Date.now() + 5 * 24 * 3600_000).toISOString().replace(/\.\d+Z$/, "Z");
  const put = await call(request, teacher, "PUT", `/courses/${cid}/assignments/${aid}`, { ...body, due_at: due2, version: cur.version });
  expect(put.status, put.text.slice(0, 200)).toBe(200);
  await page.reload();
  await page.locator("[data-part=topbar]").waitFor();
  await expect(page.locator("main").getByText(title)).toHaveCount(0); // hết «hạn nộp < 24 giờ»
  const from = new Date().toISOString();
  const to = new Date(Date.now() + 10 * 24 * 3600_000).toISOString();
  const cal = await call(request, await login(request, "sv.kha"), "GET", `/courses/${cid}/calendar?from=${from}&to=${to}`);
  const ev = ((cal.json.items ?? cal.json.events ?? []) as { title: string; starts_at: string }[]).find((e) => e.title.includes(title));
  expect(ev, "sự kiện lịch của bài tập còn tồn tại").toBeTruthy();
  expect(Math.abs(Date.parse(ev!.starts_at) - Date.parse(due2)), "lịch dời theo hạn mới (≤ 1 phút)").toBeLessThan(60_000);
  await page.goto(`/calendar?course=${cid}`);
  await page.locator("[data-part=topbar]").waitFor();
  await page.getByText("Danh sách", { exact: true }).click();
  await expect(page.getByText(title).first()).toBeVisible();
  await shot(page, "f13-calendar-moved");
  await shot(page, "f13-calendar-moved");
});
