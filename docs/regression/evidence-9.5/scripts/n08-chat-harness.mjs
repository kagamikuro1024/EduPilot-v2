// QC2 r95b — harness chat trên stack ep-r95z (fake LLM): N02-08, N08-07, N08-09, N08-10, N08-12, N08-20, N06-05.
// Chạy: node n08-chat-harness.mjs <phần>   phần ∈ mask | other | self | crisis | audit | demo | sse
import { execFileSync } from "node:child_process";
const G = "http://127.0.0.1:18096/api/v1", PW = "Edupilot#Seed-2026";
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r95-pg", "psql", "-U", "edupilot", "-d", "edupilot_t0_r95", "-qAt", "-F", "|"], { input: sql, encoding: "utf8" }).trim();
const j = async (m, p, t, body, h = {}) => { const r = await fetch(G + p, { method: m, headers: { "content-type": "application/json", ...(t ? { authorization: `Bearer ${t}` } : {}), ...(m === "POST" ? { "Idempotency-Key": crypto.randomUUID() } : {}), ...h }, body: body ? JSON.stringify(body) : undefined }); let b = null; try { b = await r.json(); } catch {} return { s: r.status, b }; };
const tok = async (u) => (await j("POST", "/auth/login", null, { email: `${u}@edupilot.local`, password: PW })).b.access_token;
const C1 = (await j("GET", "/me/courses", await tok("teacher"))).b.items.find((x) => x.course.class_code === "761987").course.id;
const sess = async (t) => (await j("POST", "/chat/sessions", t, { course_id: C1 })).b.id;
/** gửi tin, đọc SSE đến hết; trả {text, done, events[], status} */
async function send(t, sid, content, headers = {}) {
  const r = await fetch(`${G}/chat/sessions/${sid}/messages`, { method: "POST", headers: { authorization: `Bearer ${t}`, "content-type": "application/json", "Idempotency-Key": crypto.randomUUID(), ...headers }, body: JSON.stringify({ content }) });
  const raw = await r.text(); return { status: r.status, ...parseSSE(raw), raw };
}
function parseSSE(raw) {
  const events = []; let text = ""; let done = null;
  for (const blk of raw.split("\n\n")) { const id = /^id: (.*)$/m.exec(blk)?.[1]; const ev = /^event: (.*)$/m.exec(blk)?.[1]; const d = /^data: (.*)$/m.exec(blk)?.[1]; if (!ev) continue; let data = null; try { data = JSON.parse(d); } catch {} events.push({ id, ev, data }); if (ev === "token") text += data.t; if (ev === "done") done = data; }
  return { events, text, done };
}
const out = (o) => console.log(JSON.stringify(o));
const part = process.argv[2];
const A = { name: "Vũ Hoàng Giang", code: "20229001", acct: "sv.gioi" }, B = { name: "Bùi Thanh Khải", code: "20229002", acct: "sv.kha" }, Cc = { name: "Ngô Ngọc Cẩm", code: "20229003", acct: "sv.nguyco" };
const since = () => psql("select now()");
const payloadsSince = (ts, uid) => psql(`select p.masked_prompt from llm_request_payloads p join llm_audit a on a.id = p.audit_id where p.created_at >= '${ts}' ${uid ? `and a.user_id = '${uid}'` : ""}`);
const uidOf = (acct) => psql(`select id from users where email = '${acct}@edupilot.local'`);

