#!/usr/bin/env node
// (bản N06-08 riêng) TC-N06-08 (SmkS4, 2026-11-21): stack smk91 (qc2 sở hữu, profile ingest). Tải lên 3 tài liệu → dừng smk91-docling-1 120 s → chạy lại.
// Trong lúc docling tắt: chat hỏi về tài liệu đã READY (N06-08). Sau khi docling lên: không tài liệu nào kẹt PROCESSING > 10 phút (N06-07).
//   node docs/regression/evidence-9.5/scripts/n06-07-08-docling-smk91.mjs  → in JSON; cần smk91 chạy đủ (gateway 1343, worker, docling).
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { miniPdf } from "../../../../scripts/chat-seed-data.mjs";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const S = "https://localhost:1343/api/v1", PW = "Edupilot#Seed-2026", DOCLING = "smk91-docling-1";
const run = (c, a, input) => { const r = spawnSync(c, a, { encoding: "utf8", input }); if (r.status !== 0) throw new Error(`${c} ${a.join(" ")}: ${r.stderr}`); return r.stdout.trim(); };
const psql = (q) => run("docker", ["exec", "-i", "smk91-postgres-1", "sh", "-c", "psql -U edupilot -d edupilot -At -F'|' -v ON_ERROR_STOP=1; exit 0"], q);
const sleep = (s) => new Promise((r) => setTimeout(r, s * 1000));
const api = (p, tok, init = {}) => fetch(`${S}${p}`, { ...init, headers: { "content-type": "application/json", ...(tok ? { authorization: `Bearer ${tok}` } : {}), ...(init.headers ?? {}) } });
const login = async (u) => (await (await api("/auth/login", null, { method: "POST", body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }), headers: { "Idempotency-Key": crypto.randomUUID() } })).json()).access_token;

const course = psql("select id from courses where class_code='761987'");
const T = await login("teacher"), SV = await login("sv.kha");
const stamp = Date.now();
const upload = async (n) => {
  const bytes = miniPdf([`Chaos docling ${stamp} tài liệu ${n}`, "Điều 1. Điểm quá trình chiếm 40% điểm học phần.", `Điều 2. Mã kiểm thử ${stamp}-${n}.`]);
  const sha = createHash("sha256").update(bytes).digest("hex");
  const pre = await (await api(`/courses/${course}/uploads/presign`, T, { method: "POST", body: JSON.stringify({ purpose: "document", filename: `chaos-${stamp}-${n}.pdf`, mime_type: "application/pdf", size_bytes: bytes.length, sha256: sha }), headers: { "Idempotency-Key": crypto.randomUUID() } })).json();
  const put = await fetch(pre.url, { method: pre.method, headers: pre.headers, body: bytes });
  if (!put.ok) throw new Error(`PUT → ${put.status}`);
  const done = await api(`/courses/${course}/uploads/complete`, T, { method: "POST", body: JSON.stringify({ upload_id: pre.upload_id, title: `chaos-docling-${stamp}-${n}`, type: "OTHER" }), headers: { "Idempotency-Key": `chaos-doc-${pre.upload_id}` } });
  if (![200, 202].includes(done.status)) throw new Error(`complete → ${done.status} ${await done.text()}`);
};
const chat = async (label, text) => {
  const sid = (await (await api("/chat/sessions", SV, { method: "POST", body: JSON.stringify({ course_id: course }), headers: { "Idempotency-Key": crypto.randomUUID() } })).json()).id;
  const t = Date.now();
  const r = await api(`/chat/sessions/${sid}/messages`, SV, { method: "POST", body: JSON.stringify({ content: text }), headers: { "Idempotency-Key": crypto.randomUUID(), accept: "text/event-stream" } });
  const body = await r.text();
  const done = body.split("\n").filter((l) => l.startsWith("data:") && /"citations"/.test(l)).pop() ?? "";
  let cites = -1, degraded = null;
  try { const j = JSON.parse(done.slice(5)); cites = j.citations?.length ?? 0; degraded = j.degraded; } catch { /* không có done */ }
  return { label, status: r.status, seconds: +((Date.now() - t) / 1000).toFixed(1), citations: cites, degraded, has_done: /event: done/.test(body), error: (body.match(/"code":"([A-Z_]+)"/) ?? [])[1] ?? null };
};


const out = { stamp };
for (const n of [1, 2, 3]) await upload(n);
const q = `select title, status from documents where course_id='${course}' and title like 'chaos-docling-${stamp}-%' order by title`;
for (let i = 0; i < 60; i++) { const r = psql(q).split("\n").filter(Boolean); if (r.length === 3 && r.every((x) => x.endsWith("READY"))) break; await sleep(5); }
out.ready_before_stop = psql(q);
const chunk = (n) => psql(`select replace(c.text, chr(10), ' ') from content_chunks c join documents d on d.id=c.document_id where d.title='chaos-docling-${stamp}-${n}' limit 1`);
run("docker", ["stop", DOCLING]); out.docling_after_stop = run("docker", ["inspect", "-f", "{{.State.Status}}", DOCLING]);
out.chat_during = [];
for (const n of [1, 2, 3]) { const r = await chat(`docling tắt, hỏi nguyên văn đoạn của tài liệu ${n}`, chunk(n)); out.chat_during.push(r); await sleep(15); }
out.docling_still_down = run("docker", ["inspect", "-f", "{{.State.Status}}", DOCLING]);
run("docker", ["start", DOCLING]);
console.log(JSON.stringify(out, null, 1));
