// Kiểm dữ liệu mẫu CRM / ticket / kiểm duyệt của US-S7-SEED bằng API thật (tài khoản giảng viên + từng sinh viên).
//   node scripts/check-crm-seed.mjs             in kết quả, thoát ≠ 0 khi lệch (chạy lại seed.mjs để kiểm idempotent; thêm --no-rerun để bỏ)
import { readFileSync } from "node:fs";
import path from "node:path";
import { call, courses, failures, finish, listAll, must, reseed, root, tok } from "./seed-check-lib.mjs";

const noRerun = process.argv.includes("--no-rerun");
const CODE = { "sv.gioi": "20229001", "sv.kha": "20229002", "sv.nguyco": "20229003" }; // MSSV của A, B, C trong scripts/seed.mjs
const csv = readFileSync(path.join(root, "seed", "expected_crm.csv"), "utf8").trim().split("\n").slice(1).map((l) => l.split(","));

const { c1, c2 } = await courses();
const t = await tok("teacher");

async function snapshot() {
  const students = await listAll(`/courses/${c1}/students`, t);
  const att = students.reduce((n, s) => n + s.attendance.recorded, 0);
  const tickets = await listAll(`/courses/${c1}/tickets?status=ALL`, t);
  const queue = await listAll(`/courses/${c1}/moderation/queue`, t);
  const byStatus = {};
  for (const x of tickets) byStatus[x.status] = (byStatus[x.status] ?? 0) + 1;
  return { students: students.length, attendanceRows: att, tickets: tickets.length, byStatus, queue: queue.length, speak: students.reduce((n, s) => n + s.participation.speak_count, 0) };
}

const a = await snapshot();
const students = await listAll(`/courses/${c1}/students`, t);
// AC1: 270 dòng, bảng đối chiếu A, B, C
must(a.attendanceRows === 270, `attendance_records lớp 1 = ${a.attendanceRows}, cần 270`);
for (const [key, recorded, present, late, excused, absent, speak, points] of csv) {
  const s = students.find((x) => x.student.student_code === CODE[key]);
  if (!must(s, `không thấy ${key} ở danh sách sinh viên`)) continue;
  const got = [s.attendance.recorded, s.attendance.present, s.attendance.late, s.attendance.excused, s.attendance.absent, s.participation.speak_count, s.participation.points_total].join(",");
  must(got === [recorded, present, late, excused, absent, speak, points].join(","), `${key}: ${got} ≠ ${[recorded, present, late, excused, absent, speak, points].join(",")}`);
}
const absentAll = students.reduce((n, s) => n + s.attendance.absent, 0);
must(absentAll >= 18 && absentAll <= 26, `ABSENT cả lớp = ${absentAll}, cần 18–26`);
must(a.speak >= 40 && a.speak <= 80, `phát biểu cả lớp = ${a.speak}, cần ≈ 60`);
// AC3: đúng một AT_RISK (C) và hai WATCH; Hôm nay có STUDENT_ATTENTION
const attention = await listAll(`/courses/${c1}/students?filter=attention`, t);
must(attention.length === 1 && attention[0].student.student_code === CODE["sv.nguyco"], `filter=attention = ${attention.length} (cần đúng Sinh viên C)`);
const levels = students.reduce((m, s) => ((m[s.risk.level] = (m[s.risk.level] ?? 0) + 1), m), {});
must(levels.AT_RISK === 1 && levels.WATCH === 2, `mức rủi ro lớp 1 ${JSON.stringify(levels)}, cần AT_RISK 1, WATCH 2`);
const today = (await call("GET", `/courses/${c1}/today`, { token: t })).body;
must(JSON.stringify(today).includes("STUDENT_ATTENTION"), "Hôm nay của giảng viên thiếu STUDENT_ATTENTION");
must((await listAll(`/courses/${c2}/students?filter=attention`, t)).length === 0, "lớp 761988 không được có sinh viên cần chú ý");
// AC4: 5 ticket đủ bốn trạng thái; lớp 2 không có
for (const st of ["OPEN", "CLAIMED", "ANSWERED", "CLOSED"]) must((a.byStatus[st] ?? 0) >= 1, `thiếu ticket ${st}`);
must(a.tickets === 5, `ticket lớp 1 = ${a.tickets}, cần đúng 5 (BUG-S1: lịch sử chat không được tự mở ticket)`);
const ticketStudents = (await listAll(`/courses/${c1}/tickets?status=ALL`, t)).map((x) => x.student.id);
const studentB = students.find((x) => x.student.student_code === CODE["sv.kha"]);
must(!ticketStudents.includes(studentB.student.id), "sinh viên B không được có ticket (demo D3: B tự tạo ticket mới)");
must((await listAll(`/courses/${c2}/tickets?status=ALL`, t)).length === 0, "lớp 761988 không được có ticket");
// AC5: hàng đợi kiểm duyệt
must(a.queue >= 2, `hàng đợi kiểm duyệt = ${a.queue}, cần ≥ 2`);
// AC8: sinh viên A, B, C chỉ thấy của mình; D (chưa vào lớp) không thấy lớp nào
for (const k of ["sv.gioi", "sv.kha", "sv.nguyco"]) {
  const tk = await tok(k);
  const mine = await call("GET", `/courses/${c1}/me/attendance`, { token: tk });
  must(mine.status === 200 && mine.body.attendance.recorded === 9, `${k}: me/attendance ${mine.status}`);
  must((await call("GET", `/courses/${c1}/students`, { token: tk })).status === 403, `${k}: không được đọc danh sách sinh viên`);
  must((await call("GET", `/courses/${c1}/students/${students[0].student.id}/profile`, { token: tk })).status === 403, `${k}: không được đọc hồ sơ`);
}
const d = await tok("sv.moi");
must((await listAll("/me/courses", d)).length === 0, "Sinh viên D (chưa vào lớp) không được có lớp nào");
must((await call("GET", `/courses/${c1}/me/attendance`, { token: d })).status === 403, "Sinh viên D đọc điểm danh lớp 1 phải 403");
const adm = await tok("admin");
for (const r of [`/courses/${c1}/tickets?status=ALL`, `/courses/${c1}/students`, `/courses/${c1}/sessions/${(await listAll(`/courses/${c1}/sessions`, t))[0].id}/attendance`]) must((await call("GET", r, { token: adm })).status === 403, `ADMIN đọc ${r} phải 403`);

let idem = "bỏ qua (--no-rerun)";
if (!noRerun) {
  must(reseed() === 0, "chạy lại seed.mjs lỗi");
  const b = await snapshot();
  idem = JSON.stringify(a) === JSON.stringify(b) ? "ok" : "KHÁC";
  must(idem === "ok", `chạy lại seed làm đổi số liệu: ${JSON.stringify(a)} → ${JSON.stringify(b)}`);
}
finish(`attendance=${a.attendanceRows} absent=${absentAll} speak=${a.speak} risk=${JSON.stringify(levels)} tickets=${a.tickets} ${JSON.stringify(a.byStatus)} queue=${a.queue} idempotent=${idem}`);
void failures;
