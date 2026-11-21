import { chromium, openChat, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("/tmp/qc3e/ui/prep.json", "utf8")); const Q = P.Q;
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv[2];
const b = await chromium.launch({ headless: true });
const open = async (course = Q, w = 1440, h = 900) => { const r = await openChat(b, "sv.gioi", course, w, h); await r.page.goto(`${BASE}/threads`); await r.page.waitForTimeout(3000); return r; };
const compose = async (p) => { await p.getByRole("button", { name: /Đặt câu hỏi/ }).first().click(); await p.waitForTimeout(800); };
const fields = (p) => ({ title: p.getByLabel(/Tiêu đề/).first(), body: p.locator("textarea").first() });
try {
  if (which === "T1") { // 06-07, 06-08
    const { page } = await open(); await compose(page); const f = fields(page);
    let pre = 0; page.on("request", (r) => { if (r.url().includes("/threads/precheck")) pre++; });
    await f.body.click(); const t0 = Date.now(); while (Date.now() - t0 < 3000) { await page.keyboard.type("abcd ", { delay: 40 }); }
    const during = pre; await page.waitForTimeout(1500); const after = pre - during;
    out("TC-P3-06-07", during <= 1 && after <= 1, `gõ liên tục 3 s: precheck trong lúc gõ=${during} (≤1, không mỗi phím); sau khi ngừng ≥800 ms: +${after}`);
    await f.body.fill("MSSV 20229001 và email sv.gioi@edupilot.local"); await page.waitForTimeout(1600);
    const t = await text(page); const lines = (t.match(/Phát hiện[^\n]*/g) || []);
    const dlg = await page.getByRole("dialog").count();
    await f.body.fill(""); await page.waitForTimeout(1500); const t2 = await text(page);
    out("TC-P3-06-08", lines.length === 1 && /MSSV/.test(lines[0]) && /Email/.test(lines[0]) && dlg === 0 && !/Phát hiện/.test(t2), `dòng báo=${JSON.stringify(lines)} (đúng 1 dòng); hộp thoại mở=${dlg}; xoá hết chữ → dòng biến mất=${!/Phát hiện/.test(t2)}`);
  }
  if (which === "T2") { // 06-09, 06-11, 06-13
    const { page } = await open(); await compose(page); const f = fields(page);
    const draft = "Dòng một có dấu tiếng Việt\nDòng hai: Em Bùi Thanh Khải 20229002 được mấy điểm?";
    await f.title.fill("Hỏi điểm"); await f.body.fill(draft);
    const resp = []; page.on("response", (r) => { if (r.request().method() === "POST" && /\/threads$/.test(r.url())) resp.push(r.status()); });
    const reqs = []; page.on("request", (r) => { if (r.method() === "POST" && /\/threads$/.test(r.url())) reqs.push({ k: r.headers()["idempotency-key"], b: r.postData() }); });
    await page.getByRole("button", { name: /^Đăng$/ }).click(); await page.waitForTimeout(2500);
    const dlg = page.getByRole("dialog"); const dt = await dlg.innerText().catch(() => "");
    const btns = await dlg.getByRole("button").allInnerTexts().catch(() => []);
    const acts = btns.map((x) => x.trim()).filter((x) => /Chuyển sang chat riêng|Ẩn thông tin rồi đăng/.test(x));
    await page.keyboard.press("Escape"); await page.waitForTimeout(700);
    const back = (await f.body.inputValue().catch(() => "")) === draft && (await dlg.count()) === 0;
    out("TC-P3-06-09", resp.includes(422) && /Bài viết có thông tin cá nhân/.test(dt) && acts.length === 2 && back, `HTTP đầu=${resp[0]}; tiêu đề hộp thoại="Bài viết có thông tin cá nhân"=${/Bài viết có thông tin cá nhân/.test(dt)}; nút hành động=${JSON.stringify(acts)} (đúng 2, ngoài nút đóng); Esc chỉ quay lại ô soạn, chữ nguyên=${back}`);
    // 06-13 trước (Ẩn thông tin rồi đăng) rồi 06-11 (chuyển sang chat)
    const ft0 = psql(`select count(*) from forum_threads where course_id='${Q}'`);
    await page.getByRole("button", { name: /^Đăng$/ }).click(); await page.waitForTimeout(2000);
    await page.getByRole("button", { name: /Ẩn thông tin rồi đăng/ }).click(); await page.waitForTimeout(3000);
    const second = reqs[reqs.length - 1]; const first = reqs[0];
    const ft1 = psql(`select count(*) from forum_threads where course_id='${Q}'`);
    const tb = psql(`select body from forum_threads where course_id='${Q}' order by created_at desc limit 1`);
    const red = psql(`select count(*) from pii_events where action='REDACTED' and user_id=(select id from users where email='sv.gioi@edupilot.local') and created_at>now()-interval '1 minute'`);
    let rj = {}; try { rj = JSON.parse(second.b); } catch {}
    out("TC-P3-06-13", rj.redact === true && second.k && second.k !== first.k && resp.includes(201) && /\[đã ẩn\]/.test(tb) && Number(ft1) === Number(ft0) + 1 && Number(red) >= 1, `yêu cầu lần hai redact=${rj.redact}, Idempotency-Key mới=${second.k !== first.k}; trả 201=${resp.includes(201)}; thread lưu bản đã thay: ${JSON.stringify(tb.slice(0, 80))}; pii_events REDACTED mới=${red}`);
  }
  if (which === "T3") { // 06-11: chuyển sang chat riêng
    const { page } = await open(); await compose(page); const f = fields(page);
    const draft = "Dòng một có dấu tiếng Việt\nDòng hai: Em Bùi Thanh Khải 20229002 được mấy điểm?";
    await f.title.fill("Hỏi điểm 2"); await f.body.fill(draft);
    const ft0 = psql(`select count(*) from forum_threads where course_id='${Q}'`); const ev0 = psql(`select count(*) from pii_events where action='SWITCHED'`);
    await page.getByRole("button", { name: /^Đăng$/ }).click(); await page.waitForTimeout(2500);
    await page.getByRole("button", { name: /Chuyển sang chat riêng/ }).click(); await page.waitForTimeout(3500);
    const url = page.url(); const val = await page.locator("textarea").first().inputValue();
    const sid = (url.match(/session=([0-9a-f-]+)/) || [])[1];
    const cnt = sid ? psql(`select count(*) from chat_messages where session_id='${sid}'`) : "?";
    const ft1 = psql(`select count(*) from forum_threads where course_id='${Q}'`); const ev1 = psql(`select count(*) from pii_events where action='SWITCHED'`);
    out("TC-P3-06-11", /\/chat\?session=/.test(url) && val.endsWith(draft) && cnt === "0" && ft0 === ft1 && Number(ev1) > Number(ev0), `URL=${url}; ô soạn khớp bản nháp từng byte (kể cả \\n)=${val === draft}; phiên mới có ${cnt} tin; forum_threads ${ft0}→${ft1}; pii_events SWITCHED ${ev0}→${ev1}; không tự gửi`);
  }
  if (which === "T4") { // 06-34 offline/nháp
    const { page, ctx } = await open(); await compose(page); const f = fields(page);
    await f.title.fill("Hỏi về RSA"); await f.body.fill("Nội dung câu hỏi về RSA, rất dài để lưu nháp"); await page.waitForTimeout(3500);
    await ctx.setOffline(true); await page.getByRole("button", { name: /^Đăng$/ }).click(); await page.waitForTimeout(3000);
    const t = await text(page); const kept = (await f.body.inputValue().catch(() => "")).includes("RSA") && (await f.title.inputValue().catch(() => "")) === "Hỏi về RSA";
    const retry = await page.getByRole("button", { name: /Đăng lại/ }).count();
    await ctx.setOffline(false); await page.reload(); await page.waitForTimeout(3500);
    const ask = page.getByRole("button", { name: /Đặt câu hỏi/ }).first(); await ask.click().catch(() => {}); await page.waitForTimeout(1500);
    const restored = (await fields(page).body.inputValue().catch(() => "")).includes("RSA");
    out("TC-P3-06-34", kept && retry > 0 && restored, `ngoại tuyến: chữ trong ô soạn giữ nguyên=${kept}; nút «Đăng lại»=${retry}; sau tải lại nháp tiêu đề+nội dung khôi phục=${restored}`);
  }
  if (which === "T5") { // 06-37 trạng thái; 06-38 375
    const { page } = await open("01a1ef4f-0195-70a8-bc04-3e0b37610973"); // QC3S6C có thread
    const D = psql("select id from courses where class_code='QC3S6D'");
    const { page: pe } = await open(D);
    const te = await text(pe); const emptyTitle = /Chưa có câu hỏi nào/.test(te); const emptyFull = /Chưa có câu hỏi nào\. Đặt câu hỏi đầu tiên\./.test(te); const emptyBtns = await pe.getByText("Chưa có câu hỏi nào", { exact: true }).locator("xpath=..").getByRole("button").count(); const empty = emptyTitle && !emptyFull && emptyBtns === 0;
    await pe.route("**/threads*", (r) => r.request().url().includes("/api/v1/") && r.request().url().includes("/threads") && r.request().method() === "GET" ? r.fulfill({ status: 500, body: "{}" }) : r.continue());
    await pe.reload(); await pe.waitForTimeout(4000); const t5 = await text(pe); const errBtn = await pe.getByRole("button", { name: /Thử lại/ }).count();
    out("TC-P3-06-37", empty && errBtn > 0, `lớp rỗng: theo P-11-80 (BA 263920fb): chữ «Chưa có câu hỏi nào»=${emptyTitle}, không câu mời «Đặt câu hỏi đầu tiên.»=${!emptyFull}, số nút trong khối rỗng=${emptyBtns} (mong 0); API 500 → có nút «Thử lại»=${errBtn}; (khung xương khi tải: kiểm bằng route delay ở ca 52)`);
  }
} catch (e) { out("ERR-" + which, false, String(e).slice(0, 300)); }
await b.close();
