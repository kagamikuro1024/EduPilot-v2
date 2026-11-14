#!/usr/bin/env node
// Cấu hình LLM của một stack đo (FEAT-nfr-eval US-NFR-08, TL-1) bằng API quản trị có sẵn — không đụng DB. Ba pha:
//   PHASE=probe  (đầu mỗi phiên đo, KHÔNG cần stack): gọi thẳng Gemini 3 lời gọi ngắn có `json_schema` cho từng model ứng viên theo thứ tự ưu tiên;
//                model qua nếu ≥ 2/3 lời gọi trả 200 trong 30 s. In chuỗi đề nghị `MODEL=… FALLBACK=…` (tối đa 2 model, 9 lời gọi).
//   PHASE=seed   (ngay sau khi admin có, TRƯỚC khi seed nạp tài liệu): tuyến sinh chữ → nhà cung cấp `fake` (seed trích quy chế bằng bản ghi demo),
//                EMBEDDING → Gemini THẬT (tài liệu, thread seed được nhúng bằng Gemini ngay từ đầu).
//   PHASE=live   (SAU seed): 6 tuyến sinh chữ → `gemini-chat` (MODEL, rồi FALLBACK nếu có); EMBEDDING giữ Gemini.
//                CHAT_PROVIDER=9router: nhà cung cấp `router-chat` (openai_compatible, ROUTER_BASE mặc định http://host.docker.internal:20128/v1, khoá 9router). EMBEDDING mặc định `embed-gia` (fake); EMBED_PROVIDER=gemini mới dùng Gemini (cần PM cho phép).
// Nhà cung cấp (loại `gemini`, không dùng `openai_compatible`; hai nhà cung cấp tách nhau vì Scheduler tính RPM theo nhà cung cấp):
//   · "gemini-chat"  model MODEL / FALLBACK, rpm_limit = CHAT_RPM (mặc định 5);
//   · "gemini-embed" gemini-embedding-001 ép `dimensions` = 1536 (khớp cột vector), rpm_limit 60;
//   · "chat-gia"     loại `fake`, chỉ cho pha seed.
// Tham số (biến môi trường): BASE (bắt buộc cho seed/live, vd https://localhost:743), ENV_FILE (mặc định .env.local; đọc GEMINI_API_KEY và
// SEED_DEFAULT_PASSWORD nếu biến chưa đặt), ADMIN_EMAIL (mặc định admin@edupilot.local), MODEL, FALLBACK, CHAT_RPM.
// Khoá chỉ đọc từ môi trường / ENV_FILE; không in, không ghi tệp. Idempotent: nhà cung cấp đã có thì dùng lại, chỉ đặt lại tuyến.
// Từ chối chạy nếu BASE là stack demo (cổng 80 / 443 / 3000).
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { refuseDemo } from "../benchmarks/lib/meta.mjs";

const ENV_FILE = process.env.ENV_FILE || ".env.local";
const fileEnv = existsSync(ENV_FILE)
  ? Object.fromEntries(readFileSync(ENV_FILE, "utf8").split("\n").filter((l) => /^[A-Z_]+=/.test(l)).map((l) => [l.slice(0, l.indexOf("=")), l.slice(l.indexOf("=") + 1)]))
  : {};
const setting = (k, d) => process.env[k] || fileEnv[k] || d;
const PHASE = process.env.PHASE || "live";
const GEMINI = "https://generativelanguage.googleapis.com/v1beta/openai";
const GEMINI_KEY = setting("GEMINI_API_KEY");
const CANDIDATES = ["gemini-3.8-flash", "gemini-3.5-flash", "gemini-3.1-flash-lite"];
const CHAT_TASKS = ["CHAT", "CLASSIFY", "UTILITY", "GRADING", "QUESTION_GEN", "INSIGHT"];
if (!["probe", "seed", "live"].includes(PHASE)) throw new Error("PHASE phải là probe | seed | live");
// Luật LLM 2026-11-14: sinh chữ qua 9router; embedding mặc định `fake` (xác định, 1536 chiều); Gemini chỉ khi EMBED_PROVIDER=gemini (PM cho phép) hoặc PHASE=probe.
const EMBED_GEMINI = process.env.EMBED_PROVIDER === "gemini";
if ((PHASE === "probe" || EMBED_GEMINI || (PHASE !== "seed" && process.env.CHAT_PROVIDER !== "9router")) && !GEMINI_KEY) throw new Error(`thiếu GEMINI_API_KEY (biến môi trường hoặc ${ENV_FILE})`);

if (PHASE === "probe") {
  const picked = [];
  for (const model of CANDIDATES) {
    let ok = 0;
    for (let i = 0; i < 3; i++) {
      const t0 = Date.now();
      try {
        const res = await fetch(`${GEMINI}/chat/completions`, {
          method: "POST",
          headers: { Authorization: `Bearer ${GEMINI_KEY}`, "Content-Type": "application/json" },
          body: JSON.stringify({
            model, max_tokens: 64, messages: [{ role: "user", content: "Trả về JSON {\"ok\": true}" }],
            response_format: { type: "json_schema", json_schema: { name: "probe", strict: true, schema: { type: "object", properties: { ok: { type: "boolean" } }, required: ["ok"], additionalProperties: false } } },
          }),
          signal: AbortSignal.timeout(30_000),
        });
        if (res.status === 200) ok++;
        console.log(`${model} #${i + 1}: ${res.status} ${Date.now() - t0} ms`);
      } catch (e) {
        console.log(`${model} #${i + 1}: ${e.name} ${Date.now() - t0} ms`);
      }
    }
    console.log(`${model}: ${ok}/3 ${ok >= 2 ? "QUA" : "TRƯỢT"}`);
    if (ok >= 2) picked.push(model);
    if (picked.length === 2) break;
  }
  if (picked.length === 0) throw new Error("không model nào qua kiểm tra; không chạy PHASE=live");
  console.log(`MODEL=${picked[0]}${picked[1] ? ` FALLBACK=${picked[1]}` : ""}`);
  process.exit(0);
}

