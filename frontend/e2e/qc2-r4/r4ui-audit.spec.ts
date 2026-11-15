import { writeFileSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";
import { SEED_PW, apiLogin, installRec, loadAuditSrc, note, runAudit } from "./r4ui-lib";

// QC2-I r4ui TC-P212-21: bản Playwright của docs/sprints/4/qc/scripts/audit-login.mjs (run + matrix), đăng nhập thật bằng tài khoản seed (chỉ đọc).
installRec("r4ui-audit");
const FORBIDDEN = /\b(RAG|PII|fallback|trace|provider|confidence|redaction|embedding|prompt|LLM|placeholder)\b|độ tin cậy|cần chú ý|rủi ro|\[\[SV_/i;
const ROUTES: Record<string, string[]> = {
  student: ["/", "/chat", "/threads", "/threads/t-cbc", "/practice", "/practice/at-symmetric", "/practice/at-quiz01", "/practice/history", "/library", "/calendar", "/me", "/assignments/bt03", "/join", "/join/BX4P9TW"],
  ta: ["/", "/threads", "/calendar", "/inbox", "/students", "/students/sv-3", "/attendance", "/class/members", "/gradebook", "/gradebook/scheme", "/grading", "/grading/sub-bt03-sv-2", "/questions", "/documents", "/insights", "/analytics"],
  teacher: ["/", "/threads", "/threads/t-cbc", "/calendar", "/inbox", "/students", "/students/sv-3", "/attendance", "/class/members", "/gradebook", "/gradebook/scheme", "/grading", "/grading/sub-bt03-sv-2", "/questions", "/documents", "/insights", "/analytics", "/observability", "/settings/llm", "/settings/integrations"],
  admin: ["/", "/observability", "/settings/llm", "/settings/integrations", "/admin/courses", "/admin/users"],
};
const ACCT: Record<string, string> = { student: "sv.gioi", ta: "ta", teacher: "teacher", admin: "admin" };
const VPS = [{ w: 1440, h: 900, tag: "1440" }, { w: 390, h: 844, tag: "390" }, { w: 375, h: 812, tag: "375", only: (role: string, r: string) => role === "student" || r === "/attendance" || r === "/inbox" }];
const G = {
  ALL: ["/"],
  SV: ["/chat", "/practice", "/practice/at-symmetric", "/practice/at-quiz01", "/practice/history", "/library", "/me", "/assignments/bt03", "/join", "/join/BX4P9TW"],
  SV_TA_GV: ["/threads", "/threads/t-cbc", "/calendar"],
  TA_GV: ["/inbox", "/students", "/students/sv-3", "/attendance", "/class/members", "/gradebook", "/gradebook/scheme", "/grading", "/grading/sub-bt03-sv-2", "/questions", "/documents", "/insights", "/analytics"],
  GV_AD: ["/observability", "/settings/llm", "/settings/integrations"],
  ADMIN: ["/admin/courses", "/admin/users"],
};
const ALLOW: Record<string, string[]> = { student: ["ALL", "SV", "SV_TA_GV"], ta: ["ALL", "SV_TA_GV", "TA_GV"], teacher: ["ALL", "SV_TA_GV", "TA_GV", "GV_AD"], admin: ["ALL", "GV_AD", "ADMIN"] };

async function settle(page: Page, ms = 1700) {
  await page.waitForLoadState("domcontentloaded").catch(() => undefined);
  await page.waitForTimeout(ms);
}

for (const role of Object.keys(ROUTES)) {
  test(`[TC-P212-21] audit đăng nhập thật · ${role}`, async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
    test.setTimeout(285_000);
    const { AUDIT_SRC, TOUCH_SRC } = await loadAuditSrc();
    await apiLogin(page, `${ACCT[role]}@edupilot.local`, SEED_PW);
    const rows: Array<Record<string, unknown>> = [];
    for (const vp of VPS) {
      await page.setViewportSize({ width: vp.w, height: vp.h });
      for (const r of ROUTES[role]) {
        if (vp.only && !vp.only(role, r)) continue;
        try {
          await page.goto(r);
          await settle(page);
          const a = await runAudit(page, AUDIT_SRC);
          const touch = vp.w <= 390 && (role === "student" || r === "/attendance" || r === "/inbox") ? ((await page.evaluate(TOUCH_SRC)) as unknown[]) : [];
          const txt = await page.evaluate(() => document.body.innerText);
          const notFound = /Không tìm thấy trang/.test(txt);
          const blocked = /Bạn không có quyền (mở trang|xem màn) này/.test(txt);
          const forb = role === "student" ? (txt.match(new RegExp(FORBIDDEN.source, "gi")) ?? []).length : 0;
          const pass = a.ox <= 0 && a.cut.length === 0 && a.ell.length === 0 && forb === 0 && !notFound && touch.length === 0;
          rows.push({ role, route: r, w: vp.tag, pass, ox: a.ox, cut: a.cut.length, ell: a.ell.length, touch: touch.length, forb, notFound, blocked, url: new URL(page.url()).pathname });
        } catch (e) {
          rows.push({ role, route: r, w: vp.tag, pass: false, err: String(e).slice(0, 80) });
        }
      }
    }
    await page.setViewportSize({ width: 1440, height: 900 });
    writeFileSync(`/tmp/qc2-i/res/r4ui-audit-${role}.json`, JSON.stringify(rows));
    const fails = rows.filter((x) => !x.pass);
    note(`AUDIT ${role}: ${rows.length - fails.length}/${rows.length} PASS; FAIL ${fails.length}; FORBIDDEN=${rows.reduce((s, x) => s + Number(x.forb ?? 0), 0)}; ` + fails.map((x) => `${x.route}@${x.w}${x.notFound ? "(404)" : ""}${x.err ? "(err)" : ""}:${JSON.stringify({ ox: x.ox, cut: x.cut, ell: x.ell, touch: x.touch, forb: x.forb })}`).join(" | "));
    expect(fails.length).toBe(0);
  });

  test(`[TC-P212-21] ma trận vai × route (tương đương proto-curl matrix) · ${role}`, async ({ page }, info) => {
    test.skip(info.project.name !== "desktop", "chỉ desktop");
    test.setTimeout(200_000);
    await apiLogin(page, `${ACCT[role]}@edupilot.local`, SEED_PW);
    const out: Array<{ route: string; want: string; got: string }> = [];
    for (const [g, routes] of Object.entries(G)) {
      for (const r of routes) {
        const want = ALLOW[role].includes(g) ? "MO" : "CHAN";
        await page.goto(r);
        await settle(page, 1200);
        const txt = await page.evaluate(() => document.body.innerText);
        out.push({ route: r, want, got: /Bạn không có quyền (mở trang|xem màn) này/.test(txt) ? "CHAN" : "MO" });
      }
    }
    const bad = out.filter((x) => x.want !== x.got);
    writeFileSync(`/tmp/qc2-i/res/r4ui-matrix-${role}.json`, JSON.stringify(out));
    note(`matrix ${role}: ${out.length - bad.length}/${out.length} đúng; sai: ${bad.map((b) => `${b.route} want ${b.want} got ${b.got}`).join("; ")}`);
    expect(bad).toEqual([]);
  });
}
