import { chromium, openChat, text, composer, send, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("/tmp/qc3e/ui/prep.json", "utf8")); const Q = P.Q; const C1 = "01a1df27-62b5-7775-a346-5c602a43e2f9";
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv[2];
const b = await chromium.launch({ headless: true });
const cnt = (t, s) => (t.match(new RegExp(s, "g")) || []).length;
const idle = async (p, ms = 40000) => { const t0 = Date.now(); await p.waitForTimeout(1500); while (Date.now() - t0 < ms) { if (!(await p.getByRole("button", { name: /^Dừng/ }).count())) return true; await p.waitForTimeout(400); } return false; };
try {
  if (which === "D") { // 35, 36 UI
    const { page } = await openChat(b, "sv.gioi", Q);
    const reqs = []; page.on("request", (r) => reqs.push(r.url()));
    await send(page, P.qs.D); await idle(page); await page.waitForTimeout(1000);
    let t = await text(page);
    const has = t.includes("Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI");
    const learn = page.getByText(/Tìm hiểu/).first(); const hasLearn = await learn.count(); if (hasLearn) await learn.click().catch(() => {}); await page.waitForTimeout(600);
    t = await text(page); const sent = t.includes("Tên, mã số sinh viên, email và số điện thoại được thay bằng ký hiệu trước khi gửi cho AI, rồi hiện lại cho bạn.");
    const sid = psql(`select session_id from chat_messages where role='USER' and content like 'Em là Vũ Hoàng Giang, MSSV 20229001, giải thích về kiểm soát%' order by created_at desc limit 1`);
    const mc = psql(`select masked_count from chat_messages where session_id='${sid}' and role='ASSISTANT'`);
    out("TC-P3-05-35", has && hasLearn > 0 && sent && mc === "2", `dòng «Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI»=${has}; liên kết «Tìm hiểu»=${hasLearn}, mở tại chỗ đúng câu=${sent}; masked_count DB=${mc}`);
    const ph = cnt(await page.evaluate(() => document.body.innerText), "\\[\\[\\s*(SV|MSSV|EMAIL|SDT|CCCD)(_[0-9]*)?");
    await send(page, 'Giải thích đoạn bash [[ -f "$f" ]] làm gì?'); await idle(page); const t2 = await text(page);
    out("TC-P3-05-36-ui", ph === 0 && t2.includes('[[ -f "$f" ]]'), `DOM sau tin có PII: số chuỗi dạng placeholder=${ph}; tin người dùng với [[ -f "$f" ]] hiển thị nguyên=${t2.includes('[[ -f "$f" ]]')} (khung SSE đã P)`);
  }
  if (which === "E") { // 40 UI (lớp 761987 có sự kiện), 19/44 trạng thái thường
    const { page } = await openChat(b, "sv.gioi", C1);
    const reqs = []; page.on("request", (r) => reqs.push(r.url()));
    await send(page, "Tuần này em có lịch gì?"); await idle(page); await page.waitForTimeout(800);
    let t = await text(page); const hasBlock = /Lịch sắp tới/.test(t);
    const sid = psql(`select session_id from chat_messages where role='USER' and content='Tuần này em có lịch gì?' order by created_at desc limit 1`);
    await page.goto(`${BASE}/chat?session=${sid}`); await page.waitForTimeout(3500);
    const t2 = await text(page); const stillBlock = /Lịch sắp tới/.test(t2);
    out("TC-P3-05-40", hasBlock && stillBlock, `khối «Lịch sắp tới» hiện sau khi gửi=${hasBlock}; mở lại phiên (tải lại) vẫn còn=${stillBlock}; chat_messages.blocks có upcoming_events=${psql(`select (blocks::text like '%upcoming_events%') from chat_messages where session_id='${sid}' and role='ASSISTANT'`)}`);
    out("TC-P3-05-44-normal", cnt(t, "Nhờ giảng viên") === 0 && !reqs.some((u) => u.includes("escalate")), `trạng thái thường: «Nhờ giảng viên»=${cnt(t, "Nhờ giảng viên")}, yêu cầu mạng tới /escalate=${reqs.filter((u) => u.includes("escalate")).length}`);
  }
  if (which === "G") { // 41 UI
    const { page } = await openChat(b, "sv.gioi", Q);
    await send(page, P.qs.G); await idle(page); await page.waitForTimeout(800);
    await page.locator('section[aria-label^="Nguồn tham khảo"] button[aria-expanded]').first().click().catch(() => {}); await page.waitForTimeout(600);
    const link = page.getByText(/Xem tài liệu/).first(); const has = await link.count();
    const t = await text(page); const nsrc = /Nguồn|Trích dẫn|\[1\]/.test(t);
    let href = null; if (has) href = await link.evaluate((e) => (e.closest("a") || e).getAttribute("href")).catch(() => null);
    out("TC-P3-05-41", has > 0 && /\/library\//.test(href || ""), `liên kết «Xem tài liệu»=${has}, href=${href}; hiển thị nguồn [n]=${nsrc} (done.citations đủ trường đã P ở API)`);
  }
  if (which === "H") { // 42 (b): xoá tài liệu → «Nguồn đã gỡ»
    const sidd = psql(`select id from documents where id='${P.docdel}'`);
    const sess = psql(`select session_id from chat_messages where role='USER' and content='${P.qs.H}' order by created_at desc limit 1`);
     
    await new Promise((r) => setTimeout(r, 3000));
    const { page } = await openChat(b, "sv.gioi", Q);
    // mở phiên cũ
    const items = page.getByText(/qc3/).first(); await page.goto(`${BASE}/chat?session=${sess}`); await page.waitForTimeout(3500);
    await page.locator('section[aria-label^="Nguồn tham khảo"] button[aria-expanded]').first().click().catch(() => {}); await page.waitForTimeout(600); const t = await text(page);
    out("TC-P3-05-42", /Nguồn đã gỡ/.test(t), `sau khi GV xoá tài liệu: mở lại tin cũ → «Nguồn đã gỡ»=${/Nguồn đã gỡ/.test(t)}; chat_sessions.document_id còn=${psql(`select coalesce(document_id::text,'NULL') from chat_sessions where id='${sess}'`)}; (a) [9] bị bỏ đã P ở API`);
  }
} catch (e) { out("ERR-" + which, false, String(e).slice(0, 300)); }
await b.close();
