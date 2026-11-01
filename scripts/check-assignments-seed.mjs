// Kiểm dữ liệu seed Bài tập (US-S9-SEED) bằng API thật (tài khoản giảng viên + sinh viên seed). Thứ tự an toàn trên stack mới seed:
//   node scripts/check-assignments-seed.mjs bt03     26 bản ghi bài nộp của Bài tập 03 khớp từng dòng seed-plan.csv (kênh, muộn, khớp / chưa khớp)
//   node scripts/check-assignments-seed.mjs counts   số bản ghi; chạy lại seed.mjs, số không đổi (--no-rerun bỏ chạy lại)
//   node scripts/check-assignments-seed.mjs grade    `Chấm tất cả` (provider giả) → 22/22 khớp expected_grading.json
//   node scripts/check-assignments-seed.mjs demo5    DEMO bước 5: Sinh viên B sửa tiêu chí 2, duyệt, công bố → B thấy 7,50
//   node scripts/check-assignments-seed.mjs teams    `Đồng bộ từ Teams` kịch bản happy → 24 bài khớp + 1 chưa khớp
//   node scripts/check-assignments-seed.mjs v2       quy chế bản 2 → gán sv29, sv30, bỏ qua thư lạ → gắn item → công bố → xác nhận → 30/30 đúng expected_final_grades_v2.csv
// `grade`, `demo5`, `v2` ĐỔI dữ liệu (chấm, công bố, xác nhận công thức): chạy trên stack seed sạch, đúng thứ tự trên (gate-p7.sh làm vậy).
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";
import { call, courses, failures, finish, listAll, must, reseed, root, sleep, tok } from "./seed-check-lib.mjs";
import { expectedGrading, loadPlan, TEAMS_TITLE_04 } from "./assignment-seed-data.mjs";
import { accountCodes } from "./assignment-accounts.mjs";

const modes = process.argv.slice(2).filter((a) => !a.startsWith("--"));
const run = (m) => modes.includes(m);
const MOCK_GRAPH = (process.env.MOCK_GRAPH_URL || "http://localhost:8090").replace(/\/$/, "");
const { c1 } = await courses();
const t = await tok("teacher");
const plan = loadPlan(root);
const code = accountCodes();
const matchedPlan = plan.filter((r) => r.match === "MATCHED");
let summary = "";

async function assignments() {
  const all = await listAll(`/courses/${c1}/assignments`, t);
  return { bt03: all.find((a) => a.submission_code === "BT03"), bt04: all.find((a) => a.submission_code === "BT04"), all };
}
const { bt03, bt04, all: allAsg } = await assignments();
if (!bt03 || !bt04) throw new Error("Chưa có Bài tập 03 / 04 (chạy `node scripts/seed.mjs` trước).");
const queue = async (a, status = "NONE") => listAll(`/courses/${c1}/submissions?assignment_id=${a.id}&status=${status}`, t);
const jobDone = async (id, what) => {
  for (let i = 0; i < 180; i++) {
    const j = await call("GET", `/jobs/${id}`, { token: t });
    if (j.body.status === "SUCCEEDED") return j.body;
    if (j.body.status === "FAILED") throw new Error(`${what}: việc lỗi ${JSON.stringify(j.body.error)}`);
    await sleep(2000);
  }
  throw new Error(`${what}: quá 6 phút`);
};
const lateOf = (h) => (h > 0 ? Math.min(Math.ceil(h / 24), 2) : 0);
const rowOf = (rows, account) => rows.find((r) => r.student.student_code === code.get(account));

async function submissionsAll() {
  const matched = await queue(bt03, "NONE");
  const rest = [];
  for (const st of ["RUNNING", "DRAFT", "REVIEWED", "PUBLISHED", "FAILED"]) rest.push(...(await queue(bt03, st)));
  const seen = new Set();
  return [...matched, ...rest].filter((r) => (seen.has(r.id) ? false : seen.add(r.id)));
}

