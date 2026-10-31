// Kiểm dữ liệu seed sổ điểm (US-S8-SEED) bằng API thật (tài khoản giảng viên + sinh viên seed).
//   node scripts/check-grade-seed.mjs preview   30/30 dòng của lớp 1 khớp seed/expected_final_grades.csv (HP và điểm chữ)
//   node scripts/check-grade-seed.mjs abc       A 9,5 · A+, B 7,0 · B (giải trình đúng số), C 3,0 · F
//   node scripts/check-grade-seed.mjs counts    số tài liệu / cột / ô điểm; chạy lại seed.mjs, số không đổi (--no-rerun bỏ chạy lại)
//   node scripts/check-grade-seed.mjs lop2      `Chốt điểm` lớp 2 (chưa có công thức) → 409
//   node scripts/check-grade-seed.mjs impact    p95 của `POST …/grade-scheme/impact` (30 lần, lớp 1, 30 sinh viên) ≤ 300 ms (AC4 của US-P6-04)
//   node scripts/check-grade-seed.mjs           cả năm
import { readFileSync } from "node:fs";
import path from "node:path";
import { call, courses, failures, finish, listAll, must, reseed, root, tok } from "./seed-check-lib.mjs";

const modes = process.argv.slice(2).filter((a) => !a.startsWith("--"));
const run = (m) => modes.length === 0 || modes.includes(m);
const noRerun = process.argv.includes("--no-rerun");
const { c1, c2 } = await courses();
const t = await tok("teacher");
const expected = readFileSync(path.join(root, "seed", "expected_final_grades.csv"), "utf8").trim().split("\n").slice(1).map((l) => l.split(",").slice(0, 14));
const IDX = { account: 0, code: 1, hp: 12, letter: 13 };
const comma = (x) => String(Number(x)).replace(".", ","); // 9.5 → "9,5"

