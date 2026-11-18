import { chromium, openChat, text, composer, send, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("/tmp/qc3e/ui/prep.json", "utf8")); const Q = P.Q;
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv[2];
const b = await chromium.launch({ headless: true });
const lastAssistant = (sid) => psql(`select content from chat_messages where session_id='${sid}' and role='ASSISTANT' order by created_at desc limit 1`);
const newSession = async (p) => { await p.getByRole("button", { name: /Phiên mới|Cuộc trò chuyện mới|Chat mới/ }).first().click().catch(() => {}); };
const waitDone = async (p, ms = 60000) => { await p.waitForFunction(() => !document.body.innerText.includes("Dừng") || true, null, { timeout: 1000 }).catch(() => {}); const t0 = Date.now(); while (Date.now() - t0 < ms) { if (!(await p.getByRole("button", { name: /^Dừng/ }).count())) return true; await p.waitForTimeout(500); } return false; };
try {
  if (which === "A") { // 05-09: mạng chậm + tải lại giữa chừng
    const { page } = await openChat(b, "sv.gioi", Q);
    const cdp = await page.context().newCDPSession(page); await cdp.send("Network.enable");
    await cdp.send("Network.emulateNetworkConditions", { offline: false, latency: 400, downloadThroughput: 400 * 1024 / 8, uploadThroughput: 400 * 1024 / 8 });
    const q = P.qs.A; await send(page, q);
    await page.waitForFunction((x) => document.body.innerText.includes("Đây là câu"), null, { timeout: 30000 }).catch(() => {});
    await page.waitForTimeout(3000);
    const before = (await text(page)).split("\n").filter((l) => l.startsWith("Đây là câu")).join(" ");
    await page.reload(); await page.waitForTimeout(3000);
    await waitDone(page, 60000); await page.waitForTimeout(4000);
    const after = (await text(page)); const n = (after.match(/Đây là câu trả lời giả lập/g) || []).length;
    const sid = psql(`select session_id from chat_messages where role='USER' and content='${q}' order by created_at desc limit 1`);
    const db = lastAssistant(sid); const shown = db.slice(0, 40);
    out("TC-P3-05-09", before.length > 5 && n === 1 && after.includes(shown.slice(0, 30)) && psql(`select stream_status from chat_messages where session_id='${sid}' and role='ASSISTANT'`) === "DONE", `mạng 400 ms/400 kbps; chữ đã sinh trước khi tải lại: ${JSON.stringify(before.slice(0, 50))}; sau tải lại bong bóng AI xuất hiện ${n} lần (không lặp), tiếp tục tới DONE; DB content=${JSON.stringify(db.slice(0, 60))}`);
  }
  if (which === "B") { // 05-14 UI + 17 UI + 19/44
    const { page } = await openChat(b, "sv.gioi", Q);
    await send(page, P.qs.B); await page.waitForTimeout(3500);
    const stop = page.getByRole("button", { name: /^Dừng/ }); const hadStop = await stop.count();
    await stop.first().click().catch(() => {}); await page.waitForTimeout(1500);
    const t = await text(page); const ta = composer(page);
    const sid = psql(`select session_id from chat_messages where role='USER' and content='${P.qs.B}' order by created_at desc limit 1`);
    out("TC-P3-05-14", hadStop > 0 && t.includes("Đã dừng.") && (await ta.isEnabled()) && psql(`select stream_status from chat_messages where session_id='${sid}' and role='ASSISTANT'`) === "CANCELLED", `nút «Dừng» có=${hadStop}; chữ «Đã dừng.» hiện=${t.includes("Đã dừng.")}; ô soạn mở lại=${await ta.isEnabled()}; DB stream_status=CANCELLED; (API: huỷ ≤1 s, llm_audit=cancelled, đã P ở lượt trước)`);
    out("TC-P3-05-19-normal", !t.includes("Nhờ giảng viên"), `trạng thái bị huỷ/thường: find "Nhờ giảng viên"=${(t.match(/Nhờ giảng viên/g) || []).length}`);
  }
} catch (e) { out("ERR-" + which, false, String(e).slice(0, 300)); }
await b.close();
