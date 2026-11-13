#!/usr/bin/env node
// Cấu hình LLM cho VPS (OPS-VPS) bằng API quản trị có sẵn — không đụng DB trực tiếp. Hai pha:
//   PHASE=seed  (chạy NGAY SAU khi admin có, TRƯỚC khi seed nạp tài liệu): tuyến sinh chữ → nhà cung cấp `fake` (seed trích quy chế bằng bản ghi demo), EMBEDDING → Gemini THẬT
//               (tài liệu, thread seed được nhúng bằng Gemini ngay từ đầu, không phải nạp lại).
//   PHASE=live  (SAU seed): 6 tuyến sinh chữ → 9router `openai_compatible` (model MODEL, mặc định ag/gemini-3.8-flash); EMBEDDING giữ Gemini.
// Nhà cung cấp:
//   · "9router" openai_compatible http://9router:20128/v1 — khoá đọc từ sqlite của 9router TRONG tiến trình này (không in, không ghi tệp);
//   · "gemini-embed" loại `gemini` (https://generativelanguage.googleapis.com/v1beta/openai/), model gemini-embedding-001 ép `dimensions` = 1536 (khớp cột vector), khoá = GEMINI_API_KEY trong .env.local;
//   · "chat-gia" loại `fake` chỉ cho pha seed.
// Idempotent: nhà cung cấp đã có thì dùng lại, chỉ đặt lại tuyến.   /opt/node/bin/node deploy/vps-llm.mjs   (trong /opt/edupilot)
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

const env = Object.fromEntries(readFileSync(".env.local", "utf8").split("\n").filter((l) => /^[A-Z_]+=/.test(l)).map((l) => [l.slice(0, l.indexOf("=")), l.slice(l.indexOf("=") + 1)]));
const API = `https://${env.EP_HOST}/api/v1`;
const PHASE = process.env.PHASE || "live";
const MODEL = process.env.MODEL || "ag/gemini-3.8-flash";
const SQLITE = "/root/9router/data/db/data.sqlite";
const CHAT_TASKS = ["CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT"];
if (!["seed", "live"].includes(PHASE)) throw new Error("PHASE phải là seed | live");

async function call(method, route, { token, json, key } = {}) {
  const headers = { Accept: "application/json", Origin: `https://${env.EP_HOST}` };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (json !== undefined) headers["Content-Type"] = "application/json";
  if (key) headers["Idempotency-Key"] = key;
  const res = await fetch(API + route, { method, headers, body: json === undefined ? undefined : JSON.stringify(json) });
  const body = res.status === 204 ? null : await res.json().catch(() => null);
  return { status: res.status, body };
}
const must = (r, what, ok = [200, 201]) => {
  if (!ok.includes(r.status)) throw new Error(`${what} → ${r.status} ${JSON.stringify(r.body?.code ?? r.body?.message ?? "")}`);
  return r.body;
};

const login = await call("POST", "/auth/login", { json: { email: "admin@edupilot.local", password: env.SEED_DEFAULT_PASSWORD } });
const token = must(login, "đăng nhập admin").access_token;
const have = must(await call("GET", "/admin/llm/providers", { token }), "danh sách nhà cung cấp").items;
const ensure = async (name, input) => have.find((p) => p.name === name) ?? must(await call("POST", "/admin/llm/providers", { token, json: input, key: `vps-llm-${name}` }), `tạo ${name}`);
const rpm = Number(env.LLM_DEFAULT_RPM || 20);

if (!env.GEMINI_API_KEY) throw new Error("thiếu GEMINI_API_KEY trong .env.local (khoá embeddings Gemini)");
const gem = await ensure("gemini-embed", {
  type: "gemini", name: "gemini-embed", api_key: env.GEMINI_API_KEY, enabled: true, rpm_limit: 60,
  models: [{ model: "gemini-embedding-001", kind: "embedding", dims: 1536, price_in: "0", price_out: "0", enabled: true }],
});
const emb = gem.models.find((m) => m.kind === "embedding")?.id;
let chat, label;
if (PHASE === "seed") {
  const fake = await ensure("chat-gia", { type: "fake", name: "chat-gia", enabled: true, models: [{ model: "fake-chat", kind: "chat", price_in: "0", price_out: "0", enabled: true }] });
  chat = fake.models.find((m) => m.kind === "chat")?.id;
  label = "chat-gia/fake-chat (pha seed)";
} else {
  const apiKey = execFileSync("sqlite3", ["-readonly", SQLITE, "select key from apiKeys where isActive=1 limit 1;"], { encoding: "utf8" }).trim();
  if (!apiKey) throw new Error("9router không có khoá đang bật");
  const router = await ensure("9router", {
    type: "openai_compatible", name: "9router", base_url: "http://9router:20128/v1", api_key: apiKey, enabled: true, rpm_limit: rpm,
    models: [{ model: MODEL, kind: "chat", price_in: "0", price_out: "0", enabled: true }],
  });
  chat = router.models.find((m) => m.model === MODEL)?.id;
  label = `9router/${MODEL}`;
}
if (!chat || !emb) throw new Error("thiếu id mô hình sau khi tạo nhà cung cấp");

const routes = must(await call("GET", "/admin/llm/routes", { token }), "đọc tuyến");
for (const t of [...CHAT_TASKS, "EMBEDDING"]) {
  const cur = routes.items.find((r) => r.task === t);
  must(await call("PUT", "/admin/llm/routes", { token, json: { task: t, chain: [t === "EMBEDDING" ? emb : chat], version: cur?.version ?? 0 } }), `đặt tuyến ${t}`);
  console.log(`tuyến ${t} → ${t === "EMBEDDING" ? "gemini-embed/gemini-embedding-001 (1536)" : label}`);
}