async function rows() {
  const out = [];
  let cursor = "";
  do {
    const r = await call("GET", `/courses/${c1}/gradebook?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token: t });
    if (r.status !== 200) throw new Error(`GET gradebook → ${r.status}`);
    out.push(...r.body.rows);
    cursor = r.body.next_cursor ?? "";
  } while (cursor);
  return out;
}

const all = await rows();
const byCode = new Map(all.map((r) => [r.student.student_code, r]));
let summary = "";

if (run("preview")) {
  let ok = 0;
  for (const e of expected) {
    const r = byCode.get(e[IDX.code]);
    if (!r) { failures.push(`${e[IDX.account]}: không có dòng trong sổ`); continue; }
    const hp = r.result?.hp, letter = r.result?.letter;
    if (hp !== undefined && Number(hp) === Number(e[IDX.hp]) && letter === e[IDX.letter]) ok++;
    else failures.push(`${e[IDX.account]} (${e[IDX.code]}): HP ${hp} · ${letter} ≠ ${e[IDX.hp]} · ${e[IDX.letter]}`);
  }
  must(all.length === expected.length, `sổ có ${all.length} dòng, cần ${expected.length}`);
  summary += `${ok}/${expected.length} OK\n`;
}

if (run("abc")) {
  const want = { 20229001: ["9,5", "A+"], 20229002: ["7,0", "B"], 20229003: ["3,0", "F"] };
  for (const [code, [hp, letter]] of Object.entries(want)) {
    const r = byCode.get(code);
    const got = r?.result ? `${comma(r.result.hp)}${/,/.test(hp) && !String(r.result.hp).includes(".") ? ",0" : ""}` : "";
    must(r?.result && Number(r.result.hp) === Number(hp.replace(",", ".")) && r.result.letter === letter, `${code}: ${got} · ${r?.result?.letter} ≠ ${hp} · ${letter}`);
  }
  const b = byCode.get("20229002");
  const ex = await call("GET", `/courses/${c1}/gradebook/students/${b.student.id}/explain`, { token: t });
  must(ex.status === 200, `explain B → ${ex.status}`);
  const line = (label) => ex.body.lines.find((l) => l.label.startsWith(label));
  must(line("Điểm cộng")?.value === "0,5", `giải trình B: điểm cộng ${line("Điểm cộng")?.value} (cần 0,5 = 2 × 0,25)`);
  must(line("Điểm trừ")?.expr.includes("2 − 2") && line("Điểm trừ")?.value === "0", `giải trình B: điểm trừ ${line("Điểm trừ")?.expr} (vắng 2, miễn 2)`);
  must(line("Kiểm tra hằng tuần")?.value === "8", `giải trình B: kiểm tra tuần ${line("Kiểm tra hằng tuần")?.value} (cần 8)`);
  summary += "A B C OK\n";
}

async function counts() {
  const docs = await listAll(`/courses/${c1}/documents`, t);
  const gb = (await call("GET", `/courses/${c1}/gradebook?limit=100`, { token: t })).body;
  const entries = (await rows()).reduce((n, r) => n + Object.keys(r.entries).length, 0);
  const st1 = (await call("GET", `/courses/${c1}/grade-scheme/status`, { token: t })).body;
  const st2 = (await call("GET", `/courses/${c2}/grade-scheme/status`, { token: t })).body;
  must(st1.state === "CONFIRMED", `lớp 761987: ${st1.state}, cần CONFIRMED`);
  must(st2.state === "NO_POLICY", `lớp 761988: ${st2.state}, cần NO_POLICY`);
  const policies = docs.filter((d) => d.type === "COURSE_POLICY");
  must(policies.length === 1 && policies[0].status === "READY", `COURSE_POLICY lớp 1 = ${policies.length}`);
  const school = docs.find((d) => d.title === "Quy chế học vụ");
  must(school?.type === "OTHER" && school.category === "Quy chế trường", `Quyche.pdf: ${school?.type} / ${school?.category}`);
  must(gb.state.status !== "LOCKED", "sổ lớp 1 không được chốt");
  const gb2 = (await call("GET", `/courses/${c2}/gradebook`, { token: t })).body;
  must(gb2.columns.length === 0, "lớp 761988 không có cột điểm");
  return { docs: docs.length, columns: gb.columns.length, entries };
}

if (run("lop2")) {
  const r = await call("POST", `/courses/${c2}/gradebook/finalize`, { token: t, json: {} });
  must(r.status === 409, `finalize lớp 761988 → ${r.status}, cần 409`);
  summary += "lớp 2: chốt điểm 409\n";
}

if (run("impact")) {
  const cur = (await call("GET", `/courses/${c1}/grade-scheme`, { token: t })).body.confirmed;
  const body = { version: cur.version, components: cur.components.map((c) => ({ key: c.key, name: c.name, grp: c.grp, weight: c.weight, items: c.items })), rules: cur.rules.map((r) => ({ kind: r.kind, applicable: r.applicable, params: r.params })) };
  const ms = [];
  for (let i = 0; i < 30; i++) {
    const t0 = performance.now();
    const r = await call("POST", `/courses/${c1}/grade-scheme/impact`, { token: t, json: body });
    ms.push(performance.now() - t0);
    must(r.status === 200, `impact → ${r.status}`);
  }
  ms.sort((a, b) => a - b);
  const p95 = ms[Math.ceil(0.95 * ms.length) - 1];
  must(p95 <= 300, `impact p95 = ${p95.toFixed(0)} ms > 300`);
  summary += `impact p95 ${p95.toFixed(0)} ms (≤ 300)\n`;
}

if (run("counts")) {
  const a = await counts();
  must(a.columns >= 3, `cột điểm lớp 1 = ${a.columns}`);
  let note = "bỏ qua (--no-rerun)";
  if (!noRerun) {
    must(reseed() === 0, "chạy lại seed.mjs lỗi");
    const b = await counts();
    must(JSON.stringify(a) === JSON.stringify(b), `seed lần hai đổi số: ${JSON.stringify(a)} → ${JSON.stringify(b)}`);
    note = "không đổi sau lần chạy hai";
  }
  summary += `counts ${JSON.stringify(a)} — ${note}\n`;
}

finish(summary.trim());
