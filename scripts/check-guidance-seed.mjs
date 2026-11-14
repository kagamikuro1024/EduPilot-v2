// FEAT-grading-guidance L6: kiểm seed «Học từ lần sửa» (cờ, 8 kết quả đã duyệt «sở thích gắt», idempotent).
//   node scripts/check-guidance-seed.mjs                          seed mặc định: cờ TẮT, ngăn lưu ý của BT03 không bật (enabled=false)
//   SEED_GUIDANCE_DEMO=1 node scripts/check-guidance-seed.mjs     sau `SEED_GUIDANCE_DEMO=1 node scripts/seed.mjs`: cờ BẬT, `Rút lưu ý` đủ điều kiện (≥ 5 bài, ≥ 3 lần sửa)
// Chỉ gọi API thật bằng tài khoản seed; không route _test.
import { call, courses, failures, finish, listAll, must, reseed, tok } from "./seed-check-lib.mjs";

const DEMO = process.env.SEED_GUIDANCE_DEMO === "1";

async function snapshot() {
  const t = await tok("teacher");
  const { c1 } = await courses();
  const flags = await call("GET", `/courses/${c1}/guidance-flags`, { token: t });
  must(flags.status === 200, `GET guidance-flags → ${flags.status}`);
  const bt03 = (await listAll(`/courses/${c1}/assignments`, t)).find((a) => a.submission_code === "BT03");
  must(Boolean(bt03), "không thấy bài tập BT03 của lớp 761987");
  const ov = await call("GET", `/courses/${c1}/assignments/${bt03?.id}/guidance`, { token: t });
  must(ov.status === 200, `GET guidance của BT03 → ${ov.status}`);
  const stats = DEMO ? await call("GET", `/courses/${c1}/assignments/${bt03?.id}/guidance/stats`, { token: t }) : null;
  return { flags: flags.body, ov: ov.body, stats: stats?.body ?? null };
}

const a = await snapshot();
const gg = a.flags?.items?.find((f) => f.feature === "GRADING_GUIDANCE");
must(gg?.enabled === DEMO, `GRADING_GUIDANCE: enabled=${gg?.enabled}, cần ${DEMO}`);
for (const f of a.flags?.items ?? []) if (f.feature !== "GRADING_GUIDANCE") must(f.enabled === false, `${f.feature} phải TẮT ở seed`);
must(a.ov?.enabled === DEMO, `overview.enabled=${a.ov?.enabled}, cần ${DEMO}`);
if (DEMO) {
  must(a.ov.current_version === 0, `BT03 phải ở phiên bản 0 (chưa áp dụng lưu ý nào), đang ${a.ov.current_version}`);
  must(a.ov.reflect?.eligible === true && a.ov.reflect.results >= 8 && a.ov.reflect.edits >= 8, `reflect = ${JSON.stringify(a.ov.reflect)}, cần eligible, ≥ 8 bài, ≥ 8 lần sửa`);
  must((a.ov.counts?.draft ?? 0) >= 2, `lưu ý nháp mẫu: ${a.ov.counts?.draft ?? 0}, cần ≥ 2 (Chờ duyệt)`);
  const v0 = a.stats?.items?.find((x) => x.version === 0);
  must(v0?.n >= 8, `số đo phiên bản 0: n=${v0?.n}, cần ≥ 8`);
  must(v0 && Number(v0.bias_total) >= 0.9, `lệch tổng AI − giảng viên = ${v0?.bias_total}, cần ≈ +1,00 (sở thích gắt)`);
}

const code = reseed();
must(code === 0, `chạy lại seed thoát ${code}`);
const b = await snapshot();
must(JSON.stringify(b) === JSON.stringify(a), "chạy lại seed làm đổi trạng thái «học từ lần sửa» (không idempotent)");

finish(`guidance: cờ=${gg?.enabled ? "bật" : "tắt"} reflect=${JSON.stringify(a.ov?.reflect)} (idempotent)`);
void failures;
