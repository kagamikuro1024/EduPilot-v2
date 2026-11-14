// US-DM-08 AC1: kiểm seed của decision model (provider giả, ba cổng, bật demo, idempotent).
//   node scripts/check-decision-seed.mjs            seed mặc định: ba cổng TẮT, ngưỡng / thời hạn mặc định
//   SEED_DECISION_DEMO=1 node scripts/check-decision-seed.mjs   sau `SEED_DECISION_DEMO=1 node scripts/seed.mjs`: ba cổng BẬT, lớp 761987 hiệu lực
// Chạy lại seed rồi kiểm lại: trạng thái không đổi (idempotent). Chỉ gọi API thật bằng tài khoản seed; không route _test.
import { call, courses, failures, finish, must, reseed, tok } from "./seed-check-lib.mjs";

const DEMO = process.env.SEED_DECISION_DEMO === "1";
const DEFAULTS = { THREADS_PRECHECK: [0.05, 800], INJECTION_FLAG: [0.5, 3000], CHAT_ROUTING: [0.95, 800] };

async function snapshot() {
  const admin = await tok("admin");
  const { c1 } = await courses();
  const ov = await call("GET", "/admin/llm/decision", { token: admin });
  must(ov.status === 200, `GET /admin/llm/decision → ${ov.status}`);
  const cg = await call("GET", `/courses/${c1}/decision-gates`, { token: admin });
  must(cg.status === 200, `GET /courses/{id}/decision-gates → ${cg.status}`);
  return { ov: ov.body, cg: cg.body };
}

const a = await snapshot();
must(a.ov.provider?.model === "decision-fake" && a.ov.provider?.fake === true, `provider decision = ${JSON.stringify(a.ov.provider)}, cần decision-fake (giả)`);
must(a.ov.real_allowed === false, "real_allowed phải false ở seed (bản thật chỉ khi PM cho phép)");
must(a.ov.gates?.length === 3, `số cổng = ${a.ov.gates?.length}, cần 3`);
for (const g of a.ov.gates ?? []) {
  const d = DEFAULTS[g.gate];
  must(Boolean(d), `cổng lạ ${g.gate}`);
  must(g.enabled === DEMO, `${g.gate}: enabled=${g.enabled}, cần ${DEMO}`);
  must(d && Number(g.threshold) === d[0] && g.timeout_ms === d[1], `${g.gate}: ngưỡng ${g.threshold} / ${g.timeout_ms} ms, cần mặc định ${d}`);
  const c = a.cg.items.find((x) => x.gate === g.gate);
  must(c?.enabled === DEMO && c?.effective === DEMO, `${g.gate} ở lớp 761987: enabled=${c?.enabled} effective=${c?.effective}, cần ${DEMO}`);
}

const code = reseed();
must(code === 0, `chạy lại seed thoát ${code}`);
const b = await snapshot();
must(JSON.stringify(b) === JSON.stringify(a), "chạy lại seed làm đổi cấu hình decision (không idempotent)");

finish(`decision: provider=${a.ov.provider?.model} cổng=${a.ov.gates?.map((g) => `${g.gate}:${g.enabled ? "bật" : "tắt"}`).join(", ")} (idempotent)`);
void failures;
