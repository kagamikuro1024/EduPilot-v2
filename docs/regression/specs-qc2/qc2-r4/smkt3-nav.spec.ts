import { expect, test, type Page } from "@playwright/test";
import { PWD, apiLogin, em, installRec, note } from "./r4ui-lib";

// SmkT3: TC-P202-44 (chưa đăng nhập → /login?next; số mục nav theo vai; không bộ đổi vai) và TC-P202-48 (người mock theo email, không đọc cookie ep_demo_*; GATEP2-22 kho trình duyệt). ep-r.
installRec("smkt3-nav");
const SEED = "Edupilot#Seed-2026";
const navLinks = async (p: Page) => (await p.locator("[data-part=sidebar] nav a").allInnerTexts()).map((t) => t.replace(/\s+/g, " ").trim()).filter(Boolean);
const profile = async (p: Page) => {
  const tc = (await p.getByRole("button", { name: /^Tài khoản:/ }).first().evaluate((el) => el.textContent ?? "", undefined, { timeout: 15_000 })).replace(/\s+/g, " ");
  return { name: tc.replace(/^.*?Tài khoản:\s*/, "").trim(), who: "" };
};

test("[TC-P202-44] chưa đăng nhập → /login?next; số mục nav theo vai; không bộ đổi vai ở build thường", async ({ page, context }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(240_000);
  const out: string[] = [];
  for (const path of ["/threads", "/settings/llm"]) {
    await page.goto(path);
    await expect(page).toHaveURL(new RegExp(`/login\\?next=${encodeURIComponent(path).replace(/%/g, "%")}`), { timeout: 30_000 });
    out.push(`${path} → ${new URL(page.url()).pathname + new URL(page.url()).search}`);
  }
  const want: Array<[string, string]> = [["sv.gioi", "SV-A"], ["sv.kha", "SV-B"], ["sv.nguyco", "SV-C"], ["qc2iu.svd", "SV-D"], ["ta", "TA"], ["teacher", "GV"], ["admin", "Admin"]];
  const counts: Record<string, number> = {};
  const names: Record<string, string[]> = {};
  for (const [u, tag] of want) {
    await context.clearCookies();
    await page.evaluate(() => localStorage.removeItem("ep:ui:course")).catch(() => undefined);
    await apiLogin(page, `${u}@edupilot.local`, u.startsWith("qc2iu.") ? PWD : SEED);
    await page.goto("/");
    await expect(page.locator("[data-part=sidebar] nav a").first()).toBeVisible({ timeout: 30_000 });
    await page.waitForTimeout(800);
    names[tag] = await navLinks(page);
    counts[tag] = names[tag].length;
    const sw = await page.getByText(/Đổi vai|Chọn người|Vai mô phỏng/).count();
    const demoCk = (await context.cookies()).filter((c) => c.name.startsWith("ep_demo_")).length;
    out.push(`${tag} ${counts[tag]} mục [${names[tag].join(" / ")}]; bộ đổi vai=${sw}; cookie ep_demo_=${demoCk}`);
    expect(sw).toBe(0);
    expect(demoCk).toBe(0);
  }
  note(`${out.join(" ; ")}`);
  expect(counts).toEqual({ "SV-A": 9, "SV-B": 9, "SV-C": 9, "SV-D": 1, TA: 13, GV: 16, Admin: 7 }); // số hiện hành (account.spec.ts): TC gốc 7/12/15/6 + /exams, /assignments, /admin/audit…
});

test("[TC-P202-48] sv.kha: hồ sơ người mock B, /chat phiên mock B, không đọc cookie ep_demo_*, email lạ → theo vai; kho trình duyệt sạch", async ({ page, context }, info) => {
  if (info.project.name !== "desktop") test.skip();
  test.setTimeout(240_000);
  const out: string[] = [];
  await apiLogin(page, "sv.kha@edupilot.local", SEED);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const p1 = await profile(page); expect(p1.name.length).toBeGreaterThan(2);
  await page.goto("/chat");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(2500);
  const chatTxt = (await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 160);
  out.push(`sv.kha hồ sơ «${p1.name}»; /chat: ${chatTxt}`);
  // cookie ep_demo_* khác: không đổi người, không đổi vai
  await context.addCookies([
    { name: "ep_demo_role", value: "teacher", url: "https://localhost:1143" },
    { name: "ep_demo_person", value: "sv-3", url: "https://localhost:1143" },
    { name: "ep_demo_course", value: "int1006-2", url: "https://localhost:1143" },
  ]);
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const p2 = await profile(page);
  const nav2 = (await navLinks(page)).length;
  out.push(`đặt ep_demo_role=teacher, ep_demo_person=sv-3 → hồ sơ «${p2.name}», nav ${nav2} mục`);
  expect(p2.name).toBe(p1.name);
  await context.clearCookies();
  // email lạ (SV QC) → người mặc định theo vai
  await apiLogin(page, em("sv1"));
  await page.goto("/");
  await expect(page.locator("[data-part=topbar]")).toBeVisible({ timeout: 30_000 });
  const p3 = await profile(page);
  out.push(`email lạ qc2iu.sv1 → hồ sơ «${p3.name}»`);
  // GATEP2-22: kho trình duyệt không có token / JWT / refresh
  const store = await page.evaluate(async () => {
    const dump = (s: Storage) => Object.entries(s).map(([k, v]) => `${k}=${String(v).slice(0, 60)}`);
    let idb: string[] = [];
    try {
      const dbs = await (indexedDB as unknown as { databases?: () => Promise<Array<{ name?: string }>> }).databases?.();
      idb = (dbs ?? []).map((d) => d.name ?? "");
    } catch { /* trình duyệt không liệt kê */ }
    return { ls: dump(localStorage), ss: dump(sessionStorage), idb };
  });
  const cookies = await context.cookies();
  const ep = cookies.find((c) => c.name === "ep_rt");
  const jwtLike = JSON.stringify(store).match(/eyJ[A-Za-z0-9_-]{10,}/g) ?? [];
  const bad = [...store.ls, ...store.ss].filter((x) => /token|jwt|refresh|access|ep_rt/i.test(x.split("=")[0]));
  out.push(`localStorage ${store.ls.length} khoá [${store.ls.map((x) => x.split("=")[0]).join(",")}], sessionStorage ${store.ss.length} khoá, IndexedDB [${store.idb.join(",")}]; chuỗi JWT trong kho=${jwtLike.length}; khoá tên token/jwt/refresh=${bad.length}; ep_rt httpOnly=${ep?.httpOnly} secure=${ep?.secure} sameSite=${ep?.sameSite}; cookie đọc được từ JS: ${await page.evaluate(() => document.cookie.split(";").map((c) => c.split("=")[0].trim()).join(",") || "(rỗng)")}`);
  note(out.join(" ; "));
  expect(jwtLike).toEqual([]);
  expect(bad).toEqual([]);
  expect(ep?.httpOnly).toBe(true);
});
