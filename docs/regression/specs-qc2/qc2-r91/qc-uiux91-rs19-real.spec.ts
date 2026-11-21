/* eslint-disable @typescript-eslint/no-explicit-any */
// SmkR (qc2, lượt 6): US-UX91-19 /threads — mỗi TC một khẳng định thật (thay các dòng INFO của rw5b).
import { test } from "@playwright/test";
import { api, courseId, expect, noOverflow, open } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
const D = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop");

type Row = { id: string; title: string; preview: string; tags: string[]; week_no: number | null; author: { full_name: string; role: string; is_me: boolean }; answer_state: string | null; reply_count: number; last_activity_at: string; created_at: string; pinned?: boolean; hidden?: boolean; ai_state?: string };
const mk = (i: number, title: string, preview: string, st: string | null): Row => ({ id: `00000000-0000-4000-8000-${String(i).padStart(12, "0")}`, title, preview, tags: [], week_no: 3, author: { full_name: `Tác giả ${i}`, role: "STUDENT", is_me: false }, answer_state: st, reply_count: 0, last_activity_at: new Date(Date.now() - 3 * 3600_000).toISOString(), created_at: new Date(Date.now() - 3 * 3600_000).toISOString() });
const FIX_PREV = ["- - Hướng dẫn nộp bài", "· Phisher - Individual uses email…", "* Ví dụ", "# Tiêu đề lớn", "> trích dẫn", "- - - ·", "Dòng một của đoạn dài.\nDòng hai " + "chữ ".repeat(80)];
const fixture = (pending: number): Row[] => {
  const r: Row[] = [];
  for (let i = 0; i < pending; i++) r.push(mk(i + 1, `T${i + 1}`, FIX_PREV[i] ?? `Nội dung thường ${i + 1}`, "PENDING"));
  r.push(mk(101, "T8 xác nhận", "đã xác nhận", "VERIFIED"), mk(102, "T9 sửa", "đã sửa", "CORRECTED"), mk(103, "T10 loại", "đã loại", "REJECTED"), mk(104, "T11 rỗng", "", "PENDING"));
  return r;
};
const mock = (page: any, rows: Row[]) => page.route(/\/api\/v1\/courses\/[0-9a-f-]+\/threads(\?.*)?$/, (r: any) => (r.request().method() === "GET" ? r.fulfill({ json: { items: rows, pinned: [], next_cursor: null } }) : r.continue()));
const rowsOf = (page: any) => page.locator("[data-part=thread-list] > li");
const read = (page: any) => rowsOf(page).evaluateAll((els: Element[]) => els.map((e) => ({ title: (e.querySelector("a span") as HTMLElement | null)?.innerText ?? "", prev: (e.querySelector("p") as HTMLElement | null)?.innerText ?? "", meta: (e.querySelector("[class*=meta]") as HTMLElement | null)?.innerText.replace(/\n/g, " | ") ?? "" })));