if (run("bt03")) {
  const rows = await submissionsAll();
  const unmatched = await listAll(`/courses/${c1}/submissions?match_status=UNMATCHED`, t);
  const inbound = await listAll(`/courses/${c1}/mail-inbound`, t);
  let ok = 0;
  for (const r of matchedPlan) {
    const row = rowOf(rows, r.account);
    const late = r.channel === "UPLOAD" ? 0 : lateOf(r.hours);
    const versions = (await call("GET", `/courses/${c1}/assignments/${bt03.id}/versions?student_id=${row?.student.id}`, { token: t })).body?.versions;
    const wantV = r.sample2 ? 2 : 1;
    if (!row) failures.push(`${r.account}: không có bài nộp MATCHED`);
    else if (row.source !== r.channel) failures.push(`${r.account}: kênh ${row.source} ≠ ${r.channel}`);
    else if (row.late_days !== late) failures.push(`${r.account}: muộn ${row.late_days} ngày ≠ ${late}`);
    else if (versions !== wantV || row.version !== wantV) failures.push(`${r.account}: phiên bản ${row.version} (có ${versions}) ≠ ${wantV}`);
    else ok++;
  }
  must(ok === matchedPlan.length, `chỉ ${ok}/${matchedPlan.length} dòng seed-plan khớp`);
  const nonSubmit = plan.filter((r) => !r.channel && r.account !== "(ngoài lớp)");
  must(nonSubmit.length === 6, `seed-plan có ${nonSubmit.length} sinh viên không nộp, cần 6`);
  for (const r of nonSubmit) must(!rowOf(rows, r.account), `${r.account} không được có bài nộp`);
  // 22 khớp + 3 chưa khớp (sv30 tệp ZIP, sv29 thư sai mẫu, thư email lạ)
  const un = unmatched.map((u) => `${u.source}:${u.reason}`).sort();
  must(unmatched.length === 3, `chưa khớp: ${unmatched.length} (${un.join(", ")}), cần 3`);
  must(unmatched.filter((u) => u.source === "UPLOAD").length === 1 && unmatched.filter((u) => u.source === "IMAP").length === 2, `chưa khớp theo kênh sai: ${un.join(", ")}`);
  must(unmatched.filter((u) => u.assignment_id).length === 2, "đúng 2 bài chưa khớp biết bài (ZIP + thư MSSV ngoài lớp); thư sai mẫu chưa biết bài");
  must(inbound.length === 8 && inbound.every((m) => ["ACCEPTED", "IGNORED"].includes(m.status) || m.submission_id), `mail_inbound: ${inbound.length}, cần 8`);
  must(allAsg.length === 2 && bt03.published_at && bt04.published_at, "cần đúng 2 bài tập đã công bố");
  must(bt03.counts.graded === 0, "seed không được chạy chấm (chưa có grading_results)");
  const bs = new Map();
  for (const r of matchedPlan) bs.set(r.channel, (bs.get(r.channel) ?? 0) + 1 + (r.sample2 ? 1 : 0));
  summary += `OK ${ok}/${matchedPlan.length} khớp ngay + 3 chưa khớp + 6 không nộp; bản ghi: IN_APP ${bs.get("IN_APP")} · IMAP ${bs.get("IMAP") + 2} (kèm sv29 + thư lạ) · UPLOAD ${bs.get("UPLOAD") + 1} (kèm sv30)\n`;
}

async function counts() {
  const rows = await submissionsAll();
  const un = await listAll(`/courses/${c1}/submissions?match_status=UNMATCHED`, t);
  const inbound = await listAll(`/courses/${c1}/mail-inbound`, t);
  const docs = await listAll(`/courses/${c1}/documents`, t);
  const imap = (await call("GET", `/courses/${c1}/integrations/imap`, { token: t })).body;
  const teams = (await call("GET", `/courses/${c1}/integrations/teams`, { token: t })).body;
  let versions = 0;
  for (const r of rows.filter((x) => x.source === "IN_APP")) versions += (await call("GET", `/courses/${c1}/assignments/${bt03.id}/versions?student_id=${r.student.id}`, { token: t })).body.versions;
  return { assignments: allAsg.length, matched: rows.length, unmatched: un.length, inApp: versions, inbound: inbound.length, documents: docs.length, imap: !!imap.configured, teams: !!teams.configured };
}

