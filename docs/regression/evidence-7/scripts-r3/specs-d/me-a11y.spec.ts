import { test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { go, FX, log, newUser, shot } from "../specs/lib";
const FM = FX("fx_me") as { M: string }; const FP = FX("fx_p5") as { B: string; Cc: string };
const R = (tc: string, ok: boolean, note: string) => log("p506d", `${ok ? "PASS" : "FAIL"} ${tc} ${note}`);
test("17 axe /me", async ({ browser }) => {
  const rows: string[] = []; let bad = 0; let n = 0;
  for (const [u, cid, label] of [["qc3me01", FM.M, "đủ dữ liệu"], ["qc3me03", FM.M, "tuần thấp"], ["qc3me04", FM.M, "chưa dữ liệu"], ["qc3p5s04", FP.B, "lớp chưa điểm danh"]] as const) {
    for (const w of [1440, 390]) {
      const x = await newUser(browser, u, { width: w, height: w > 700 ? 900 : 844 }); await go(x.page, `/me?course=${cid}`); await x.page.getByText("Kết quả của tôi").first().waitFor({ timeout: 30000 }); await x.page.waitForTimeout(1500);
      const res = await new AxeBuilder({ page: x.page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze(); n++;
      const sc = res.violations.filter((v) => v.impact === "critical" || v.impact === "serious" || v.id === "color-contrast");
      bad += sc.length; rows.push(`${label}@${w}: vi phạm ${res.violations.length} (nghiêm trọng/serious/contrast ${sc.length}${sc.length ? " " + sc.map((v) => v.id + "×" + v.nodes.length).join(",") : ""}), pass ${res.passes.length}`);
      await x.ctx.close();
    }
  }
  R("TC-P5-06-17-axe", bad === 0, `axe (wcag2a/2aa/21a/21aa) trên /me thật ep-r, ${n} lượt (4 trạng thái × 1440/390): ${rows.join("; ")}`);
});
