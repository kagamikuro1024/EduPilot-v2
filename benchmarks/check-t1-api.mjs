#!/usr/bin/env node
// US-NFR-03 AC4: sau seed-t1 + ANALYZE, 6 trang chính bằng STUDENT / TEACHER không 5xx; SV lớp X không đọc được lớp Y (≥ 20 cặp ngẫu nhiên → 403/404).
//   BASE=https://localhost:843 SEED_DEFAULT_PASSWORD=… node benchmarks/check-t1-api.mjs   (stack ep-* only)
import { mkdirSync, writeFileSync } from "node:fs";
import { meta } from "./lib/meta.mjs";
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
const BASE = process.env.BASE ?? "";
if (!/^https:\/\/localhost:(?!443$)\d+$/.test(BASE)) { console.error("Từ chối: BASE phải là cổng HTTPS của stack ep-* (≠ 443)."); process.exit(1); }
const PW = process.env.SEED_DEFAULT_PASSWORD;
if (!PW) { console.error("Thiếu SEED_DEFAULT_PASSWORD."); process.exit(1); }
const api = async (path, tok, init = {}) => fetch(`${BASE}/api/v1${path}`, { ...init, headers: { "Content-Type": "application/json", ...(tok ? { Authorization: `Bearer ${tok}` } : {}) } });
const login = async (u) => { const r = await api("/auth/login", null, { method: "POST", body: JSON.stringify({ email: `${u}@edupilot.local`, password: PW }) }); if (!r.ok) throw new Error(`login ${u}: ${r.status}`); return (await r.json()).access_token; };
const rows = []; const rec = (who, path, status) => rows.push({ who, path, status });
const get = async (who, tok, path) => { const r = await api(path, tok); rec(who, path.replace(/[0-9a-f-]{36}/g, ":id"), r.status); return r; };

const gv = await login("t1gv1");
const gvCourses = (await (await get("GV", gv, "/me/courses")).json()).items.map((i) => i.course.id);
const c = gvCourses[0];
await get("GV", gv, "/me/today"); await get("GV", gv, `/courses/${c}/today`);
const list = await (await get("GV", gv, `/courses/${c}/students?limit=100`)).json();
await get("GV", gv, `/courses/${c}/gradebook`);
await get("GV", gv, `/courses/${c}/students/${list.items[0].student.id}/profile`);
const mine = [];
for (const n of [1, 2, 51, 101, 201, 501, 801, 1000]) {
  const u = `t1sv${String(n).padStart(4, "0")}`; const tok = await login(u);
  const cs = (await (await get("SV", tok, "/me/courses")).json()).items.map((i) => i.course.id); mine.push({ tok, own: cs[0] });
  await get("SV", tok, "/me/today"); for (const p of ["me/grades", "me/attendance?limit=100", "threads?limit=30", "library"]) await get("SV", tok, `/courses/${cs[0]}/${p}`);
}
const allCourses = [...new Set(mine.map((m) => m.own))];
let pairs = 0, leaks = 0;
for (let i = 0; pairs < 24; i++) {
  const m = mine[i % mine.length]; const other = allCourses[(i + 1 + Math.floor(i / mine.length)) % allCourses.length];
  if (other === m.own) continue;
  for (const p of ["me/grades", "threads?limit=30", "library"]) { const r = await get("SV-x", m.tok, `/courses/${other}/${p}`); if (![403, 404].includes(r.status)) leaks++; }
  pairs++;
}
const e5xx = rows.filter((r) => r.status >= 500).length, own = rows.filter((r) => r.who !== "SV-x" && r.status >= 400);
const by = {}; for (const r of rows) { const k = `${r.who} ${r.path} ${r.status}`; by[k] = (by[k] ?? 0) + 1; }
console.log(Object.entries(by).map(([k, n]) => `${n}× ${k}`).join("\n"));
console.log(`\n5xx: ${e5xx} · lỗi 4xx ở đường hợp lệ: ${own.length} · cặp chéo lớp: ${pairs} · rò: ${leaks}`);
mkdirSync("benchmarks/reports", { recursive: true });
writeFileSync("benchmarks/reports/t1-api-check.json", JSON.stringify({ meta: meta({ scenario: "t1-api-check", base: BASE }), pairs, leaks, errors_5xx: e5xx, errors_4xx_valid: own, summary: by }, null, 2));
process.exit(e5xx || leaks || own.length ? 1 : 0);
