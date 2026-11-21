import { expect, test } from "@playwright/test";
import { clearRL, em, installRec, note, PWD } from "./r4ui-lib";

// SmkT2: TC-P202-41 (open redirect khi đăng nhập thật qua màn /login?next=…) trên ep-r, tài khoản qc2iu.gv1
installRec("smkt2-next");
const EVIL = ["//evil.example", "https://evil.example", "/\\evil.example", "javascript:alert(1)", "/%2F%2Fevil.example", "/%5Cevil.example", "/foo%0d%0aSet-Cookie:x=1", "/a:b", "data:text/html,x", "///evil.example", " //evil.example"];
const GOOD: Array<[string, string]> = [["/threads", "/threads"], ["/class/x?tab=a", "/class/x?tab=a"], ["/settings/llm", "/settings/llm"], ["/ok?x=//evil", "/ok?x=//evil"]];

test("[TC-P202-41] next độc hại → /, hợp lệ → đúng đường dẫn, không header injection", async ({ browser }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(240_000);
  const out: string[] = [];
  const run = async (next: string) => {
    const ctx = await browser.newContext({ ignoreHTTPSErrors: true, locale: "vi-VN" });
    const page = await ctx.newPage();
    const external: string[] = [];
    page.on("request", (r) => { if (!r.url().startsWith("https://localhost:1143") && !r.url().startsWith("data:") && !r.url().startsWith("blob:")) external.push(r.url()); });
    clearRL();
    await page.goto("/login?next=" + encodeURIComponent(next));
    await page.getByLabel("Email").fill(em("gv1"));
    await page.getByLabel("Mật khẩu", { exact: true }).fill(PWD);
    await page.getByRole("button", { name: "Đăng nhập", exact: true }).click();
    await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
    await page.waitForTimeout(1200);
    const u = new URL(page.url());
    const cookies = (await ctx.cookies()).map((c) => c.name);
    await ctx.close();
    return { origin: u.origin, path: u.pathname + u.search, cookies, external };
  };
  for (const n of EVIL) {
    const r = await run(n);
    out.push(`${JSON.stringify(n)} → ${r.path}`);
    expect(r.origin, n).toBe("https://localhost:1143");
    expect(r.path, n).toBe("/");
    expect(r.cookies.includes("x"), n).toBe(false);
    expect(r.external, n).toEqual([]);
  }
  for (const [n, want] of GOOD) {
    const r = await run(n);
    out.push(`${JSON.stringify(n)} → ${r.path}`);
    expect(r.origin).toBe("https://localhost:1143");
    expect(r.path, n).toBe(want);
  }
  note(`${EVIL.length} giá trị độc hại đều về «/», ${GOOD.length} hợp lệ giữ đường dẫn: ${out.join(" ; ")}`);
});
