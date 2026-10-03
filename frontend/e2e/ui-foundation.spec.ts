import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { expect, test } from "@playwright/test";
import { navFor } from "../src/shared/shell/nav";
import { asDemo, type DemoRole } from "./support/session";

// Quét mọi route × vai của bản dựng cổng (US-PU-02 AC14): một nút primary mỗi vùng làm việc, trang không cuộn ngang.
// (AUDIT đầy đủ — cut / ell sau khi cuộn, chuyển trạng thái — là lượt chạy của QC theo audit-baseline.md; ở đây chỉ chặn tràn ngang.)
const ROLES: DemoRole[] = ["student", "ta", "teacher", "admin"];
const REGION = "main section, [role=dialog], [data-part=work-region], form";
const allow: Array<{ route: string; region: string; reason: string }> = JSON.parse(readFileSync("e2e/primary-allow.json", "utf8"));

const routes = (role: DemoRole) => [...new Set(navFor(role, true).flatMap((g) => g.items.map((i) => i.href)))];

test("primary-allow.json: tối đa 5 mục, mỗi mục đủ route / vùng / lý do", () => {
  expect(allow.length).toBeLessThanOrEqual(5);
  for (const a of allow) expect(a.route && a.region && a.reason.trim()).toBeTruthy();
});

for (const role of ROLES) {
  test(`one-primary sweep: ${role}`, async ({ page, context }) => {
    test.setTimeout(120_000);
    await asDemo(context, role);
    const over: string[] = [];
    const audit: string[] = [];
    for (const href of routes(role)) {
      await page.goto(href);
      await page.locator("main").first().waitFor();
      await page.waitForTimeout(150);
      const regions = await page.locator(REGION).evaluateAll((els) =>
        els.map((e) => ({ n: Array.from(e.querySelectorAll('[data-variant="primary"]')).filter((b) => (b as HTMLElement).offsetParent !== null).length, tag: e.tagName + (e.getAttribute("data-part") ? `[${e.getAttribute("data-part")}]` : "") })).filter((r) => r.n > 1),
      );
      for (const r of regions) if (!allow.some((a) => a.route === href && a.region === r.tag)) over.push(`${href} ${r.tag} có ${r.n} nút primary`);
      const ox = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      if (ox !== 0) audit.push(`${href}: tràn ngang ${ox}px`);
    }
    expect(over, "vùng có > 1 nút primary").toEqual([]);
    expect(audit, "tràn ngang").toEqual([]);
  });
}

test("ConfirmIrreversible: thiếu consequence ⇒ lỗi biên dịch nêu `consequence`", async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ghi tệp tạm dùng chung");
  test.setTimeout(90_000);
  const dir = "src/__tsc__";
  mkdirSync(dir, { recursive: true });
  writeFileSync(`${dir}/bad.tsx`, 'import { ConfirmIrreversible } from "@/shared/ui";\nexport const X = () => <ConfirmIrreversible open onClose={() => {}} onConfirm={() => {}} title="x" confirmLabel="Xoá" />;\n');
  try {
    let out = "";
    try {
      execFileSync("pnpm", ["exec", "tsc", "--noEmit"], { encoding: "utf8", stdio: "pipe" });
    } catch (e) {
      out = String((e as { stdout?: string }).stdout ?? "");
    }
    expect(out).toContain("consequence");
    expect(out).toContain("__tsc__/bad.tsx");
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
