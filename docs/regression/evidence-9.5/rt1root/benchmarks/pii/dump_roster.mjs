// Dựng benchmarks/pii/roster_seed.json = roster lớp 1 (30 sinh viên) ĐÚNG như scripts/seed.mjs sinh ra (cùng hạt giống SEED_RNG): trích đoạn sinh tên / mã của seed rồi chạy.
// Dùng khi dựng bộ dữ liệu E1; chạy lại khi seed.mjs đổi cách sinh roster:  node benchmarks/pii/dump_roster.mjs > benchmarks/pii/roster_seed.json
import { readFileSync } from "node:fs";
const src = readFileSync(new URL("../../scripts/seed.mjs", import.meta.url), "utf8");
const a = src.indexOf("function mulberry32");
const b = src.indexOf("const A = accounts.get");
if (a < 0 || b < 0) throw new Error("scripts/seed.mjs đổi cấu trúc: sửa dump_roster.mjs");
const RNG_SEED = Number(process.env.SEED_RNG || "20261029");
const DOMAIN = "@edupilot.local";
const out = new Function("RNG_SEED", "DOMAIN", `${src.slice(a, b)}; return roster1.map((k, i) => ({ n: i + 1, local: k, ...accounts.get(k) }));`)(RNG_SEED, DOMAIN);
console.log(JSON.stringify(out, null, 1));
