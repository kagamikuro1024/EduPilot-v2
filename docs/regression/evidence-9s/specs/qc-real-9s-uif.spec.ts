import { readFileSync, writeFileSync } from "node:fs";
import { test } from "@playwright/test";
import { needReal, shot, uiLogin } from "./qc-real-lib";

needReal();
const S = JSON.parse(readFileSync("/tmp/q9s/ui.json", "utf8")) as { course: string };
const F = JSON.parse(readFileSync("/tmp/q9s/ui08f.json", "utf8")) as { sub: Record<string, string> };
test("@real UIF 08-11 bản cũ sau khi SV nộp lại", async ({ page }) => {
  const out: Record<string, unknown> = {};
  await page.addInitScript((c) => localStorage.setItem("ep:ui:course", c), S.course);
  await uiLogin(page, "teacher");
  await page.goto(`/grading/${F.sub["sv33"]}?course=${S.course}`); await page.locator("main").waitFor(); await page.waitForTimeout(4000);
  const btn = page.getByRole("button", { name: "Duyệt bài" });
  out.bannerOnLoad = (await page.locator("main").innerText()).split("\n").filter((l) => /nộp lại|phiên bản mới/.test(l)); out.approveDisabledOnLoad = await btn.isDisabled().catch(() => null);
  if (!(out.approveDisabledOnLoad)) { await btn.click(); await page.waitForTimeout(2500); out.afterClickBanner = (await page.locator("main").innerText()).split("\n").filter((l) => /nộp lại|phiên bản mới/.test(l)); out.approveDisabledAfter = await btn.isDisabled().catch(() => null); }
  out.links = await page.locator("main a").allInnerTexts(); await shot(page, "p708-superseded");
  writeFileSync("/tmp/q9s/uif.json", JSON.stringify(out, null, 1));
});
