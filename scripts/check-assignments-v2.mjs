// Mode `v2` của check-assignments-seed.mjs (US-S9-SEED AC4): quy chế lớp 1 bản 2 → gán sv29, sv30 + bỏ qua thư lạ → gắn item "Bài tập" → công bố → xác nhận → 30/30 đúng expected_final_grades_v2.csv.
// ĐỔI dữ liệu (xác nhận công thức bản 2): chạy trên stack seed sạch, sau `grade` và `demo5`.
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import path from "node:path";
import { call, failures, listAll, must, root, sleep } from "./seed-check-lib.mjs";

export async function runV2({ c1, t, bt03, plan, code, queue, jobDone, submissionsAll, rowOf }) {
  const pdf = readFileSync(path.join(root, "seed", "demo", "quy-che-lop1-v2.pdf"));
  const st0 = (await call("GET", `/courses/${c1}/grade-scheme/status`, { token: t })).body;
  const gb = async () => {
    const out = [];
    let cursor = "";
    do {
      const r = await call("GET", `/courses/${c1}/gradebook/preview?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token: t });
      if (r.status !== 200) throw new Error(`GET gradebook/preview → ${r.status}`);
      out.push(...r.body.items);
      cursor = r.body.next_cursor ?? "";
    } while (cursor);
    return out;
  };
  // 1) công bố mọi bài đã chấm của 22 sinh viên (duyệt từng bài rồi công bố hàng loạt)
  for (const row of await queue(bt03, "DRAFT")) {
    const v = (await call("GET", `/courses/${c1}/grading-results/${row.result.id}`, { token: t })).body;
    const r = await call("POST", `/courses/${c1}/grading-results/${row.result.id}/review`, { token: t, json: { version: v.result.version } });
    must(r.status === 200, `duyệt ${row.student.student_code} → ${r.status}`);
  }
  await call("POST", `/courses/${c1}/grading-results/publish`, { token: t, json: { assignment_id: bt03.id, scope: "ALL_REVIEWED" } });
  // 2) tải bản 2, trích (provider giả phát extraction-lop1-v2.json)
  const sha = createHash("sha256").update(pdf).digest("hex");
  const docs = await listAll(`/courses/${c1}/documents`, t);
  let doc = docs.find((d) => d.title === "Quy chế học phần (bản 2)");
  if (!doc) {
    const pre = await call("POST", `/courses/${c1}/uploads/presign`, { token: t, json: { purpose: "document", filename: "quy-che-lop1-v2.pdf", mime_type: "application/pdf", size_bytes: pdf.length, sha256: sha } });
    const put = await fetch(pre.body.url, { method: pre.body.method, headers: pre.body.headers, body: pdf });
    must(put.ok, `PUT bản 2 → ${put.status}`);
    const done = await call("POST", `/courses/${c1}/uploads/complete`, { token: t, json: { upload_id: pre.body.upload_id, title: "Quy chế học phần (bản 2)", type: "COURSE_POLICY" }, key: `chk-v2-doc-${sha.slice(0, 8)}` });
    await jobDone(done.body.job_id, "đọc quy chế bản 2");
    doc = (await listAll(`/courses/${c1}/documents`, t)).find((d) => d.title === "Quy chế học phần (bản 2)");
  }
  must(doc?.status === "READY", `bản 2: ${doc?.status}`);
  const ex = await call("POST", `/courses/${c1}/grade-scheme/extract`, { token: t, json: { document_id: doc.id, replace_draft: true }, key: `chk-v2-extract-${doc.id.slice(0, 8)}-${Date.now()}` });
  await jobDone(ex.body.job_id, "trích công thức bản 2");
  const draft = (await call("GET", `/courses/${c1}/grade-scheme`, { token: t })).body.draft;
  const comps = draft.components.map((c) => `${c.key}:${Number(c.weight)}`).join(" ");
  must(comps === "midterm:10 weekly:10 assignment:20 final:60", `bản nháp: ${comps}`);
  must(draft.unclear_items.length === 0, `bản nháp còn ${draft.unclear_items.length} mục chưa rõ`);
  must(draft.rules.some((r) => r.kind === "MISSING_SCORE" && JSON.stringify(r.params).includes("assignment")), "luật MISSING_SCORE chưa có `assignment`");
  // 3) gắn item "Bài tập" vào BT03 (+ item kiểm tra tuần vào bài thi như bản 1), lưu, xác nhận
  const confirmed = (await call("GET", `/courses/${c1}/grade-scheme`, { token: t })).body.confirmed;
  const weeklyExam = confirmed.components.find((c) => c.key === "weekly").items[0].exam_id;
  const put = await call("PUT", `/courses/${c1}/grade-scheme`, { token: t, json: {
    version: draft.version,
    components: draft.components.map((c) => ({ key: c.key, name: c.name, grp: c.grp, weight: c.weight, items: c.items.map((i) => (c.key === "weekly" ? { ...i, source: "EXAM", exam_id: weeklyExam } : c.key === "assignment" ? { ...i, source: "ASSIGNMENT", assignment_id: bt03.id } : i)) })),
    rules: draft.rules.map((r) => ({ kind: r.kind, applicable: r.applicable, params: r.params })),
  } });
  must(put.status === 200, `lưu bản nháp có item Bài tập → ${put.status} ${JSON.stringify(put.body?.error ?? put.body?.errors ?? "")}`);
  const cf = await call("POST", `/courses/${c1}/grade-scheme/confirm`, { token: t, json: { version: put.body.version ?? draft.version + 1 }, key: `chk-v2-confirm-${doc.id.slice(0, 8)}-${Date.now()}` });
  must(cf.status === 200, `xác nhận bản 2 → ${cf.status} ${JSON.stringify(cf.body?.error ?? cf.body)} (đã lưu ${JSON.stringify(put.body?.version)})`);
  // 4) còn bài chưa khớp: sinh viên không nộp INCOMPLETE, Chốt điểm bị chặn
  let rows = await gb();
  const nonSubmit = plan.filter((r) => !r.channel && r.account !== "(ngoài lớp)").map((r) => code.get(r.account));
  const inc = rows.filter((r) => r.status === "INCOMPLETE").map((r) => r.student_code);
  must(nonSubmit.every((c) => inc.includes(c)), `6 sinh viên không nộp phải INCOMPLETE khi còn bài chưa khớp; INCOMPLETE: ${inc.length}`);
  const guard = nonSubmit.every((c) => inc.includes(c)); // chỉ thử Chốt điểm khi đã thấy INCOMPLETE: nếu sai, Chốt điểm sẽ chốt thật
  const fin = !guard ? { status: 0, body: null } : await call("POST", `/courses/${c1}/gradebook/finalize`, { token: t, key: `chk-v2-finalize-${Date.now()}`, json: {} });
  must(fin.status === 409 && fin.body?.code === "GRADES_INCOMPLETE", `Chốt điểm khi còn bài chưa khớp → ${fin.status} ${fin.body?.code}`);
  // 5) giảng viên gán tay sv29, sv30; bỏ qua thư email lạ
  const un = await listAll(`/courses/${c1}/submissions?match_status=UNMATCHED`, t);
  const members = await listAll(`/courses/${c1}/members`, t);
  const sidOf = (account) => members.find((m) => m.student_code === code.get(account))?.user_id;
  for (const u of un) {
    const asSv29 = u.source === "IMAP" && !u.assignment_id, asSv30 = u.source === "UPLOAD";
    if (asSv29 || asSv30) {
      const uid = sidOf(asSv29 ? "sv29" : "sv30");
      const m = await call("PUT", `/courses/${c1}/submissions/${u.id}/match`, { token: t, json: { student_id: uid, assignment_id: bt03.id, version: u.version } });
      must(m.status === 200, `gán tay ${asSv29 ? "sv29" : "sv30"} → ${m.status} ${JSON.stringify(m.body)}`);
    } else {
      const d = await call("POST", `/courses/${c1}/submissions/${u.id}/dismiss`, { token: t, json: { reason: "Thư từ email lạ, MSSV ngoài lớp", version: u.version }, key: `chk-v2-dismiss-${u.id}` });
      must(d.status === 200 || d.status === 204, `bỏ qua thư lạ → ${d.status}`);
    }
  }
  // 6) chấm 2 bài mới gán, duyệt, công bố
  const g = await call("POST", `/courses/${c1}/assignments/${bt03.id}/grade-all`, { token: t, json: {} });
  if (g.status === 202) await jobDone(g.body.job_id, "chấm bài mới gán");
  for (const row of await queue(bt03, "DRAFT")) {
    const v = (await call("GET", `/courses/${c1}/grading-results/${row.result.id}`, { token: t })).body;
    await call("POST", `/courses/${c1}/grading-results/${row.result.id}/review`, { token: t, json: { version: v.result.version } });
  }
  const pub = await call("POST", `/courses/${c1}/grading-results/publish`, { token: t, json: { assignment_id: bt03.id, scope: "ALL_REVIEWED" } });
  must(pub.status === 200, `công bố → ${pub.status}`);
  const stillOpen = (await submissionsAll()).filter((r) => r.result?.status !== "PUBLISHED").length;
  must(stillOpen === 0, `còn ${stillOpen} bài chưa công bố`);
  // 7) 30/30 đúng expected_final_grades_v2.csv
  await sleep(500);
  rows = await gb();
  const want = readFileSync(path.join(root, "seed", "expected_final_grades_v2.csv"), "utf8").trim().split("\n").slice(1).map((l) => l.split(","));
  let ok = 0;
  for (const w of want) {
    const r = rows.find((x) => x.student_code === w[1]);
    if (r && r.status === "COMPLETE" && Number(r.hp) === Number(w[9]) && r.letter === w[10] && (w[11] === "OK" ? r.verdict === "OK" : String(r.verdict).startsWith(w[11].split(":")[0]))) ok++;
    else failures.push(`${w[0]} (${w[1]}): ${r?.status} ${r?.hp} ${r?.letter} ${r?.verdict} ≠ ${w[9]} ${w[10]} ${w[11]}`);
  }
  must(ok === want.length && rows.length === want.length, `${ok}/${want.length} đúng expected_final_grades_v2.csv (sổ ${rows.length} dòng)`);
  return `${ok}/${want.length} OK (bản 2)\n`;
}
