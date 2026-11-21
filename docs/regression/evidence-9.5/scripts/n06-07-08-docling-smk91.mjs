#!/usr/bin/env node
// TC-N06-07 + TC-N06-08 (SmkS4, 2026-11-21): stack smk91 (qc2 sở hữu, profile ingest). Tải lên 3 tài liệu → dừng smk91-docling-1 120 s → chạy lại.
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
const chat = async (label) => {
  const sid = (await (await api("/chat/sessions", SV, { method: "POST", body: JSON.stringify({ course_id: course }), headers: { "Idempotency-Key": crypto.randomUUID() } })).json()).id;
  const t = Date.now();
  const r = await api(`/chat/sessions/${sid}/messages`, SV, { method: "POST", body: JSON.stringify({ content: "Điểm quá trình chiếm bao nhiêu phần trăm điểm học phần?" }), headers: { "Idempotency-Key": crypto.randomUUID(), accept: "text/event-stream" } });
  const body = await r.text();
  const done = body.split("\n").filter((l) => l.startsWith("data:") && /"citations"/.test(l)).pop() ?? "";
  let cites = -1, degraded = null;
  try { const j = JSON.parse(done.slice(5)); cites = j.citations?.length ?? 0; degraded = j.degraded; } catch { /* không có done */ }
  return { label, status: r.status, seconds: +((Date.now() - t) / 1000).toFixed(1), citations: cites, degraded, has_done: /event: done/.test(body), error: (body.match(/"code":"([A-Z_]+)"/) ?? [])[1] ?? null };
};

const out = { stamp, course, docling_status_before: run("docker", ["inspect", "-f", "{{.State.Status}}", DOCLING]) };
out.chat_before = await chat("trước khi tắt docling");
const t0 = new Date().toISOString();
for (const n of [1, 2, 3]) await upload(n);
run("docker", ["stop", DOCLING]);
const down = Date.now();
out.chat_during = [];
for (let i = 0; i < 3; i++) { out.chat_during.push(await chat(`docling tắt, lần ${i + 1}`)); await sleep(20); }
const wait = 120 - (Date.now() - down) / 1000;
if (wait > 0) await sleep(wait);
out.docling_down_seconds = Math.round((Date.now() - down) / 1000);
const q = `select title, status, coalesce(error,'') from documents where course_id='${course}' and title like 'chaos-docling-${stamp}-%' order by title`;
out.rows_at_restart = psql(q);
run("docker", ["start", DOCLING]);
const back = Date.now();
let rows = [];
while (Date.now() - back < 660_000) {
  rows = psql(q).split("\n").filter(Boolean).map((l) => l.split("|"));
  if (rows.length === 3 && rows.every((r) => ["READY", "FAILED"].includes(r[1]))) break;
  await sleep(5);
}
out.seconds_to_terminal_after_up = Math.round((Date.now() - back) / 1000);
out.rows_final = rows;
out.stuck_processing_gt_10min = rows.filter((r) => !["READY", "FAILED"].includes(r[1])).length;
out.status_counts = psql(`select status, count(*) from documents where course_id='${course}' and title like 'chaos-docling-${stamp}-%' group by 1`);
out.chat_after = await chat("sau khi docling lên");
out.started_at = t0;
console.log(JSON.stringify(out, null, 1));
