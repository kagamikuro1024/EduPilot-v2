// US-P10-07 AC1–AC2: seed A (quan sát AI) + seed B (lỗ hổng kiến thức + điểm thi), chạy lại seed không đổi số hàng.
//   node scripts/check-p10-seed.mjs            (thêm --no-rerun để bỏ bước chạy lại seed)
// Đọc Postgres qua SEED_PSQL_CMD (mặc định compose project `edupilot`), KHÔNG gọi API ghi. Chỉ đếm / nhóm — không in chữ câu hỏi.
import { execSync } from "node:child_process";
import { failures, finish, must, reseed, root } from "./seed-check-lib.mjs";

const psql = process.env.SEED_PSQL_CMD ?? `docker compose --env-file .env.local -f docker-compose.local.yml -p ${process.env.COMPOSE_PROJECT_NAME || "edupilot"} exec -T postgres psql -U edupilot -d edupilot -At`;
const q = (sql) => execSync(`${psql} -F '|' -c "${sql.replace(/"/g, '\\"')}"`, { cwd: root, encoding: "utf8" }).trim().split("\n").filter(Boolean).map((l) => l.split("|"));
const one = (sql) => Number(q(sql)[0]?.[0] ?? 0);

function snapshot() {
  const t = (name, where = "true") => one(`select count(*) from ${name} where ${where}`);
  return {
    llm_audit: t("llm_audit"), payloads: t("llm_request_payloads"), chat: t("chat_messages"), exams: t("exams"), attempts: t("exam_attempts"),
    question_bank: t("question_bank"), insight_reports: t("insight_reports"), forum_threads: t("forum_threads"), mail_outbox: t("mail_outbox"),
  };
}

// ---- seed A: quan sát AI ----
const days = one("select count(distinct (created_at at time zone 'Asia/Ho_Chi_Minh')::date) from llm_audit");
must(one("select count(*) from llm_audit") >= 300, "llm_audit < 300 dòng");
must(days >= 7, `llm_audit chỉ có ${days} ngày (cần ≥ 7)`);
must(one("select count(*) from llm_audit where status not in ('ok','degraded','cancelled')") > 0, "llm_audit không có dòng lỗi");
must(one("select count(*) from llm_audit where degraded") > 0, "llm_audit không có dòng degraded");
must(one("select count(*) from llm_audit where fallback_index > 0") > 0, "llm_audit không có dòng chuyển dự phòng");
must(one("select count(*) from llm_audit where pii_masked_count > 0") > 0, "llm_audit không có dòng che thông tin");
must(one("select count(distinct course_id) from llm_audit where course_id is not null") >= 2, "llm_audit < 2 lớp");
must(one("select count(distinct task) from llm_audit") >= 3, "llm_audit < 3 tác vụ");
must(one("select count(*) from llm_request_payloads") >= 20, "llm_request_payloads < 20 dòng");
const leaks = one("select count(*) from llm_request_payloads p where p.masked_prompt ~* '[a-z0-9._-]+@[a-z0-9.-]+\\.[a-z]{2,}' or p.masked_prompt ~ '\\m[0-9]{8,}\\M'");
must(leaks === 0, `${leaks} bản prompt đã che còn email / dãy số dài`);
must(one("select count(*) from mail_outbox where status = 'DEAD'") >= 1, "không có thư gửi lỗi (mail_outbox DEAD)");

// ---- seed B: lỗ hổng kiến thức ----
const per = (code, where = "true") => one(`select count(*) from chat_messages m join courses c on c.id = m.course_id where c.class_code = '${code}' and m.role = 'USER' and ${where}`);
const c1 = per("761987", "m.content not like 'QC-%'");
const c2 = per("761988");
must(c1 >= 145 && c1 <= 165, `lớp 761987: ${c1} câu hỏi chat (cần ≈ 150)`);
must(c2 >= 35 && c2 <= 50, `lớp 761988: ${c2} câu hỏi chat (cần ≈ 40)`);
const noCtx = one("select count(*) from chat_messages a join courses c on c.id = a.course_id where c.class_code = '761987' and a.role = 'ASSISTANT' and a.no_context");
must(noCtx >= 10, `lớp 761987: ${noCtx} câu ngoài tài liệu (cần ≥ 10)`);
// ≥ 3 SV mỗi chủ đề lớn; chủ đề D đúng 2 SV (đếm theo từ khoá chủ đề — dữ liệu seed cố định)
const studentsFor = (code, like) => one(`select count(distinct m.user_id) from chat_messages m join courses c on c.id = m.course_id where c.class_code = '${code}' and m.role = 'USER' and (${like})`);
const A = studentsFor("761987", "m.content ~* 'phishing|ransomware|ddos|botnet|firewall|zero-day|spyware'");
const B = studentsFor("761987", "m.content ~* 'dự thi|học vụ|phúc khảo|thôi học|tốt nghiệp|bảo lưu'");
const D = studentsFor("761987", "m.content ~* 'trung bình trượt|hồi quy|mùa vụ|làm trơn|dự báo'");
const Cl2 = studentsFor("761988", "m.content ~* 'tcp|udp|dns|subnet|nat|dhcp|osi|http'");
must(A >= 3 && B >= 3, `chủ đề lớn do A=${A} / B=${B} sinh viên hỏi (cần ≥ 3)`);
must(D === 2, `chủ đề D do ${D} sinh viên hỏi (cần đúng 2)`);
must(Cl2 >= 3, `chủ đề C (lớp 761988) do ${Cl2} sinh viên hỏi (cần ≥ 3)`);
must(studentsFor("761987", "m.content ~* 'tcp|udp|dns|subnet|dhcp|osi'") === 0, "lớp 761987 có câu hỏi của chủ đề C");

// ---- seed B: điểm thi ----
must(one("select count(*) from exams e join courses c on c.id = e.course_id where c.class_code = '761987' and e.status = 'PUBLISHED'") >= 4, "lớp 761987 < 4 bài thi PE đã công bố");
const decl = q(`with ex as (select e.id, row_number() over (order by coalesce(e.opens_at, e.created_at), e.id) rn from exams e join courses c on c.id = e.course_id where c.class_code = '761987' and e.status = 'PUBLISHED'),
  sc as (select a.student_id, ex.rn, coalesce(a.adjusted_score, a.auto_score) s from exam_attempts a join ex on ex.id = a.exam_id where a.status = 'GRADED'),
  t as (select student_id, rn, s, lag(s) over w s1, lag(s, 2) over w s2, lag(rn) over w r1, lag(rn, 2) over w r2 from sc window w as (partition by student_id order by rn))
  select count(distinct student_id) from t where r1 = rn - 1 and r2 = rn - 2 and s1 > s and s2 > s1`);
must(Number(decl[0]?.[0]) >= 1, "không sinh viên nào giảm điểm 3 bài thi liên tiếp");

// ---- idempotent ----
const before = snapshot();
if (!process.argv.includes("--no-rerun")) {
  const code = reseed();
  must(code === 0, `chạy lại seed thoát ${code}`);
  const after = snapshot();
  for (const k of Object.keys(before)) must(before[k] === after[k], `chạy lại seed làm đổi ${k}: ${before[k]} → ${after[k]}`);
}

finish(`p10: llm_audit=${before.llm_audit}/${days} ngày · chat 761987=${c1} (A=${A} B=${B} D=${D} ngoài=${noCtx}) 761988=${c2} · điểm thi ≥ 4 bài, có ca giảm 3 liên tiếp${process.argv.includes("--no-rerun") ? "" : " (idempotent)"}`);
void failures;
