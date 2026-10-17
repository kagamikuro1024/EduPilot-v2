// Kế hoạch điểm danh / phát biểu của lớp 1 (761987) cho US-S7-SEED (FEAT-crm-360 SRS 4.10.1). Hàm thuần, không gọi mạng: `scripts/seed.mjs` dùng để gọi API,
// `backend-go/internal/seedtool` kiểm bằng `seed/crm_plan.json` (sinh bởi `node scripts/crm-plan.mjs --json > seed/crm_plan.json`, CI so khớp).
// Buổi 1–9 một buổi mỗi tuần (buổi 9 = tuần gần nhất đã trọn). Số liệu của A, B, C và vắng buổi 8 / 9 là CỐ ĐỊNH; phần còn lại rải theo SEED_RNG.
import { fileURLToPath } from "node:url";

function mulberry32(a) {
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export const SESSIONS = 9;
const sv = (n) => `sv${String(n).padStart(2, "0")}`;
export const ROSTER1 = ["sv.gioi", "sv.kha", "sv.nguyco", ...Array.from({ length: 27 }, (_, i) => sv(i + 4))];

/** Vắng / muộn cố định (số buổi 1-based). Người không có ở đây: có mặt, trừ phần rải ngẫu nhiên. */
const FIXED = {
  "sv.gioi": { absent: [], late: [] },
  "sv.kha": { absent: [3, 6], late: [4] }, // vắng 2, muộn 1
  "sv.nguyco": { absent: [2, 5, 7, 8, 9], late: [3] }, // vắng 5, muộn 1
  sv12: { absent: [9], late: [] }, // vắng chỉ buổi 9 (+ học < 60 phút tuần 9) → WATCH
  sv19: { absent: [9], late: [] },
};
const SPEAK_FIXED = { "sv.gioi": 12, "sv.kha": 2, "sv.nguyco": 1 };

export function crmPlan(rngSeed = 20261029) {
  const rand = mulberry32((Number(rngSeed) ^ 0xc4d1) >>> 0);
  const free = ROSTER1.filter((k) => !(k in FIXED)); // 25 người rải ngẫu nhiên
  const shuffled = [...free];
  for (let i = shuffled.length - 1; i > 0; i--) { // Fisher–Yates: kết quả không phụ thuộc cách engine sắp xếp
    const j = Math.floor(rand() * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
  }
  const absentExtra = new Set(shuffled.slice(0, 12)); // 12 người vắng đúng một buổi trong 1–7 → ABSENT cả lớp = 9 + 12 = 21
  const lateExtra = new Set(shuffled.slice(8, 14)); // 6 người muộn một buổi trong 1–7 (một số trùng người vắng nhưng khác buổi)
  const attendance = {};
  for (const k of ROSTER1) {
    const fx = FIXED[k] ?? { absent: [], late: [] };
    const row = Array.from({ length: SESSIONS }, (_, i) => (fx.absent.includes(i + 1) ? "ABSENT" : fx.late.includes(i + 1) ? "LATE" : "PRESENT"));
    if (absentExtra.has(k)) row[Math.floor(rand() * 7)] = "ABSENT";
    if (lateExtra.has(k)) {
      const open = row.slice(0, 7).map((s, i) => (s === "PRESENT" ? i : -1)).filter((i) => i >= 0);
      if (open.length) row[open[Math.floor(rand() * open.length)]] = "LATE";
    }
    attendance[k] = row;
  }
  const speak = {};
  for (const k of ROSTER1) speak[k] = k in SPEAK_FIXED ? SPEAK_FIXED[k] : Math.floor(rand() * 4); // 0–3, A nhiều nhất
  const count = (k, s) => attendance[k].filter((x) => x === s).length;
  const totals = ROSTER1.reduce((a, k) => ({ absent: a.absent + count(k, "ABSENT"), speak: a.speak + speak[k] }), { absent: 0, speak: 0 });
  const expected = Object.fromEntries(
    ["sv.gioi", "sv.kha", "sv.nguyco"].map((k) => [k, { recorded: SESSIONS, present: count(k, "PRESENT"), late: count(k, "LATE"), excused: 0, absent: count(k, "ABSENT"), speak_count: speak[k], points_total: (speak[k] * 0.25).toFixed(2) }]),
  );
  return { rng: Number(rngSeed), sessions: SESSIONS, attendance, speak, totals, expected };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const plan = crmPlan(Number(process.env.SEED_RNG ?? 20261029));
  if (process.argv.includes("--csv")) {
    console.log("student,recorded,present,late,excused,absent,speak_count,points_total");
    for (const [k, e] of Object.entries(plan.expected)) console.log([k, e.recorded, e.present, e.late, e.excused, e.absent, e.speak_count, e.points_total].join(","));
  } else console.log(JSON.stringify(plan, null, 1));
}
