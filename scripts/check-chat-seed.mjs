// Kiểm dữ liệu mẫu chat + thread của US-P3-08 bằng API thật, từng tài khoản.
//   node scripts/check-chat-seed.mjs            chat=150±5 … threads=12/3 idempotent=ok   (chạy lại seed.mjs để kiểm idempotent; thêm --no-rerun để bỏ)
//   node scripts/check-chat-seed.mjs threads    bảng trạng thái bài AI của 12 thread lớp 1 (AC2)
import { call, courses, failures, finish, listAll, must, reseed, tok } from "./seed-check-lib.mjs";

const STUDENTS = ["sv.gioi", "sv.kha", "sv.nguyco", ...Array.from({ length: 27 }, (_, i) => `sv${String(i + 4).padStart(2, "0")}`)];
const mode = process.argv[2] && !process.argv[2].startsWith("--") ? process.argv[2] : "all";
const noRerun = process.argv.includes("--no-rerun");

async function chatCounts(c1) {
  let sessions = 0, messages = 0, outside = 0;
  for (const k of STUDENTS) {
    const t = await tok(k);
    for (const s of await listAll(`/chat/sessions?course_id=${c1}`, t)) {
      sessions++;
      for (const m of (await listAll(`/chat/sessions/${s.id}/messages`, t)).filter((x) => x.role === "USER")) {
        messages++;
        if (/thời tiết|phở bò|bóng đá|giá vàng|bộ phim|xương rồng|điện thoại|bóng rổ|cà phê muối|du lịch/i.test(m.content)) outside++;
      }
    }
  }
  return { sessions, messages, outside };
}

/** Bảng đếm bài AI của lớp: thread (Staff thấy hết) → trạng thái bài AI hoặc SKIPPED. */
async function threadTable(course) {
  const t = await tok("teacher");
  const rows = await listAll(`/courses/${course}/threads`, t);
  const by = {};
  for (const r of rows) {
    const v = (await call("GET", `/courses/${course}/threads/${r.id}`, { token: t })).body;
    const ai = v.posts.find((p) => p.kind === "AI");
    const key = ai ? ai.verification_state : v.thread.ai_state === "SKIPPED" ? "SKIPPED" : `AI_${v.thread.ai_state}`;
    by[key] = (by[key] ?? 0) + 1;
  }
  return { total: rows.length, by };
}

const { c1, c2 } = await courses();
const snap = async () => ({ chat: await chatCounts(c1), t1: await threadTable(c1), t2: await threadTable(c2) });
const a = await snap();

must(a.chat.messages >= 145 && a.chat.messages <= 155, `chat=${a.chat.messages} (cần 150±5)`);
must(a.chat.outside === 10, `câu ngoài tài liệu = ${a.chat.outside} (cần 10)`);
must(a.chat.sessions >= 33, `phiên chat = ${a.chat.sessions} (cần ≥ 33: sv.gioi 4 phiên + 29 người)`);
must(a.t1.total === 12, `thread lớp 1 = ${a.t1.total} (cần 12)`);
must(a.t2.total === 3, `thread lớp 2 = ${a.t2.total} (cần 3)`);
const want = { PENDING: 4, VERIFIED: 3, CORRECTED: 2, REJECTED: 1, SKIPPED: 2 };
for (const [k, n] of Object.entries(want)) must((a.t1.by[k] ?? 0) === n, `thread lớp 1 trạng thái ${k} = ${a.t1.by[k] ?? 0} (cần ${n})`);

// sinh viên không thấy bài AI REJECTED (thread vẫn hiện), không thấy khoá cấm
const sv = await tok("sv.gioi");
const seen = await listAll(`/courses/${c1}/threads`, sv);
must(seen.length === 12, `sv.gioi thấy ${seen.length} thread (cần 12: thread có bài AI REJECTED vẫn hiện)`);
const banned = /"(confidence|retrieval_score|groundedness|ai_body|hidden_reason)"/;
for (const r of seen) {
  const raw = JSON.stringify((await call("GET", `/courses/${c1}/threads/${r.id}`, { token: sv })).body);
  if (banned.test(raw)) { must(false, `thread ${r.id}: phản hồi sinh viên có khoá cấm`); break; }
}

let idem = "bỏ qua (--no-rerun)";
if (mode === "threads") {
  console.table(a.t1.by);
} else if (!noRerun) {
  must(reseed() === 0, "chạy lại seed.mjs lỗi");
  const b = await snap();
  idem = JSON.stringify(a) === JSON.stringify(b) ? "ok" : "KHÁC";
  must(idem === "ok", `chạy lại seed làm đổi số liệu: ${JSON.stringify(a)} → ${JSON.stringify(b)}`);
}
finish(`chat=${a.chat.messages} sessions=${a.chat.sessions} outside=${a.chat.outside} threads=${a.t1.total}/${a.t2.total} states=${JSON.stringify(a.t1.by)} idempotent=${idem}`);
void failures;
