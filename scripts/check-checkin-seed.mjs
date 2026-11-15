// FEAT-attendance-selfcheck L7 (US-AS-14 AC1): kiểm seed «Điểm danh bằng mã» của lớp 761988 qua API thật (tài khoản seed), rồi chạy lại seed để kiểm idempotent.
//   node scripts/check-checkin-seed.mjs        (sau `node scripts/seed.mjs`)
import { call, courses, failures, finish, listAll, must, reseed, tok } from "./seed-check-lib.mjs";

async function snapshot() {
  const t = await tok("teacher");
  const { c2 } = await courses();
  const course = await call("GET", `/courses/${c2}`, { token: t });
  must(course.status === 200 && course.body?.attendance?.self_checkin === true, `761988 phải bật self_checkin (status ${course.status})`);
  const sessions = await listAll(`/courses/${c2}/sessions`, t);
  const first = [...sessions].sort((a, b) => a.session_no - b.session_no)[0];
  must(Boolean(first), "761988 chưa có buổi nào");
  const wins = (await call("GET", `/courses/${c2}/sessions/${first?.id}/checkin-windows`, { token: t })).body?.items ?? [];
  must(wins.length === 2 && wins.every((w) => w.closed_at), `cần 2 cửa sổ ĐÃ ĐÓNG (không có cửa sổ mở), thấy ${wins.length}`);
  must(wins.some((w) => w.round === 1) && wins.some((w) => w.round === 2), "thiếu vòng 1 hoặc vòng 2");
  must(wins.some((w) => w.over_headcount === true), "thiếu cửa sổ over_headcount (headcount thấp hơn số lượt)");
  const rows = await listAll(`/courses/${c2}/sessions/${first?.id}/checkins`, t);
  const r1 = rows.filter((r) => r.round === 1);
  must(r1.length >= 20, `vòng 1 cần ≥ 20 lượt, thấy ${r1.length}`);
  const has = (code) => rows.filter((r) => r.flags.some((f) => f.code === code)).length;
  must(has("DEVICE_SHARED") === 2, `DEVICE_SHARED: ${has("DEVICE_SHARED")}, cần 2 (một cặp)`);
  for (const code of ["DEVICE_NEW", "NO_DEVICE_KEY", "TOKEN_LATE", "RECHECK_MISS"]) must(has(code) === 1, `${code}: ${has(code)}, cần 1`);
  must(r1.some((r) => r.flags.length === 0), "thiếu ca không cờ");
  must(rows.every((r) => r.student.student_code !== "20229001"), "sv.gioi (20229001) không được có lượt nào");
  const grid = await call("GET", `/courses/${c2}/sessions/${first?.id}/attendance`, { token: t });
  const late = (grid.body?.students ?? []).filter((s) => s.record?.status === "LATE").length;
  must(late === 1, `LATE: ${late}, cần 1`);
  return { windows: wins.map((w) => [w.round, w.headcount, w.checked_in]), rows: rows.map((r) => [r.round, r.student.id, r.flags.map((f) => f.code).sort().join(",")]).sort() };
}

const a = await snapshot();
const code = reseed();
must(code === 0, `chạy lại seed thoát ${code}`);
const b = await snapshot();
must(JSON.stringify(b) === JSON.stringify(a), "chạy lại seed làm đổi dữ liệu điểm danh bằng mã (không idempotent)");
finish(`checkin: ${a.windows.length} cửa sổ, ${a.rows.length} lượt (idempotent)`);
void failures;
