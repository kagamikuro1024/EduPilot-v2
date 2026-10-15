// US-S7-SEED AC5 / TC-15: bài AI VERIFIED / CORRECTED và ticket đã lưu tri thức PHẢI được trợ lý chat riêng trích dẫn ("Hỏi đáp đã được giảng viên xác nhận").
//   node scripts/check-knowledge-seed.mjs
// Vì sao không hỏi bằng tiêu đề thread: ở LLM_PROVIDER=fake vectơ nhúng là băm của CẢ chuỗi (không có ngữ nghĩa) và đoạn tri thức được nhúng là
// "Câu hỏi: <tiêu đề>\n<nội dung>\nTrả lời: <đáp>" (knowledge.chunkText) ⇒ chỉ chuỗi đúng y như vậy mới có cosine 1. Provider thật không có giới hạn này.
// Nên script đọc đúng chữ đoạn tri thức từ Postgres (đặt SEED_PSQL_CMD nếu stack không phải compose project `edupilot`) và gửi nó như một câu hỏi, rồi xoá phiên thử.
import { execSync } from "node:child_process";
import { call, courses, failures, finish, must, root, tok, API, ORIGIN } from "./seed-check-lib.mjs";

const psql = process.env.SEED_PSQL_CMD ?? `docker compose --env-file .env.local -f docker-compose.local.yml -p ${process.env.COMPOSE_PROJECT_NAME || "edupilot"} exec -T postgres psql -U edupilot -d edupilot -At`;
const sql = `select k.source_type || '|' || encode(convert_to(c.text, 'UTF8'), 'hex') from content_chunks c join knowledge_items k on k.chunk_id = c.id order by k.source_type, k.created_at`;
const rows = execSync(`${psql} -c "${sql}"`, { cwd: root, encoding: "utf8" }).trim().split("\n").filter(Boolean).map((l) => {
  const [type, hex] = l.trim().split("|");
  return { type, text: Buffer.from(hex, "hex").toString("utf8") };
});
must(rows.filter((r) => r.type === "POST").length >= 5, `tri thức từ bài AI = ${rows.filter((r) => r.type === "POST").length}, cần ≥ 5 (3 VERIFIED + 2 CORRECTED)`);
must(rows.some((r) => r.type === "TICKET"), "thiếu tri thức từ ticket ANSWERED");

const { c1 } = await courses();
const sv = await tok("sv.gioi");
let cited = 0;
for (const r of rows) {
  const sess = (await call("POST", "/chat/sessions", { token: sv, json: { course_id: c1 } })).body;
  const res = await fetch(`${API}/chat/sessions/${sess.id}/messages`, {
    method: "POST", headers: { Authorization: `Bearer ${sv}`, "Content-Type": "application/json", Accept: "text/event-stream", Origin: ORIGIN, "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify({ content: r.text }),
  });
  const out = await res.text();
  // Lần đầu: citations có tài liệu ảo này. Lần sau cùng câu hỏi: trả từ cache câu trả lời (agent.Cache, theo lớp) — cache chỉ giữ chữ, `citations` rỗng, nhưng chữ vẫn mang tên tài liệu và không `low_confidence`.
  const ok = res.ok && /Hỏi đáp đã được giảng viên xác nhận/.test(out) && !/"low_confidence":true/.test(out);
  if (ok) cited++;
  must(ok, `${r.type} "${r.text.slice(0, 40).replace(/\n/g, " ")}…": không có trích dẫn tri thức (${res.status})`);
  await call("DELETE", `/chat/sessions/${sess.id}`, { token: sv });
}
finish(`tri thức=${rows.length} (POST ${rows.filter((r) => r.type === "POST").length}, TICKET ${rows.filter((r) => r.type === "TICKET").length}) có trích dẫn=${cited}`);
void failures;
