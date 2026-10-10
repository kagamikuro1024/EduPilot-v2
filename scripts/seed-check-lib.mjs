// Dùng chung cho `check-chat-seed.mjs` và `check-docs-seed.mjs`: gọi API thật bằng tài khoản seed, đếm, chạy lại seed để kiểm idempotent.
import { spawnSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
function envFileValue(key) {
  for (const f of [".env.local", ".env.example"]) {
    const p = path.join(root, f);
    if (!existsSync(p)) continue;
    const line = readFileSync(p, "utf8").split("\n").find((l) => l.startsWith(`${key}=`));
    if (line) return line.slice(key.length + 1).trim();
  }
  return undefined;
}
const setting = (k, d) => process.env[k] || envFileValue(k) || d;
export const API = setting("API_URL", "https://localhost/api/v1").replace(/\/$/, "");
export const PASSWORD = setting("SEED_DEFAULT_PASSWORD", "");
if (/^https:\/\/(localhost|127\.0\.0\.1)/.test(API)) process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
export const ORIGIN = new URL(API).origin;
export const DOMAIN = "@edupilot.local";
export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
export const failures = [];
export const must = (cond, msg) => {
  if (!cond) failures.push(msg);
  return cond;
};

export async function call(method, route, { token, json } = {}) {
  const headers = { Accept: "application/json", Origin: ORIGIN };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (json !== undefined) headers["Content-Type"] = "application/json";
  if (method === "POST") headers["Idempotency-Key"] = `chk-${crypto.randomUUID()}`;
  for (let i = 0; ; i++) {
    const res = await fetch(API + route, { method, headers, body: json === undefined ? undefined : JSON.stringify(json) });
    const body = res.status === 204 ? null : await res.json().catch(() => null);
    if (res.status === 429 && i < 4) {
      await sleep(2000);
      continue;
    }
    return { status: res.status, body };
  }
}

const tokens = {};
export async function tok(key) {
  if (!PASSWORD) throw new Error("Thiếu SEED_DEFAULT_PASSWORD (xem .env.example).");
  if (tokens[key]) return tokens[key];
  const r = await call("POST", "/auth/login", { json: { email: key + DOMAIN, password: PASSWORD } });
  if (r.status !== 200) throw new Error(`Không đăng nhập được ${key}: ${r.status}`);
  return (tokens[key] = r.body.access_token);
}

export async function listAll(route, token) {
  const out = [];
  let cursor = "";
  for (let i = 0; i < 30; i++) {
    const r = await call("GET", `${route}${route.includes("?") ? "&" : "?"}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { token });
    if (r.status !== 200) throw new Error(`GET ${route} → ${r.status}`);
    out.push(...r.body.items);
    if (!r.body.next_cursor) break;
    cursor = r.body.next_cursor;
  }
  return out;
}

/** id của hai lớp theo mã lớp, từ `GET /me/courses` của giảng viên. */
export async function courses() {
  const mine = await listAll("/me/courses", await tok("teacher"));
  const id = (code) => mine.find((m) => m.course.class_code === code)?.course.id;
  const c1 = id("761987"), c2 = id("761988");
  if (!c1 || !c2) throw new Error("Chưa có hai lớp 761987 / 761988 (chạy `node scripts/seed.mjs` trước).");
  return { c1, c2 };
}

/** Chạy lại seed; trả mã thoát. */
export function reseed() {
  const r = spawnSync(process.execPath, [path.join(root, "scripts", "seed.mjs")], { stdio: "inherit", env: process.env });
  return r.status ?? 1;
}

export function finish(summary) {
  console.log(summary);
  if (failures.length > 0) {
    for (const f of failures) console.error(`  ✗ ${f}`);
    process.exit(1);
  }
}
