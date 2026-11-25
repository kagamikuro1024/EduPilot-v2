// Cấu hình LLM thật (OpenCode Go) cho stack VPS qua API quản trị: nhà cung cấp «OpenCode Go» (qwen3-embedding-8b 1536 chiều + deepseek-v4.1-flash) và các tuyến đã chọn.
//   OPENCODE_KEY_FILE=/opt/edupilot/secrets/opencode-go.key [ENV_FILE=.env.local] [BASE=https://<host>] node deploy/vps-llm-setup.mjs [TASK,TASK…]   (mặc định EMBEDDING,INSIGHT)
// Khoá chỉ đọc từ TỆP (chmod 600, ngoài repo), không bao giờ in; mật khẩu admin = SEED_DEFAULT_PASSWORD trong ENV_FILE. Chạy lại an toàn (nhà cung cấp có sẵn thì dùng lại, chỉ đặt tuyến).
// Seed cần CHAT / GRADING / QUESTION_GEN… ở bản giả (kết quả mong đợi tất định): đặt EMBEDDING + INSIGHT TRƯỚC seed, thêm CHAT (và các tuyến khác nếu muốn) SAU seed.
import { readFileSync } from "node:fs";
const ENV_FILE = process.env.ENV_FILE ?? ".env.local";
const env = Object.fromEntries(readFileSync(ENV_FILE, "utf8").split("\n").filter((l) => /^[A-Z_]+=/.test(l)).map((l) => [l.slice(0, l.indexOf("=")), l.slice(l.indexOf("=") + 1)]));
const BASE = process.env.BASE ?? `https://${env.EP_HOST}`, API = BASE + "/api/v1";
if (!process.env.OPENCODE_KEY_FILE) throw new Error("đặt OPENCODE_KEY_FILE (đường dẫn tệp khoá, không nhúng khoá vào lệnh)");
const key = readFileSync(process.env.OPENCODE_KEY_FILE, "utf8").trim();
async function call(method, route, { token, json, k } = {}) {
  const h = { Accept: "application/json", Origin: BASE };
  if (token) h.Authorization = `Bearer ${token}`;
  if (json !== undefined) h["Content-Type"] = "application/json";
  if (k) h["Idempotency-Key"] = k;
  const r = await fetch(API + route, { method, headers: h, body: json === undefined ? undefined : JSON.stringify(json) });
  return { status: r.status, body: r.status === 204 ? null : await r.json().catch(() => null) };
}
const must = (r, w) => { if (![200, 201].includes(r.status)) throw new Error(`${w} → ${r.status} ${r.body?.code ?? ""}`); return r.body; };
const tasks = (process.argv[2] ?? "EMBEDDING,INSIGHT").split(",");
const token = must(await call("POST", "/auth/login", { json: { email: "admin@edupilot.local", password: env.SEED_DEFAULT_PASSWORD } }), "login").access_token;
const have = must(await call("GET", "/admin/llm/providers", { token }), "list").items;
console.log("providers hiện có:", have.map((p) => p.name).join(",") || "(trống)");
let p = have.find((x) => x.name === "OpenCode Go");
if (!p) p = must(await call("POST", "/admin/llm/providers", { token, k: "llm-opencode-go", json: { type: "openai_compatible", name: "OpenCode Go", base_url: "https://opencode.ai/inference/go/openai/v1", api_key: key, enabled: true, models: [
  { model: "qwen3-embedding-8b", kind: "embedding", dims: 1536, price_in: "0", price_out: "0", enabled: true },
  { model: "deepseek-v4.1-flash", kind: "chat", price_in: "0", price_out: "0", enabled: true }] } }), "tạo provider");
const emb = p.models.find((m) => m.kind === "embedding")?.id, chat = p.models.find((m) => m.kind === "chat")?.id;
if (!emb || !chat) throw new Error("thiếu id model");
const routes = must(await call("GET", "/admin/llm/routes", { token }), "routes");
for (const t of tasks) {
  const cur = routes.items.find((r) => r.task === t);
  must(await call("PUT", "/admin/llm/routes", { token, json: { task: t, chain: [t === "EMBEDDING" ? emb : chat], version: cur?.version ?? 0 } }), `route ${t}`);
  console.log("tuyến", t, "→ OpenCode Go");
}
console.log(must(await call("GET", "/admin/llm/routes", { token }), "routes2").items.map((r) => `${r.task}:${r.chain?.length ?? 0}`).join(" "));
