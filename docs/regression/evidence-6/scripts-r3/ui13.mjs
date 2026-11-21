import { chromium, text, out, openChat, send } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("/tmp/qc3e/ui/prep.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const b = await chromium.launch({ headless: true }); let t = "";
try {
  const { page } = await openChat(b, "sv.gioi", P.Q);
  psql("alter table exam_attempts rename to exam_attempts_qc3tmp");
  try { await send(page, "xin chào"); await page.waitForTimeout(3500); t = await text(page); }
  finally { psql("alter table exam_attempts_qc3tmp rename to exam_attempts"); }
  const ok = t.includes("Chat chưa dùng được lúc này. Thử lại sau ít phút.") && !t.includes("Chat tạm thời chưa dùng được");
  out("TC-P3-05-26", ok, `bảng exam_attempts đổi tên tạm ~4 s (khoá không tra được) → UI chat hiện «Chat chưa dùng được lúc này. Thử lại sau ít phút.»=${ok} (chữ cũ không còn); API 503 CHAT_UNAVAILABLE (lượt trước); bảng đã đổi lại=${psql("select to_regclass('exam_attempts')")}`);
} catch (e) { out("ERR-13", false, String(e).slice(0, 300)); }
await b.close();