if (run("counts")) {
  const before = await counts();
  must(before.assignments === 2 && before.matched === 22 && before.unmatched === 3 && before.inApp === 13 && before.inbound === 8, `số bản ghi sai: ${JSON.stringify(before)}`);
  must(before.imap && before.teams, "thiếu thẻ IMAP / Teams của lớp 1");
  if (!process.argv.includes("--no-rerun")) {
    const rc = reseed();
    must(rc === 0, `chạy lại seed.mjs thoát ${rc}`);
    const after = await counts();
    must(JSON.stringify(before) === JSON.stringify(after), `số đổi sau khi chạy lại seed: ${JSON.stringify(before)} → ${JSON.stringify(after)}`);
  }
  summary += `counts ${JSON.stringify(before)} (không đổi sau seed lần hai)\n`;
}

const flagsOf = (r) => [...(r.flags ?? [])].sort().join(",");
if (run("grade")) {
  const g = await call("POST", `/courses/${c1}/assignments/${bt03.id}/grade-all`, { token: t, json: {} });
  if (g.status === 202) await jobDone(g.body.job_id, "grade-all");
  for (let i = 0; i < 180 && (await queue(bt03, "RUNNING")).length > 0; i++) await sleep(2000); // lô khác (k6, lần chạy trước) còn đang chấm: chờ xong
  const exp = new Map(expectedGrading(root).map((e) => [e.id, e]));
  const rows = [];
  for (const st of ["DRAFT", "REVIEWED", "PUBLISHED"]) rows.push(...(await queue(bt03, st)));
  must(rows.length === 22, `đã chấm ${rows.length}/22 bài`);
  let ok = 0;
  for (const r of matchedPlan) {
    const sample = r.sample2 || r.sample1, e = exp.get(sample).expected;
    const row = rowOf(rows, r.account);
    if (!row?.result) { failures.push(`${r.account}: chưa có kết quả chấm`); continue; }
    const v = (await call("GET", `/courses/${c1}/grading-results/${row.result.id}`, { token: t })).body;
    const got = Object.fromEntries(v.criteria.map((c) => [c.id, c.score]));
    const bad = [];
    for (const [id, s] of Object.entries(e.criteria_scores)) if (Number(got[id]) !== Number(s)) bad.push(`${id} ${got[id]}≠${s}`);
    if (Number(v.result.ai_total_before_late) !== Number(e.total_before_late)) bad.push(`tổng trước muộn ${v.result.ai_total_before_late}≠${e.total_before_late}`);
    const wantTotal = Math.max(0, Number(e.total_before_late) - 0.5 * (r.channel === "UPLOAD" ? 0 : lateOf(r.hours))); // muộn theo giờ của seed-plan (sv08: 2 ngày → −1,0); mẫu S7 đã gồm muộn của B
    if (Number(v.result.total) !== wantTotal) bad.push(`tổng ${v.result.total}≠${wantTotal}`);
    if (flagsOf(row.result) !== [...e.flags].sort().join(",")) bad.push(`cờ [${flagsOf(row.result)}]≠[${e.flags}]`);
    if (!!row.result.needs_review !== e.needs_review) bad.push(`needs_review ${row.result.needs_review}≠${e.needs_review}`);
    if (bad.length) failures.push(`${r.account} (${sample}): ${bad.join("; ")}`);
    else ok++;
  }
  must(ok === 22, `${ok}/22 khớp expected_grading.json`);
  summary += `${ok}/22 khớp expected_grading.json\n`;
}

