// QC US-P8-01: thư viện gọi API hộp đen (fetch + psql qua docker exec). Biến: GW (mặc định :18080), QC_DB, QC_REDIS_DB.
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
export const GW = process.env.GW ?? "http://localhost:18080/api/v1";
export const PW = process.env.SEED_DEFAULT_PASSWORD;
export const sha = (buf) => createHash("sha256").update(buf).digest("hex");
export const psql = (sql) => spawnSync("docker", ["exec", "edupilot-postgres-1", "psql", "-U", "edupilot", "-d", process.env.QC_DB ?? "qc_p801", "-At", "-F|", "-c", sql], { encoding: "utf8" }).stdout.trim();
export const rds = (...a) => spawnSync("docker", ["exec", "edupilot-redis-1", "redis-cli", "-n", process.env.QC_REDIS_DB ?? "9", ...a], { encoding: "utf8" }).stdout.trim();
let n = 0;
export const idem = () => `qc-${Date.now()}-${++n}`;
export async function call(method, path, { token, body, key, headers = {} } = {}) {
  const h = { "Content-Type": "application/json", ...headers };
  if (token) h.Authorization = `Bearer ${token}`;
  if (key) h["Idempotency-Key"] = key;
  const r = await fetch(GW + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  const t = await r.text(); let j; try { j = JSON.parse(t); } catch { j = t; }
  return { status: r.status, body: j, headers: r.headers };
}
export async function login(email) { const r = await call("POST", "/auth/login", { body: { email, password: PW } }); if (!r.body.access_token) throw new Error("login " + email + " " + JSON.stringify(r.body)); return r.body.access_token; }
export async function ctx() {
  const adm = await login("admin@edupilot.local");
  const cs = (await call("GET", "/admin/courses?limit=50", { token: adm })).body.items;
  return { adm, tch: await login("teacher@edupilot.local"), ta: await login("ta@edupilot.local"), sva: await login("sv.gioi@edupilot.local"), svb: await login("sv.kha@edupilot.local"),
    C1: cs.find((c) => c.class_code === "761987").id, C2: cs.find((c) => c.class_code === "761988").id };
}
export function presignBody(file, buf, over = {}) { return { purpose: "document", filename: file, mime_type: over.mime ?? "application/pdf", size_bytes: over.size ?? buf.length, sha256: over.sha ?? sha(buf), ...over.extra }; }
import { writeFileSync } from "node:fs";
// PUT bằng curl: fetch của Node không cho gửi Content-Length lệch thân (cần cho TC kích thước sai).
export async function put(url, headers, buf) { const f = `/tmp/qc-put-${process.pid}.bin`; writeFileSync(f, buf); const a = ["-s", "-o", "/dev/null", "-w", "%{http_code}", "-X", "PUT", "--data-binary", "@" + f]; for (const [k, v] of Object.entries(headers ?? {})) a.push("-H", `${k}: ${v}`); a.push(url); return Number(spawnSync("curl", a, { encoding: "utf8" }).stdout); }
export async function upload(c, token, file, buf, { title, type = "LECTURE", extra = {}, key, putBuf } = {}) {
  const p = await call("POST", `/courses/${c}/uploads/presign`, { token, body: presignBody(file, buf) });
  if (p.status !== 200) return { step: "presign", ...p };
  const ps = await put(p.body.url, p.body.headers ?? {}, putBuf ?? buf);
  const r = await call("POST", `/courses/${c}/uploads/complete`, { token, key: key ?? idem(), body: { upload_id: p.body.upload_id, title: title ?? file, type, ...extra } });
  return { step: "complete", putStatus: ps, presign: p.body, ...r };
}
export const waitDoc = async (id, want = ["READY", "FAILED"], ms = 600000) => { const t0 = Date.now(); for (;;) { const s = psql(`select status from documents where id='${id}'`); if (want.includes(s)) return s; if (Date.now() - t0 > ms) return "TIMEOUT:" + s; await new Promise((r) => setTimeout(r, 3000)); } };
export const rd = (p) => readFileSync(p);
export const out = (tc, ok, note = "") => console.log(`${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
