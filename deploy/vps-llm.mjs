#!/usr/bin/env node
// Cấu hình LLM thật cho VPS (OPS-VPS) bằng API quản trị có sẵn — không đụng DB trực tiếp:
//   · nhà cung cấp `openai_compatible` "9router" (http://9router:20128/v1, model chat MODEL) cho mọi tác vụ sinh chữ;
//   · nhà cung cấp `fake` "embedding-gia" (model embedding 1536 chiều, tất định) cho EMBEDDING — 9router KHÔNG có embeddings.
// Khoá 9router chỉ được đọc từ sqlite của 9router TRONG tiến trình này rồi gửi thẳng cho gateway (lưu mã hoá AES-GCM); không in, không ghi tệp.
//   /opt/node/bin/node deploy/vps-llm.mjs        (chạy trên VPS, trong /opt/edupilot; idempotent: nhà cung cấp đã có thì chỉ đặt lại tuyến)
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";

const env = Object.fromEntries(readFileSync(".env.local", "utf8").split("\n").filter((l) => /^[A-Z_]+=/.test(l)).map((l) => [l.slice(0, l.indexOf("=")), l.slice(l.indexOf("=") + 1)]));
const API = `https://${env.EP_HOST}/api/v1`;
const MODEL = process.env.MODEL || "ag/gemini-3.8-flash";
const SQLITE = "/root/9router/data/db/data.sqlite";
const CHAT_TASKS = ["CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT"];

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

async function ensure(name, input) {
  const cur = have.find((p) => p.name === name);
  if (cur) return cur;
  return must(await call("POST", "/admin/llm/providers", { token, json: input, key: `vps-llm-${name}` }), `tạo ${name}`);
}

const apiKey = execFileSync("sqlite3", ["-readonly", SQLITE, "select key from apiKeys where isActive=1 limit 1;"], { encoding: "utf8" }).trim();
if (!apiKey) throw new Error("9router không có khoá đang bật");
const router = await ensure("9router", {
  type: "openai_compatible", name: "9router", base_url: "http://9router:20128/v1", api_key: apiKey, enabled: true,
  rpm_limit: Number(env.LLM_DEFAULT_RPM || 20),
  models: [{ model: MODEL, kind: "chat", price_in: "0", price_out: "0", enabled: true }],
});
const embed = await ensure("embedding-gia", {
  type: "fake", name: "embedding-gia", enabled: true,
  models: [{ model: "fake-embedding", kind: "embedding", dims: 1536, price_in: "0", price_out: "0", enabled: true }],
});
const chat = router.models.find((m) => m.model === MODEL)?.id;
const emb = embed.models.find((m) => m.kind === "embedding")?.id;
if (!chat || !emb) throw new Error("thiếu id mô hình sau khi tạo nhà cung cấp");

const routes = must(await call("GET", "/admin/llm/routes", { token }), "đọc tuyến");
for (const t of [...CHAT_TASKS, "EMBEDDING"]) {
  const cur = routes.items.find((r) => r.task === t);
  const r = await call("PUT", "/admin/llm/routes", { token, json: { task: t, chain: [t === "EMBEDDING" ? emb : chat], version: cur?.version ?? 0 } });
  must(r, `đặt tuyến ${t}`);
  console.log(`tuyến ${t} → ${t === "EMBEDDING" ? "embedding-gia/fake-embedding" : `9router/${MODEL}`}`);
}