async function sv(account) {
  const r = await call("POST", "/auth/login", { json: { email: `${account}@edupilot.local`, password: process.env.SEED_DEFAULT_PASSWORD || (await import("./seed-check-lib.mjs")).PASSWORD } });
  return r.body.access_token;
}

if (run("demo5")) {
  const b = rowOf(await queue(bt03, "DRAFT"), "sv.kha") ?? rowOf(await queue(bt03, "REVIEWED"), "sv.kha");
  const sb = await sv("sv.kha");
  const mine = async () => (await call("GET", `/courses/${c1}/assignments/${bt03.id}/mine`, { token: sb })).body;
  const before = await mine();
  if (before.state !== "GRADED") {
    must(["SUBMITTED", "GRADING"].includes(before.state) && before.result === null && before.submission?.late_days === 1, `B trước công bố: ${before.state}, muộn ${before.submission?.late_days} (cần SUBMITTED / GRADING, 1 ngày, không điểm nháp)`);
    must(!JSON.stringify(before).includes("6.50") && !JSON.stringify(before).includes("CANARY-DRAFT"), "B thấy điểm nháp");
    const v0 = (await call("GET", `/courses/${c1}/grading-results/${b.result.id}`, { token: t })).body;
    must(v0.flags.some((f) => (f.code ?? f) === "NEEDS_REVIEW") && v0.pass_detail != null, "bài của B thiếu cờ 'cần xem kỹ' / chi tiết hai lượt");
    const e = await call("PUT", `/courses/${c1}/grading-results/${b.result.id}`, { token: t, json: { version: v0.result.version, criteria: [{ id: "c2", score: "2.00", comment: "Chuỗi tấn công đã đủ các bước then chốt, có căn cứ kỹ thuật." }] } });
    must(e.status === 200 && e.body.result.total === "7.50", `sau sửa tiêu chí 2: ${e.body?.result?.total}, cần 7.50 (8,00 − 0,50)`);
    const rv = await call("POST", `/courses/${c1}/grading-results/${b.result.id}/review`, { token: t, json: { version: e.body.result.version } });
    must(rv.status === 200, `duyệt bài B → ${rv.status}`);
    const pub = await call("POST", `/courses/${c1}/grading-results/publish`, { token: t, json: { result_ids: [b.result.id] } });
    must(pub.status === 200 && pub.body.published === 1, `công bố bài B → ${pub.status}`);
  }
  const after = await mine();
  must(after.state === "GRADED" && after.result?.total === "7.50", `B sau công bố: ${after.state} ${after.result?.total}`);
  must(after.result?.criteria?.every((c) => c.comment && Array.isArray(c.evidence)), "B thiếu nhận xét / đoạn trích theo tiêu chí");
  summary += `B ${after.result?.total}\n`;
}

if (run("teams")) {
  await fetch(`${MOCK_GRAPH}/__mock/reset`, { method: "POST" });
  await fetch(`${MOCK_GRAPH}/__mock/scenario`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ scenario: "happy" }) });
  const s = await call("POST", `/courses/${c1}/assignments/${bt04.id}/teams/sync`, { token: t, key: `chk-teams-${Date.now()}` });
  const j = await jobDone(s.body.job_id, "đồng bộ Teams");
  const rows = await queue(bt04);
  const un = (await listAll(`/courses/${c1}/submissions?match_status=UNMATCHED`, t)).filter((u) => u.assignment_id === bt04.id);
  must(rows.length === 24 && rows.every((r) => r.source === "TEAMS"), `Teams: ${rows.length} bài khớp (cần 24, kênh TEAMS) · việc ${JSON.stringify(j.result)}`);
  must(un.length === 1, `Teams: ${un.length} chưa khớp, cần 1`);
  summary += `teams OK: 24 khớp + 1 chưa khớp (${TEAMS_TITLE_04.slice(0, 10)}…)\n`;
}

if (run("v2")) summary += await (await import("./check-assignments-v2.mjs")).runV2({ c1, t, bt03, plan, code, queue, jobDone, submissionsAll, rowOf, sv });
finish(summary);