test("TC-UX19-1/2/3/9 GV: đoạn trích sạch, nhãn đa số (17/21 ≥ 80 %), đoạn rỗng", async ({ browser }, info) => {
  D(info); test.setTimeout(180_000);
  const o = await open(browser, "teacher");
  // 17 «Chờ xác nhận» + T8,T9,T10 = 20 → 17/21 (kèm T11 rỗng PENDING = 18/21 = 85,7 %)
  await mock(o.page, fixture(17));
  await o.page.goto("/threads"); await rowsOf(o.page).first().waitFor({ timeout: 30_000 });
  const d = await read(o.page);
  const T = (n: number) => d.find((x: any) => x.title === `T${n}`)!;
  // UX19-1
  for (let n = 1; n <= 5; n++) expect(T(n).prev, `T${n} không bắt đầu bằng ký tự markdown`).not.toMatch(/^[\s\-*·#>•–—]/);
  expect(T(1).prev).toBe("Hướng dẫn nộp bài");
  // UX19-2
  expect(T(7).prev).not.toMatch(/\n/); expect(T(7).prev.length).toBeLessThanOrEqual(140); expect(T(7).prev.endsWith("…")).toBe(true);
  expect(T(7).prev.startsWith("Dòng một của đoạn dài. Dòng hai")).toBe(true);
  // UX19-9: T6 (chỉ ký tự markdown) và rỗng: có tiêu đề, không dòng trống / ký tự thừa
  const raw = await rowsOf(o.page).evaluateAll((els: Element[]) => els.filter((e) => /T6|T11/.test((e.querySelector("a span") as HTMLElement)?.innerText ?? "")).map((e) => ({ t: (e.querySelector("a span") as HTMLElement).innerText, p: [...e.querySelectorAll("p")].map((p) => p.innerText) })));
  console.log("INFO UX19-9", JSON.stringify(raw));
  for (const r of raw) { expect(r.t.length).toBeGreaterThan(0); for (const p of r.p) { expect(p.trim().length, "không dòng trống").toBeGreaterThan(0); expect(p, "không dòng chỉ ký tự thừa").toMatch(/[\p{L}\p{N}]/u); } }
  // UX19-3
  const txt = await o.page.locator("main").innerText();
  const n = +(txt.match(/(\d+) câu chờ xác nhận/)?.[1] ?? -1);
  expect(n, "dòng đầu «{N} câu chờ xác nhận», N đúng").toBe(d.length - 3);
  const pendRows = d.filter((x: any) => /^T\d+$/.test(x.title) || x.title === "T11 rỗng");
  expect(pendRows.filter((x: any) => x.meta.includes("Chờ xác nhận")).length, "nhãn «Chờ xác nhận» không hiện trên thread đa số").toBe(0);
  expect(d.find((x: any) => x.title === "T10 loại")!.meta).toContain("Đã loại");
  // UX19-4 (GV)
  expect(d.find((x: any) => x.title === "T8 xác nhận")!.meta).toContain("Đã được giảng viên xác nhận");
  expect(d.find((x: any) => x.title === "T9 sửa")!.meta).toContain("Đã sửa bởi giảng viên");
  await o.ctx.close();
});

test("TC-UX19-4/5/13 SV + GV: < 80 % thì mọi thread có nhãn; nhãn xác nhận trên từng thread; h1", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  for (const u of ["teacher", "sv.kha", "ta"]) {
    const o = await open(browser, u);
    await mock(o.page, fixture(10));
    await o.page.goto("/threads"); await rowsOf(o.page).first().waitFor({ timeout: 30_000 });
    const d = await read(o.page);
    const txt = await o.page.locator("main").innerText();
    // 11 PENDING / 14 = 78,6 % < 80 %
    expect(txt, `${u}: không dòng gom`).not.toMatch(/\d+ câu chờ xác nhận/);
    const pend = d.filter((x: any) => /^T\d+$|^T11/.test(x.title));
    expect(pend.length).toBe(11);
    expect(pend.filter((x: any) => x.meta.includes("Chờ xác nhận")).length, `${u}: mọi thread PENDING mang nhãn`).toBe(11);
    expect(d.find((x: any) => x.title === "T8 xác nhận")!.meta).toContain("Đã được giảng viên xác nhận");
    expect(d.find((x: any) => x.title === "T9 sửa")!.meta).toContain("Đã sửa bởi giảng viên");
    expect(await o.page.locator("main h1").allInnerTexts()).toEqual(["Hỏi đáp công khai"]);
    await o.ctx.close();
  }
});

test("TC-UX19-8/PG dữ liệu thật: thời gian đúng US-02; limit ≤ 100 + cursor, Xem thêm", async ({ browser }, info) => {
  D(info); test.setTimeout(240_000);
  const o = await open(browser, "teacher");
  const urls: string[] = [];
  o.page.on("request", (r) => { if (/\/courses\/[0-9a-f-]+\/threads\?/.test(r.url())) urls.push(new URL(r.url()).search); });
  const c88 = await courseId(o.page, "761988");
  await o.page.goto(`/threads?course=${c88}`); await rowsOf(o.page).first().waitFor({ timeout: 30_000 });
  const d = await read(o.page);
  const times = d.map((x: any) => x.meta.split(" | ").pop() as string);
  console.log("INFO UX19-8 thời gian:", JSON.stringify(times.slice(0, 8)));
  for (const t of times) { expect(t).not.toMatch(/^\d\d:\d\d \d\d-\d\d$/); expect(t).toMatch(/^(vừa xong|\d+ phút trước|\d+ giờ trước|(Hôm nay|Hôm qua|Ngày mai|Thứ (Hai|Ba|Tư|Năm|Sáu|Bảy)|Chủ Nhật)(, \d\d\/\d\d(\/\d{4})?)?, ?(· )?\d\d:\d\d|(Thứ [A-Za-zÀ-ỹ ]+|Chủ Nhật), \d\d\/\d\d(\/\d{4})? · \d\d:\d\d)$/); }
  const lim = +(new URLSearchParams(urls[0]).get("limit") ?? 0);
  console.log("INFO UX19-PG request:", JSON.stringify(urls.slice(0, 3)));
  expect(lim).toBeGreaterThan(0); expect(lim).toBeLessThanOrEqual(100);
  await o.ctx.close();
  // «Xem thêm» → cursor (dữ liệu giả hai trang, đo yêu cầu thứ hai)
  const m = await open(browser, "teacher");
  const seen: string[] = [];
  await m.page.route(/\/api\/v1\/courses\/[0-9a-f-]+\/threads(\?.*)?$/, (r: any) => {
    if (r.request().method() !== "GET") return r.continue();
    const u = new URL(r.request().url()); seen.push(u.search);
    return u.searchParams.get("cursor") ? r.fulfill({ json: { items: [mk(201, "Trang hai", "p2", "VERIFIED")], pinned: [], next_cursor: null } })
      : r.fulfill({ json: { items: [mk(1, "Trang một", "p1", "VERIFIED")], pinned: [], next_cursor: "CUR1" } });
  });
  await m.page.goto("/threads"); await rowsOf(m.page).first().waitFor({ timeout: 30_000 });
  await m.page.getByRole("button", { name: "Xem thêm" }).click();
  await expect(rowsOf(m.page)).toHaveCount(2);
  console.log("INFO UX19-PG trang hai:", JSON.stringify(seen));
  expect(seen.some((s) => new URLSearchParams(s).get("cursor") === "CUR1" && +(new URLSearchParams(s).get("limit") ?? 0) <= 100)).toBe(true);
  expect(await noOverflow(m.page)).toBe(true);
  await m.ctx.close();
});