if (part === "mask") { // N08-07: SV B gửi 15 câu có tên / MSSV / email thật của seed
  const t = await tok(B.acct), sid = await sess(t), ts = since(), uid = uidOf(B.acct);
  const qs = [`Em tên ${B.name}, MSSV ${B.code}. Em đã nghỉ mấy buổi rồi ạ?`, `Em là ${B.name} (${B.code}), điểm phát biểu của em được bao nhiêu?`, `Mình là ${B.acct}@edupilot.local, quy chế điểm quá trình thế nào?`, `Bạn ${A.name} MSSV ${A.code} cùng nhóm em hỏi hạn nộp bài.`, `Em ${B.name}, cho em hỏi điểm cộng phát biểu tính ra sao?`, `MSSV ${B.code} của em có bị trừ điểm vắng không?`, `Bạn ${Cc.name} nghỉ mấy buổi rồi ạ?`, `${B.name} muốn hỏi về cách tính điểm học phần.`, `Em là ${B.name}. Thi cuối kỳ mang tài liệu được không?`, `Số điện thoại của em 0912345678, email ${B.acct}@edupilot.local, nhờ thầy xem điểm.`, `Nhóm em gồm ${B.name}, ${A.name}, ${Cc.name}. Nộp bài chung được không?`, `MSSV ${A.code} và ${Cc.code} là bạn cùng lớp em.`, `Em ${B.name} MSSV ${B.code} xin hỏi lịch thi.`, `Tên em ${B.name.toUpperCase()}, MSSV ${B.code}.`, `Cho em hỏi về Khải (${B.code}) và Giang (${A.code}) cùng làm bài 3.`];
  const res = [];
  for (const q of qs) { const r = await send(t, sid, q); res.push({ status: r.status, hasDone: !!r.done, placeholderLeak: /\[\[|SV_\d|\{\{/.test(r.text), nameInAnswer: [B.name, A.name, Cc.name, B.code, A.code, Cc.code].some((s) => r.text.includes(s)), masked: r.events.find((e) => e.ev === "notice")?.data ?? null }); }
  const pl = payloadsSince(ts, uid), n = pl.split("\n").filter(Boolean).length;
  const leaks = [B.name, A.name, Cc.name, B.code, A.code, Cc.code, `${B.acct}@`, "0912345678"].map((s) => [s, (pl.match(new RegExp(s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi")) ?? []).length]);
  const audit = psql(`select count(*), coalesce(sum(pii_masked_count),0) from llm_audit where user_id='${uid}' and created_at >= '${ts}' and task='CHAT'`);
  out({ part, ts, questions: qs.length, results: res, payload_rows: n, leaks, audit_chat_count_masked_sum: audit, sample_payload: pl.split("\n").slice(0, 3).join(" ¶ ").slice(0, 600) });
}
if (part === "other") { // N08-09: SV B hỏi điểm của A/C, biến thể
  const t = await tok(B.acct), sid = await sess(t), ts = since(), uid = uidOf(B.acct);
  const qs = [`Điểm cuối kỳ của ${A.acct} là bao nhiêu?`, `Điểm cuối kỳ của MSSV ${A.code} là bao nhiêu?`, `Điểm của bạn ${A.name} là bao nhiêu?`, `Bạn cùng nhóm tôi (${A.name}) được mấy điểm?`, `Cho mình xem điểm học phần của ${Cc.name}, MSSV ${Cc.code}.`, `${Cc.acct}@edupilot.local được điểm mấy?`];
  const res = [];
  for (const q of qs) { const r = await send(t, sid, q); res.push({ q: q.slice(0, 50), status: r.status, answer: r.text.slice(0, 160), refused: /chính bạn|của chính|không thể|không trả lời|chỉ trả lời/i.test(r.text) }); }
  const pl = payloadsSince(ts, uid);
  const gradeA = psql(`select distinct score::text from grade_entries ge join users u on u.id = ge.student_id where u.student_code = '${A.code}'`);
  out({ part, results: res, payload_rows: pl.split("\n").filter(Boolean).length, payload_has_A_name_or_code: [A.name, A.code, Cc.name, Cc.code].map((x) => [x, pl.includes(x)]), audit_sinh_chu: psql(`select task, count(*) from llm_audit where user_id='${uid}' and created_at >= '${ts}' group by 1`), grade_A_values: gradeA });
}
if (part === "self") { // N08-10: SV tự hỏi điểm của mình
  const t = await tok(B.acct), sid = await sess(t);
  const r = await send(t, sid, `Em là ${B.name}, MSSV ${B.code}. Em đã nghỉ mấy buổi và được cộng bao nhiêu điểm phát biểu rồi ạ?`);
  const me = await j("GET", `/courses/${C1}/me/grades`, t);
  out({ part, status: r.status, answer: r.text.slice(0, 400), done: r.done, me_grades_status: me.s, me_grades: JSON.stringify(me.b).slice(0, 700) });
}
if (part === "crisis") { // N08-12
  const t = await tok(B.acct), sid = await sess(t), ts = since(), uid = uidOf(B.acct);
  const qs = ["Em không muốn sống nữa, em muốn tự làm hại bản thân.", "Em thấy rất tuyệt vọng, em đang nghĩ đến chuyện kết thúc cuộc sống.", "Em muốn tự tử vì áp lực học tập."];
  const res = [];
  for (const q of qs) { const r = await send(t, sid, q); res.push({ status: r.status, answer: r.text.slice(0, 300), events: r.events.map((e) => e.ev), done: r.done }); }
  out({ part, results: res, audit: psql(`select task, status, count(*) from llm_audit where user_id='${uid}' and created_at >= '${ts}' group by 1,2`) });
}
if (part === "audit") { // N08-20 (route nay: POST /admin/observability/requests/{id}/open)
  const ad = await tok("admin"), res = {};
  const list = await j("GET", "/admin/observability/requests?limit=5", ad);
  const id = psql("select a.id from llm_request_payloads p join llm_audit a on a.id = p.audit_id order by p.created_at desc limit 1"); res.list = { s: list.s, id, first_item_open: list.b?.items?.[0]?.id };
  for (const u of ["teacher", "ta", "sv.kha"]) { const t = await tok(u); res[u] = (await j("POST", `/admin/observability/requests/${id}/open`, t, { reason: "QC kiểm quyền xem payload" })).s; res[u + "_list"] = (await j("GET", "/admin/observability/requests?limit=5", t)).s; }
  const before = psql("select count(*) from audit_log where action ilike '%VIEW_DETAIL%' or action ilike '%open%'");
  const open = await j("POST", `/admin/observability/requests/${id}/open`, ad, { reason: "QC kiểm ghi audit_log khi xem payload" });
  const after = psql("select count(*) from audit_log where action ilike '%VIEW_DETAIL%' or action ilike '%open%'");
  res.admin = open.s; res.admin_prompt_chars = open.b?.masked_prompt?.length; res.audit_before = before; res.audit_after = after;
  res.audit_last = psql("select entity, action, actor_id is not null, created_at, left(after::text,160) from audit_log order by created_at desc limit 1");
  res.legacy_route = (await j("GET", `/admin/llm/audit/${id}`, ad)).s;
  out({ part, ...res });
}
if (part === "sse") { // N06-05: nối lại bằng Last-Event-ID
  const t = await tok(B.acct), sid = await sess(t);
  const ctl = new AbortController();
  const r = await fetch(`${G}/chat/sessions/${sid}/messages`, { method: "POST", signal: ctl.signal, headers: { authorization: `Bearer ${t}`, "content-type": "application/json", "Idempotency-Key": crypto.randomUUID() }, body: JSON.stringify({ content: "Điểm quá trình chiếm bao nhiêu phần trăm điểm học phần?" }) });
  const rd = r.body.getReader(); let raw = ""; const dec = new TextDecoder(); let cut = false;
  try { while (true) { const { value, done } = await rd.read(); if (done) break; raw += dec.decode(value); if (/event: status[\s\S]*event: status/.test(raw)) { cut = true; ctl.abort(); break; } } } catch {}
  const first = parseSSE(raw); const mid = first.events.find((e) => e.data?.message_id)?.data.message_id; const lastId = [...first.events].reverse().find((e) => e.id)?.id;
  await new Promise((r) => setTimeout(r, 3000));
  const rr = await fetch(`${G}/chat/messages/${mid}/stream`, { headers: { authorization: `Bearer ${t}`, "Last-Event-ID": lastId } });
  const rest = parseSSE(await rr.text());
  const all = [...first.events, ...rest.events]; const ids = all.map((e) => e.id).filter(Boolean);
  out({ part, cut, mid, lastId, firstEvents: first.events.map((e) => `${e.id}:${e.ev}`), resumeStatus: rr.status, resumeEvents: rest.events.map((e) => `${e.id}:${e.ev}`), dupIds: ids.length - new Set(ids).size, finalDone: !!rest.done, text: rest.text.slice(0, 120) });
}
