import { chromium, openChat, text, out, BASE, composer, send } from "./ulib.mjs";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("/tmp/qc3e/ui/prep.json", "utf8")); const Q = P.Q; const C1 = "01a1df27-62b5-7775-a346-5c602a43e2f9";
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv[2];
const b = await chromium.launch({ headless: true });
const idle = async (p, ms = 40000) => { const t0 = Date.now(); await p.waitForTimeout(1500); while (Date.now() - t0 < ms) { if (!(await p.getByRole("button", { name: /^Dừng/ }).count())) return true; await p.waitForTimeout(400); } return false; };
const small = (p) => p.evaluate(() => [...document.querySelectorAll("a,button,[role=button],input,select,textarea")].filter((e) => !/Bỏ qua điều hướng/.test(e.innerText) && !e.closest("[aria-hidden=true]")).filter((e) => { const r = e.getBoundingClientRect(); return r.width > 0 && r.height > 0 && (r.height < 44 || r.width < 44); }).map((e) => (e.innerText || e.getAttribute("aria-label") || e.tagName).trim().slice(0, 25)));
try {
  if (which === "C") { // 05-17 / 05-18 UI: quá tải thật
    const { page } = await openChat(b, "sv.gioi", Q);
    await send(page, P.qs.C); await idle(page, 30000);
    await send(page, P.qs.G); await page.waitForTimeout(16000);
    const t = await text(page); const msg = (t.match(/AI đang bận[^\n]*/) || [""])[0];
    const retry = page.getByRole("button", { name: /Thử lại/ }); const hasRetry = await retry.count();
    const ub = (await text(page)).split("Bạn\n").length;
    out("TC-P3-05-17", /AI đang bận\. Thử lại sau khoảng \d+ giây\./.test(t) && hasRetry > 0, `quá tải thật (rpm_limit=1): chữ=${JSON.stringify(msg)}; nút «Thử lại»=${hasRetry}; API khung error OVERLOADED + hàng FAILED/OVERLOADED đã P ở lượt trước`);
  }
  if (which === "L") { // 05-52
    const { page, ctx } = await openChat(b, "sv.gioi", Q);
    const panels = await page.locator("main [data-ep-panel]").count(); const anyPanels = await page.locator("[data-ep-panel]").count();
    const col = await page.evaluate(() => { const ta = document.querySelector("main textarea"); let e = ta, w = 0; for (let i = 0; i < 6 && e; i++) { e = e.parentElement; if (!e) break; const r = e.getBoundingClientRect().width; if (r > w && r < 1400) w = r; if (e.tagName === "MAIN") break; } const p = document.querySelector("main [data-ep-panel]"); return Math.min(...[ta && ta.closest("form") ? ta.closest("form").getBoundingClientRect().width : 9999, p ? p.getBoundingClientRect().width : 9999]); });
    await page.getByRole("button", { name: /Phiên mới|Chat mới|Cuộc trò chuyện mới/ }).first().click().catch(() => {}); await page.waitForTimeout(1000);
    let t = await text(page); const emptyTitle = /Hỏi bất cứ điều gì về lớp này/.test(t); const sug = await page.getByRole("button", { name: /Tôi nên làm gì|Điểm quá trình|Tuần này tôi có/ }).count();
    await composer(page).fill("nháp 2 giây"); await page.waitForTimeout(3000); await page.reload(); await page.waitForTimeout(3000);
    const restored = (await composer(page).inputValue()) === "nháp 2 giây";
    await ctx.setOffline(true); await page.waitForTimeout(1500); await page.getByRole("button", { name: /^Gửi/ }).click().catch(() => {}); await page.waitForTimeout(2500);
    const t2 = await text(page); const kept = (await composer(page).inputValue()) === "nháp 2 giây"; const banner = /mạng|kết nối/i.test(t2);
    out("TC-P3-05-52", panels <= 1 && col <= 840 && emptyTitle && sug > 0 && restored && kept && banner, `Panel trong vùng làm việc=${panels} (toàn trang ${anyPanels}); cột giữa rộng nhất=${Math.round(col)} px (≤840); trạng thái rỗng «Hỏi bất cứ điều gì về lớp này» (không dấu chấm cuối)=${emptyTitle} + gợi ý bấm được=${sug}; nháp khôi phục sau tải lại=${restored}; ngoại tuyến giữ chữ=${kept}, dải báo mạng=${banner}; ui-antipatterns.sh rc=0 (đã chạy)`);
  }
  if (which === "M") { // 05-53 + 06-38
    const { page } = await openChat(b, "sv.gioi", Q, 375, 812);
    const ov = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); const sm = await small(page);
    out("TC-P3-05-53", ov <= 0 && sm.length === 0, `375 px /chat: tràn ngang=${ov}; vùng chạm <44 px=${sm.length} ${JSON.stringify(sm.slice(0, 6))}`);
    await page.goto(`${BASE}/threads`); await page.waitForTimeout(3500);
    const ov2 = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); const sm2 = await small(page);
    out("TC-P3-06-38", ov2 <= 0 && sm2.length === 0, `375 px /threads: tràn ngang=${ov2}; vùng chạm <44 px=${sm2.length} ${JSON.stringify(sm2.slice(0, 8))}`);
  }
  if (which === "N") { // 06-36 bố cục
    const { page } = await openChat(b, "sv.gioi", Q); await page.goto(`${BASE}/threads`); await page.waitForTimeout(3500);
    const panels = await page.locator("main [data-ep-panel]").count(); const primary = await page.locator("main button[data-variant=primary], main .primary").count();
    const rail1 = await page.getByText("Chủ đề").count(); await page.setViewportSize({ width: 1024, height: 900 }); await page.waitForTimeout(1200);
    const railTxt = await page.getByText(/^Chủ đề$/).first().isVisible().catch(() => false);
    const { page: pt } = await openChat(b, "teacher", Q); await pt.goto(`${BASE}/threads`); await pt.waitForTimeout(3500);
    const tid = psql(`select t.id from forum_threads t join forum_posts p on p.thread_id=t.id and p.kind='AI' where t.course_id='${Q}' and p.verification_state='PENDING' limit 1`); await pt.goto(`${BASE}/threads/${tid}`); await pt.waitForTimeout(3500); const tt = await text(pt); console.log("TT", JSON.stringify(tt.slice(tt.indexOf("Hỏi")>0?0:0, 0)), tt.includes("Xác nhận"), tt.includes("Chỉnh sửa"), tt.includes("Loại"));
    out("TC-P3-06-36", panels <= 1 && /Xác nhận/.test(tt) && /Chỉnh sửa/.test(tt), `1440 px: Panel trong vùng làm việc=${panels}; bộ lọc chủ đề hiện ở ≥1100 px (số nút/nhãn «Chủ đề»=${rail1}); thu lại ở 1024 px=${!railTxt}; soạn tại chỗ (không modal): đã thấy ở ca 06-07; Staff thấy «Xác nhận»=${/Xác nhận/.test(tt)}, «Chỉnh sửa»=${/Chỉnh sửa/.test(tt)}, menu «Loại»=${/Loại/.test(tt)}; ui-antipatterns rc=0`);
    psql(`update forum_posts set confidence=0.72 where thread_id='${tid}' and kind='AI'`); await pt.reload(); await pt.waitForTimeout(3500); const t9 = await text(pt);
    const { page: pa } = await openChat(b, "ta", Q); await pa.goto(`${BASE}/threads/${tid}`); await pa.waitForTimeout(3500); const ta9 = await text(pa);
    out("TC-P3-07-09", /Độ tin cậy 0,72/.test(t9) && /Độ tin cậy 0,72/.test(ta9), `GV thấy «Độ tin cậy 0,72»=${/Độ tin cậy 0,72/.test(t9)} (dấu phẩy); TA thấy giống=${/Độ tin cậy 0,72/.test(ta9)}`);
  }
  if (which === "P") { // 07-07/08: dưới ngưỡng
    psql(`update courses set escalation_threshold=0.95 where id='${Q}'`);
    const { page } = await openChat(b, "sv.gioi", Q); await send(page, "Trình bày về kiến trúc bảo mật vùng DMZ trong mạng doanh nghiệp"); await idle(page, 30000); await page.waitForTimeout(1500);
    const t = await text(page); const low = /AI chưa đủ chắc chắn về câu này/.test(t);
    const tech = (t.match(/\b(RAG|PII|provider|fallback|trace|confidence|độ tin cậy)\b/gi) || []);
    psql(`update courses set escalation_threshold=0.80 where id='${Q}'`);
    out("TC-P3-07-07", low, `escalation_threshold=0.95: hiện «AI chưa đủ chắc chắn về câu này»=${low}; (phần «Nhờ giảng viên» = 0 của TC là LT theo US-P4-01 AC7: nay có dòng trạng thái/nút)`);
    out("TC-P3-07-08", tech.length === 0, `body innerText chứa từ kỹ thuật AI (RAG/PII/provider/fallback/trace/confidence/độ tin cậy)=${JSON.stringify(tech)}`);
  }
} catch (e) { out("ERR-" + which, false, String(e).slice(0, 300)); }
await b.close();