const BASE = (process.env.BASE || "").replace(/\/$/, "");
if (!BASE) throw new Error("thiếu BASE (vd BASE=https://localhost:743, cổng HTTPS của Caddy trong stack ep-*)");
refuseDemo(BASE);
if (/^https:\/\/(localhost|127\.0\.0\.1)/.test(BASE)) process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0"; // chứng chỉ nội bộ của Caddy; probe đã chạy xong ở trên, nên không ảnh hưởng gọi Google
const API = `${BASE}/api/v1`;
const MODEL = process.env.MODEL;
const FALLBACK = process.env.FALLBACK;
const CHAT_RPM = Number(process.env.CHAT_RPM || 5);
if (PHASE === "live" && !MODEL) throw new Error("PHASE=live cần MODEL (chạy PHASE=probe để chọn)");

async function call(method, route, { token, json, key } = {}) {
  const headers = { Accept: "application/json", Origin: new URL(BASE).origin };
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

const password = setting("SEED_DEFAULT_PASSWORD");
if (!password) throw new Error(`thiếu SEED_DEFAULT_PASSWORD (biến môi trường hoặc ${ENV_FILE})`);
const login = await call("POST", "/auth/login", { json: { email: setting("ADMIN_EMAIL", "admin@edupilot.local"), password } });
const token = must(login, "đăng nhập admin").access_token;
const have = must(await call("GET", "/admin/llm/providers", { token }), "danh sách nhà cung cấp").items;
const ensure = async (name, input) => have.find((p) => p.name === name) ?? must(await call("POST", "/admin/llm/providers", { token, json: input, key: `llm-${name}` }), `tạo ${name}`);
// Khoá 9router: biến ROUTER_API_KEY, hoặc khoá đang bật đầu tiên trong ~/.9router/db/data.sqlite (không in).
function routerKey() {
  if (process.env.ROUTER_API_KEY) return process.env.ROUTER_API_KEY;
  const db = `${process.env.HOME}/.9router/db/data.sqlite`;
  const key = execFileSync("sqlite3", [db, "select key from apiKeys where isActive = 1 limit 1"], { encoding: "utf8" }).trim();
  if (!key) throw new Error("không có khoá 9router đang bật (đặt ROUTER_API_KEY)");
  return key;
}
const price = { price_in: "0", price_out: "0", enabled: true };

const gem = EMBED_GEMINI
  ? await ensure("gemini-embed", { type: "gemini", name: "gemini-embed", api_key: GEMINI_KEY, enabled: true, rpm_limit: 60, models: [{ model: "gemini-embedding-001", kind: "embedding", dims: 1536, ...price }] })
  : await ensure("embed-gia", { type: "fake", name: "embed-gia", enabled: true, models: [{ model: "fake-embed", kind: "embedding", dims: 1536, ...price }] });
const embLabel = EMBED_GEMINI ? "gemini-embed/gemini-embedding-001 (1536)" : "embed-gia/fake-embed (1536)";
const emb = gem.models.find((m) => m.kind === "embedding")?.id;
let chain, label;
if (PHASE === "seed") {
  const fake = await ensure("chat-gia", { type: "fake", name: "chat-gia", enabled: true, models: [{ model: "fake-chat", kind: "chat", ...price }] });
  chain = [fake.models.find((m) => m.kind === "chat")?.id];
  label = "chat-gia/fake-chat (pha seed)";
} else {
  const models = [MODEL, FALLBACK].filter(Boolean);
  const via9r = process.env.CHAT_PROVIDER === "9router"; // luật LLM 2026-11-14: sinh chữ đi qua 9router; GEMINI_API_KEY chỉ còn embedding
  const name = via9r ? "router-chat" : "gemini-chat";
  const input = via9r
    ? { type: "openai_compatible", name, base_url: process.env.ROUTER_BASE || "http://host.docker.internal:20128/v1", api_key: routerKey(), enabled: true, rpm_limit: CHAT_RPM, models: models.map((model) => ({ model, kind: "chat", ...price })) }
    : { type: "gemini", name, api_key: GEMINI_KEY, enabled: true, rpm_limit: CHAT_RPM, models: models.map((model) => ({ model, kind: "chat", ...price })) };
  const chat = await ensure(name, input);
  chain = models.map((m) => chat.models.find((x) => x.model === m)?.id);
  label = `${name}/${models.join(" → ")}`;
}
if (chain.some((id) => !id) || !emb) throw new Error("thiếu id mô hình sau khi tạo nhà cung cấp (nhà cung cấp cũ chưa có model này? xoá nó ở /settings/llm rồi chạy lại)");

const routes = must(await call("GET", "/admin/llm/routes", { token }), "đọc tuyến");
for (const t of [...CHAT_TASKS, "EMBEDDING"]) {
  const cur = routes.items.find((r) => r.task === t);
  must(await call("PUT", "/admin/llm/routes", { token, json: { task: t, chain: t === "EMBEDDING" ? [emb] : chain, version: cur?.version ?? 0 } }), `đặt tuyến ${t}`);
  console.log(`tuyến ${t} → ${t === "EMBEDDING" ? embLabel : label}`);
}
