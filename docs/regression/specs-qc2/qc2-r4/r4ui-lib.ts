// QC2-I slice r4ui: thư viện dùng chung (đăng nhập thật qua API, xoá giới hạn auth, psql, Mailpit, ghi kết quả).
import { execFileSync } from "node:child_process";
import { appendFileSync, existsSync, readFileSync } from "node:fs";
import { randomUUID } from "node:crypto";
import { expect, test, type BrowserContext, type Page } from "@playwright/test";

process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
export const BASE = process.env.QC_BASE ?? "https://localhost:1143";
export const API = `${BASE}/api/v1`;
export const PWD = "Qc2iu!Pass-2026x";
export const SEED_PW = "Edupilot#Seed-2026";
export const MP = "http://localhost:8725/api/v1";
export const S = (existsSync("../docs/regression/evidence-4/qc2iu-state.json") ? JSON.parse(readFileSync("../docs/regression/evidence-4/qc2iu-state.json", "utf8")) : {}) as Record<string, any>;
export const em = (k: string) => `qc2iu.${k}@edupilot.local`;

export const clearRL = () => {
  try {
    execFileSync("docker", ["exec", "ep-r-redis-1", "sh", "-c", "for p in 'ep:rl:auth*' 'ep:auth:forgot:*' 'ep:auth:ipfail:*' 'ep:auth:fail:*'; do redis-cli --scan --pattern \"$p\" | xargs -r redis-cli del >/dev/null; done"]);
  } catch {
    /* môi trường: bỏ qua */
  }
};
export const psql = (q: string) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-F", "|", "-c", q]).toString().trim();
export const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** Gọi API trực tiếp (Node fetch), chịu 5xx bằng thử lại. */
export async function call(method: string, path: string, token?: string | null, body?: unknown, headers: Record<string, string> = {}) {
  for (let i = 0; i < 6; i++) {
    const h: Record<string, string> = { "content-type": "application/json", ...headers };
    if (token) h.authorization = `Bearer ${token}`;
    if (method === "POST" && !h["Idempotency-Key"]) h["Idempotency-Key"] = randomUUID();
    try {
      const r = await fetch(API + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
      if ([502, 503, 504].includes(r.status) && i < 5) { await sleep(8000); continue; }
      const t = await r.text();
      let j: any = {};
      try { j = JSON.parse(t); } catch { j = { _raw: t.slice(0, 300) }; }
      return { status: r.status, body: j, headers: r.headers };
    } catch (e) {
      if (i === 5) throw e;
      await sleep(8000);
    }
  }
  throw new Error("unreachable");
}
export async function tokenOf(email: string, pw = PWD): Promise<string> {
  clearRL();
  const r = await call("POST", "/auth/login", null, { email, password: pw });
  expect(r.status, `login ${email}`).toBe(200);
  return r.body.access_token;
}

/** Đăng nhập thật qua API trong ngữ cảnh trình duyệt (cookie ep_rt vào context) + đánh dấu máy đã đăng nhập (US-UX91-21). */
export async function apiLogin(page: Page, email: string, pw = PWD) {
  clearRL();
  const ctx = page.context();
  await ctx.addInitScript(() => localStorage.setItem("ep_returning", "1"));
  const r = await ctx.request.post("/api/v1/auth/login", { data: { email, password: pw }, headers: { "Idempotency-Key": randomUUID() } });
  expect(r.status(), `login ${email}`).toBe(200);
  return (await r.json()).access_token as string;
}
/** Đăng nhập thật qua màn /login. */
export async function uiLogin(page: Page, email: string, pw = PWD, next?: string) {
  clearRL();
  await page.goto("/login" + (next ? `?next=${encodeURIComponent(next)}` : ""));
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Mật khẩu", { exact: true }).fill(pw);
  await page.getByRole("button", { name: "Đăng nhập", exact: true }).click();
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
}

import { loadAudit, runAudit } from "../../../../frontend/e2e/support/audit";
export { loadAudit as loadAuditSrc, runAudit };

export const mp = async (path: string) => (await fetch(MP + path)).json() as Promise<any>;
export async function mailTo(addr: string, subj: RegExp, since: number, wait = 40) {
  const end = Date.now() + wait * 1000;
  for (;;) {
    const j = await mp("/messages?limit=300");
    const ms = (j.messages ?? []).filter((m: any) => m.To.some((a: any) => a.Address === addr) && subj.test(m.Subject) && new Date(m.Created).getTime() >= since - 3000);
    if (ms.length) {
      ms.sort((a: any, b: any) => a.Created.localeCompare(b.Created));
      return mp("/message/" + ms.at(-1).ID);
    }
    if (Date.now() > end) throw new Error("không có thư " + addr);
    await sleep(1000);
  }
}
export const linkToken = (m: any, needle: RegExp) => {
  for (const u of (m.Text as string).match(/https?:\/\/[^\s<>")]+/g) ?? []) if (needle.test(u)) { const p = new URL(u); return p.searchParams.get("token") ?? p.pathname.split("/").pop()!; }
  throw new Error("không có liên kết");
};

export const FORBID = /\b(RAG|PII|fallback|trace|provider|confidence|redaction|embedding|prompt|LLM|placeholder)\b|\bsession\b|\bjwt\b|\bbcrypt\b/i;
/** Từ kỹ thuật trên một trang theo cách TC: chữ trong <main> (không phân biệt hoa thường). */
export async function techWords(page: Page, words = ["token", "session", "refresh", "jwt", "bcrypt", "oidc"]) {
  const t = (await page.locator("main").innerText()).toLowerCase();
  return words.filter((w) => t.includes(w));
}

/** Ghi KQ từng ca: afterEach đọc mã TC trong tiêu đề "[TC-Pxxx-yy ...]" và ghi jsonl. */
export function installRec(file: string) {
  test.afterEach(async ({}, info) => {
    const ids = info.title.match(/TC-P\d+-\d+/g) ?? [];
    const notes = info.annotations.filter((a) => a.type === "note").map((a) => a.description).join(" ; ");
    for (const id of ids)
      appendFileSync(`../docs/regression/evidence-4/logs/s4n-res/${file}.jsonl`, JSON.stringify({ id, project: info.project.name, status: info.status, ms: info.duration, notes, err: (info.error?.message ?? "").split("\n").slice(0, 6).join(" ").slice(0, 500) }) + "\n");
  });
}
export const note = (s: string) => test.info().annotations.push({ type: "note", description: s });
export type { BrowserContext, Page };

import AxeBuilder from "@axe-core/playwright";
/** axe WCAG 2.2 AA: trả danh sách "luật:số nút" của vi phạm serious/critical (a11y.spec.ts của sản phẩm chặn đúng hai mức này). */
export async function axeBad(page: Page) {
  const r = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
  return r.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => `${v.id}:${v.nodes.length}`);
}
/** AUDIT + TOUCH ở bề rộng hiện tại: {ox, cut, touch}. */
export async function layout(page: Page) {
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  const touch = (await page.evaluate(TOUCH_SRC)) as unknown[];
  return { ox: a.ox, cut: a.cut, ell: a.ell, touch };
}

export const stamp = () => Date.now().toString(36).toUpperCase();
/** Sinh viên mới (đăng ký + xác minh qua Mailpit); trả email. */
export async function freshStudent(tag: string, code = "") {
  clearRL();
  const t0 = Date.now();
  const email = `qc2iu.${tag}${t0}@edupilot.local`;
  const r = await call("POST", "/auth/register", null, { email, password: PWD, full_name: `QC2IU ${tag}`, student_code: code });
  expect(r.status).toBe(202);
  const tok = linkToken(await mailTo(email, /Xác minh/i, t0), /verify-email/);
  expect((await call("POST", "/auth/verify-email", null, { token: tok })).status).toBe(200);
  return email;
}
/** Lớp mới do Admin riêng mở cho giảng viên `teacherId` (tuỳ chọn trợ giảng). */
export async function newCourse(teacherId: string, tag: string, taIds: string[] = []) {
  const adm = await tokenOf(em("adm"));
  const code = `QC2IU${tag}${stamp()}`.slice(0, 16);
  const r = await call("POST", "/admin/courses", adm, { subject_code: "INT1006", class_code: code, name: `QC2IU ${tag}`, semester: "2026-2027-HK1", teacher_id: teacherId, ...(taIds.length ? { ta_ids: taIds } : {}) });
  expect(r.status, JSON.stringify(r.body)).toBe(201);
  return { id: r.body.id as string, code };
}
/** Mã tham gia hiện tại của lớp (đọc bằng token giảng viên). */
export async function joinInfo(courseId: string, teacherEmail: string) {
  const t = await tokenOf(teacherEmail);
  const r = await call("GET", `/courses/${courseId}/join-code`, t);
  expect(r.status).toBe(200);
  return { ...r.body, token: t } as { join_code: string; version: number; token: string; [k: string]: unknown };
}
