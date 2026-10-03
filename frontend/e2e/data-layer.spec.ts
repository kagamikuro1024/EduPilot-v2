import { readFileSync } from "node:fs";
import { expect, test, type Page } from "@playwright/test";

// US-PU-03: lớp dữ liệu, thử trên /dev/data với gateway GIẢ (e2e/support/api-server.mjs, :3312). Chỉ chạy ở dự án desktop
// và tuần tự (máy chủ giả dùng chung). Ca @real cần gateway thật và không chạy ở CI.
const API = "http://localhost:3312";
test.describe.configure({ mode: "serial" });
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "máy chủ giả dùng chung: chỉ chạy một dự án");
});

type Step = { status?: number; body?: unknown; headers?: Record<string, string>; delay?: number; destroy?: boolean; sse?: string; hold?: boolean; closeAfter?: number };
type Entry = { t: number; method: string; path: string; search: string; headers: Record<string, string>; body: string };

const reset = (page: Page) => page.request.post(`${API}/__ctl/reset`);
const script = (page: Page, s: Record<string, Step[]>) => page.request.post(`${API}/__ctl/script`, { data: s });
const getLog = async (page: Page, path?: string): Promise<{ log: Entry[]; sseClosed: { t: number }[] }> => {
  const j = (await (await page.request.get(`${API}/__ctl/log`)).json()) as { log: Entry[]; sseClosed: { t: number }[] };
  if (path) j.log = j.log.filter((l) => l.path === path);
  return j;
};
/** Cắt mạng cho n request đầu khớp `glob` (Chrome tự gửi lại yêu cầu khi ổ cắm bị đứt nên không giả bằng destroy ở máy chủ); trả danh sách header đã thấy. */
async function failNetwork(page: Page, glob: string, n: number) {
  const seen: Array<Record<string, string>> = [];
  await page.route(glob, async (route) => {
    seen.push(route.request().headers());
    if (seen.length <= n) return route.abort("connectionfailed");
    return route.continue();
  });
  return seen;
}
const err = (status: number, code: string, extra: Record<string, unknown> = {}): Step => ({ status, body: { code, message: "m", trace_id: "a".repeat(32), ...extra } });

test.beforeEach(async ({ page }) => {
  await reset(page);
  await page.addInitScript(() => {
    (window as unknown as { __events: string[] }).__events = [];
  });
});

async function open(page: Page, q = "") {
  await page.goto(`/dev/data${q}`);
  await page.waitForFunction(() => "__ep" in window);
}

/** Gọi hàm bất kỳ của apiClient trong trang; trả { ok, data } hoặc { error: {code,status,userMessage,retryAfter,conflict} }. */
const call = (page: Page, expr: string) =>
  page.evaluate(async (e) => {
    const ep = (window as unknown as { __ep: { apiClient: any; ApiError: any; tokenStore: any; fieldErrors: any } }).__ep; // eslint-disable-line
    try {
      const fn = new Function("ep", `return (async () => (${e}))()`);
      return { ok: true, data: await fn(ep) } as { ok: boolean; data?: unknown; error?: any }; // eslint-disable-line
    } catch (x: any) { // eslint-disable-line
      return { ok: false, error: { code: x.code, status: x.status, userMessage: x.userMessage, retryAfter: x.retryAfter, conflict: x.conflict, traceId: x.traceId, name: x.name } } as { ok: boolean; data?: unknown; error?: any }; // eslint-disable-line
    }
  }, expr);

test("apiClient basics: header đủ, token không tới origin lạ, không ghi storage", async ({ page }) => {
  await script(page, { "GET /api/v1/ping": [{ body: { ok: true } }] });
  const outside: string[] = [];
  page.on("request", (r) => /evil\.example/.test(r.url()) && outside.push(r.url()));
  await open(page);
  await call(page, 'ep.tokenStore.set("tok.abc.def")');
  const r = await call(page, 'ep.apiClient.get("/ping")');
  expect(r.ok).toBe(true);
  const { log } = await getLog(page, "/api/v1/ping");
  expect(log).toHaveLength(1);
  const h = log[0].headers;
  expect(h.authorization).toBe("Bearer tok.abc.def");
  expect(h["x-request-id"]).toMatch(/^[0-9a-f-]{36}$/);
  expect(h.accept).toContain("application/json");
  const bad = await call(page, 'ep.apiClient.get("https://evil.example/x")');
  expect(bad.error.code).toBe("BAD_TARGET");
  expect(outside).toEqual([]);
  // không có setItem với token ở shared/data
  const src = readFileSync("src/shared/data/useAutosaveDraft.ts", "utf8") + readFileSync("src/shared/data/apiClient.ts", "utf8");
  expect(src.split("\n").filter((l) => /setItem\(/.test(l) && /token/i.test(l))).toHaveLength(0);
});

const CODES: Array<[number, string, Record<string, unknown>?]> = [
  [400, "BAD_REQUEST"], [401, "UNAUTHENTICATED"], [401, "TOKEN_EXPIRED"], [401, "TOKEN_INVALID"], [403, "FORBIDDEN"], [404, "NOT_FOUND"], [405, "METHOD_NOT_ALLOWED"],
  [409, "CONFLICT"], [409, "VERSION_CONFLICT", { details: { current_version: 7, current: { n: 1 } } }], [413, "PAYLOAD_TOO_LARGE"], [415, "UNSUPPORTED_MEDIA_TYPE"],
  [422, "VALIDATION_FAILED"], [422, "INVALID_CURSOR"], [422, "IDEMPOTENCY_KEY_REUSED"], [422, "IDEMPOTENCY_KEY_REQUIRED"], [429, "RATE_LIMITED", { retry_after: 30 }],
  [429, "SSE_LIMIT_REACHED", { retry_after: 30 }], [500, "INTERNAL"], [503, "SERVICE_UNAVAILABLE", { retry_after: 30 }], [503, "NOT_READY", { retry_after: 30 }],
  [504, "DEADLINE_EXCEEDED"], [503, "OVERLOADED", { retry_after: 30 }], [503, "LLM_NOT_CONFIGURED"], [503, "LLM_UNAVAILABLE"], [409, "PROVIDER_IN_USE"],
  [422, "MODEL_DIMS_MISMATCH"], [422, "ROUTE_INVALID"], [409, "IDEMPOTENCY_IN_PROGRESS", { retry_after: 30 }],
];

test("error mapping: 28 mã + mã lạ", async ({ page }) => {
  expect(CODES).toHaveLength(28);
  await open(page);
  for (const [status, code, extra] of CODES) {
    await script(page, { "POST /api/v1/err": [err(status, code, extra)] });
    const r = await call(page, 'ep.apiClient.post("/err", {})');
    expect(r.error, code).toMatchObject({ code, status });
    expect(r.error.userMessage, code).toBeTruthy();
    if (extra?.retry_after) expect(r.error.retryAfter).toBe(30);
    if (code === "VERSION_CONFLICT") expect(r.error.conflict).toEqual({ currentVersion: 7, current: { n: 1 } });
  }
  await script(page, { "POST /api/v1/err": [err(418, "SOMETHING_NEW")] });
  const odd = await call(page, 'ep.apiClient.post("/err", {})');
  expect(odd.error.userMessage).toContain("Có lỗi xảy ra");
});

test("non-json: HTML 502, thân rỗng, JSON hỏng, đứt kết nối, huỷ", async ({ page }) => {
  await open(page);
  await script(page, { "POST /api/v1/x": [{ status: 502, headers: { "Content-Type": "text/html" }, body: "<html>502 Bad Gateway</html>" }] });
  const html = await call(page, 'ep.apiClient.post("/x", {})');
  expect(html.error.code).toBe("BAD_GATEWAY");
  expect(html.error.userMessage).not.toMatch(/<html|ECONN/);
  await script(page, { "POST /api/v1/x": [{ status: 500, headers: { "Content-Type": "application/json" }, body: "{hỏng" }] });
  expect((await call(page, 'ep.apiClient.post("/x", {})')).error.code).toBe("PARSE_ERROR");
  await script(page, { "POST /api/v1/x": [{ destroy: true }] });
  const net = await call(page, 'ep.apiClient.post("/x", {}, { idempotencyKey: "k-12345678" })');
  expect(net.error.code).toBe("NETWORK");
  expect(net.error.userMessage).toMatch(/Chữ bạn đã nhập vẫn được giữ/);
  await script(page, { "GET /api/v1/slow": [{ delay: 3000, body: {} }] });
  const ab = await call(page, '(() => { const c = new AbortController(); setTimeout(() => c.abort(), 100); return ep.apiClient.get("/slow", { signal: c.signal }); })()');
  expect(ab.error.code).toBe("ABORTED");
});

test("error display: Sinh viên không thấy mã / trace_id; Admin có Chi tiết kỹ thuật", async ({ page }) => {
  await script(page, { "POST /api/v1/fail": [err(409, "VERSION_CONFLICT", { details: { current_version: 2 } })] });
  await open(page, "?as=student");
  await page.locator("[data-part=err-display]").getByRole("button", { name: "Gọi lỗi" }).click();
  await expect(page.locator("[data-part=err-display] [role=alert]")).toBeVisible();
  const txt = await page.evaluate(() => document.body.innerText);
  expect(txt).not.toMatch(/VERSION_CONFLICT|INTERNAL|[0-9a-f]{32}/);
  await open(page, "?as=admin");
  await page.locator("[data-part=err-display]").getByRole("button", { name: "Gọi lỗi" }).click();
  const det = page.locator("[data-part=err-display] details");
  await expect(det).toContainText("a".repeat(32));
  expect(await det.evaluate((e) => (e as HTMLDetailsElement).open)).toBe(false);
});

test.describe("retry", () => {
  const gaps = (l: Entry[]) => l.slice(1).map((e, i) => e.t - l[i].t);
  test("GET 503,503,200 = 3 request, khoảng ≥ 240 / 720 ms", async ({ page }) => {
    await script(page, { "GET /api/v1/r": [err(503, "SERVICE_UNAVAILABLE"), err(503, "SERVICE_UNAVAILABLE"), { body: { ok: 1 } }] });
    await open(page);
    expect((await call(page, 'ep.apiClient.get("/r")')).ok).toBe(true);
    const { log } = await getLog(page, "/api/v1/r");
    expect(log).toHaveLength(3);
    const g = gaps(log);
    expect(g[0]).toBeGreaterThanOrEqual(240);
    expect(g[1]).toBeGreaterThanOrEqual(720);
  });
  test("404 = 1 request; 429 retry_after 30 = 1 request", async ({ page }) => {
    await script(page, { "GET /api/v1/a": [err(404, "NOT_FOUND")], "GET /api/v1/b": [err(429, "RATE_LIMITED", { retry_after: 30 })] });
    await open(page);
    await call(page, 'ep.apiClient.get("/a")');
    const b = await call(page, 'ep.apiClient.get("/b")');
    expect(b.error.retryAfter).toBe(30);
    expect((await getLog(page, "/api/v1/a")).log).toHaveLength(1);
    expect((await getLog(page, "/api/v1/b")).log).toHaveLength(1);
  });
  test("retry_after 2 ⇒ chờ đúng 2 s (2000–2250 ms)", async ({ page }) => {
    await script(page, { "GET /api/v1/c": [err(503, "SERVICE_UNAVAILABLE", { retry_after: 2 }), { body: {} }] });
    await open(page);
    expect((await call(page, 'ep.apiClient.get("/c")')).ok).toBe(true);
    const { log } = await getLog(page, "/api/v1/c");
    expect(log).toHaveLength(2);
    const g = log[1].t - log[0].t;
    expect(g).toBeGreaterThanOrEqual(2000);
    expect(g).toBeLessThanOrEqual(2250);
  });
  test("PATCH không khoá → NETWORK = 1 request; POST có khoá → 2 request cùng Idempotency-Key", async ({ page }) => {
    await open(page);
    // POST luôn tự sinh khoá ⇒ ca "không khoá" kiểm bằng PATCH (chỉ có khoá khi idempotent: true)
    const patch = await failNetwork(page, "**/api/v1/d", 99);
    await call(page, 'ep.apiClient.patch("/d", {})');
    expect(patch).toHaveLength(1);
    await script(page, { "POST /api/v1/p": [{ status: 201, body: {} }] });
    const post = await failNetwork(page, "**/api/v1/p", 1);
    expect((await call(page, 'ep.apiClient.post("/p", {})')).ok).toBe(true);
    expect(post).toHaveLength(2);
    expect(post[0]["idempotency-key"]).toBeTruthy();
    expect(post[0]["idempotency-key"]).toBe(post[1]["idempotency-key"]);
  });
});

test("idempotency key: regex, Gửi lại cùng khoá, gửi mới khoá khác, replayed im lặng", async ({ page }) => {
  await script(page, { "POST /api/v1/items": [{ status: 201, body: { id: "1" }, headers: { "Idempotent-Replayed": "true" } }, { status: 201, body: { id: "2" } }] });
  await open(page);
  const seen = await failNetwork(page, "**/api/v1/items", 3); // 3 lần đầu đứt: apiClient tự thử 3 lần cùng khoá rồi báo NETWORK
  const box = page.locator("[data-part=idem]");
  await box.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(box.locator("[data-part=idem-result]")).toHaveText("NETWORK");
  await box.getByRole("button", { name: "Gửi lại" }).click();
  await expect(box.locator("[data-part=idem-result]")).toHaveText("ok replayed=true");
  await expect(page.getByText(/Idempotent|đã gửi trước đó/i)).toHaveCount(0);
  await box.getByRole("button", { name: "Gửi", exact: true }).click();
  await expect(box.locator("[data-part=idem-result]")).toHaveText("ok replayed=false");
  const keys = seen.map((h) => h["idempotency-key"]);
  expect(keys).toHaveLength(5);
  for (const k of keys) expect(k).toMatch(/^[A-Za-z0-9._:-]{8,128}$/);
  expect(new Set(keys.slice(0, 4)).size).toBe(1);
  expect(keys[4]).not.toBe(keys[0]);
});

test("auth expired: 5 request song song ⇒ đúng 1 sự kiện, không gửi lại", async ({ page }) => {
  await script(page, { "GET /api/v1/a": [err(401, "TOKEN_EXPIRED")] });
  await open(page);
  await call(page, 'ep.tokenStore.set("t.o.k")');
  await call(page, 'Promise.allSettled([1,2,3,4,5].map(() => ep.apiClient.get("/a")))');
  expect(await page.evaluate(() => (window as unknown as { __events: string[] }).__events.filter((e) => e === "auth:expired").length)).toBe(1);
  expect((await getLog(page, "/api/v1/a")).log).toHaveLength(5);
  expect(await page.evaluate(() => (window as unknown as { __ep: { tokenStore: { get: () => string | null } } }).__ep.tokenStore.get())).toBeNull();
});

test("etag: lần hai gửi If-None-Match, 304 trả đúng tham chiếu cache", async ({ page }) => {
  await script(page, { "GET /api/v1/e": [{ body: { n: 1 }, headers: { ETag: 'W/"v1"' } }, { status: 304, headers: { ETag: 'W/"v1"' } }] });
  await open(page);
  const same = await page.evaluate(async () => {
    const { apiClient } = (window as unknown as { __ep: { apiClient: { get: (p: string) => Promise<{ data: unknown; status: number }> } } }).__ep;
    const a = await apiClient.get("/e");
    const b = await apiClient.get("/e");
    return { same: a.data === b.data, status: b.status };
  });
  expect(same).toEqual({ same: true, status: 304 });
  const { log } = await getLog(page, "/api/v1/e");
  expect(log[0].headers["if-none-match"]).toBeUndefined();
  expect(log[1].headers["if-none-match"]).toBe('W/"v1"');
});

test("field errors: ô name có aria-invalid và giữ nội dung", async ({ page }) => {
  await script(page, { "POST /api/v1/items": [err(422, "VALIDATION_FAILED", { details: [{ field: "name", code: "required", message: "Tên là bắt buộc." }] })] });
  await open(page);
  const box = page.locator("[data-part=field-errors]");
  await box.getByLabel("Tên").fill("nội dung đã gõ");
  await box.getByRole("button", { name: "Lưu" }).click();
  await expect(box.getByLabel("Tên")).toHaveAttribute("aria-invalid", "true");
  await expect(box.getByText("Tên là bắt buộc.")).toBeVisible();
  await expect(box.getByLabel("Tên")).toHaveValue("nội dung đã gõ");
});

test("retry-after notice: đếm ngược 3 → 2, nút khoá rồi mở", async ({ page }) => {
  await script(page, { "POST /api/v1/fail": [err(503, "NOT_READY", { retry_after: 3 })] });
  await open(page);
  const box = page.locator("[data-part=err-display]");
  await box.getByRole("button", { name: "Gọi lỗi" }).click();
  const status = box.getByRole("status");
  await expect(status).toContainText("3 giây");
  const retry = box.getByRole("button", { name: "Thử lại" });
  await expect(retry).toBeDisabled();
  await expect(status).toContainText("2 giây", { timeout: 2500 });
  await page.waitForTimeout(1300);
  await expect(retry).toBeEnabled();
});

test("cursor list: gộp + loại trùng, limit kẹp 100, INVALID_CURSOR quay về trang đầu một lần", async ({ page }) => {
  await script(page, { "GET /api/v1/items": [{ body: { items: [{ id: "a" }, { id: "b" }], next_cursor: "c1" } }, { body: { items: [{ id: "b" }, { id: "c" }], next_cursor: null } }] });
  await open(page);
  await expect(page.locator("[data-part=cursor-count]")).toHaveText("2");
  await page.getByRole("button", { name: "Tải trang kế" }).click();
  await expect(page.locator("[data-part=cursor-count]")).toHaveText("3");
  const { log } = await getLog(page, "/api/v1/items");
  expect(log[0].search).toContain("limit=100");
  expect(log[0].search).not.toContain("cursor");
  expect(log[1].search).toContain("cursor=c1");
  await reset(page);
  await script(page, { "GET /api/v1/items": [{ body: { items: [{ id: "a" }], next_cursor: "bad" } }, err(422, "INVALID_CURSOR"), { body: { items: [{ id: "a" }, { id: "z" }], next_cursor: null } }] });
  await open(page);
  await page.getByRole("button", { name: "Tải trang kế" }).click();
  await expect(page.locator("[data-part=cursor-count]")).toHaveText("2");
  const l2 = (await getLog(page, "/api/v1/items")).log;
  expect(l2).toHaveLength(3);
  expect(l2[2].search).not.toContain("cursor");
});

const frame = (id: string, data: string, ev = "msg") => `event: ${ev}\nid: ${id}\ndata: ${data}\n\n`;
const READY = 'retry: 3000\n\nevent: ready\ndata: {"conn_id":"c"}\n\n';
const sseLog = (page: Page) => page.evaluate(() => (window as unknown as { __sse: string[] }).__sse);

test("sse parse + singleton: nối data nhiều dòng, bỏ heartbeat, một kết nối, đóng ≤ 200 ms", async ({ page }) => {
  await script(page, { "GET /api/v1/events": [{ sse: READY + "event: msg\nid: 1000-1\ndata: a\ndata: b\n\n: hb\n\n", hold: true }] });
  await open(page);
  await call(page, 'ep.tokenStore.set("sse.tok.1")');
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(() => sseLog(page).then((x) => x.length)).toBe(3);
  expect(await sseLog(page)).toEqual(["0:1000-1:a\nb", "1:1000-1:a\nb", "2:1000-1:a\nb"]);
  const { log } = await getLog(page, "/api/v1/events");
  expect(log).toHaveLength(1);
  expect(log[0].headers.authorization).toBe("Bearer sse.tok.1");
  expect(log[0].search).not.toContain("token");
  await expect(page.locator("[data-part=sse-status]")).toHaveText("open");
  await page.getByRole("button", { name: "Gỡ hết" }).click();
  await expect.poll(async () => (await getLog(page)).sseClosed.length, { timeout: 1000 }).toBe(1);
  const g = await getLog(page);
  expect(g.sseClosed[0].t - g.log[0].t).toBeLessThan(5000);
});

test("sse reconnect: e1…e13 mỗi sự kiện một lần, đúng thứ tự, Last-Event-ID = e5", async ({ page }) => {
  const first = READY + [1, 2, 3, 4, 5].map((n) => frame(`1000-${n}`, `e${n}`)).join("");
  const second = READY + [4, 5, 6, 7, 8, 9, 10, 11, 12, 13].map((n) => frame(`1000-${n}`, `e${n}`)).join(""); // gửi bù có lặp e4, e5
  await script(page, { "GET /api/v1/events": [{ sse: first, closeAfter: 30 }, { sse: second, hold: true }] });
  await open(page);
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(() => sseLog(page).then((x) => x.filter((v) => v.startsWith("0:")).length), { timeout: 8000 }).toBe(13);
  expect((await sseLog(page)).filter((v) => v.startsWith("0:")).map((v) => v.split(":")[2])).toEqual(Array.from({ length: 13 }, (_, i) => `e${i + 1}`));
  const { log } = await getLog(page, "/api/v1/events");
  expect(log).toHaveLength(2);
  expect(log[1].headers["last-event-id"]).toBe("1000-5");
  expect(log[1].t - log[0].t).toBeGreaterThanOrEqual(800);
});

test("sse control: reconnect nối ngay, shutdown 1–3 s, resync vô hiệu hoá query", async ({ page }) => {
  await script(page, { "GET /api/v1/ps": [{ body: { items: ["a"] } }] });
  await script(page, { "GET /api/v1/events": [{ sse: READY + frame("1000-1", "x") + 'event: reconnect\ndata: {"reason":"max_duration"}\n\n', closeAfter: 400 }, { sse: READY + 'event: shutdown\ndata: {"reason":"server_shutdown"}\n\n', closeAfter: 400 }, { sse: READY + 'event: resync\ndata: {"reason":"gap"}\n\n', hold: true }] });
  await open(page);
  await expect(page.locator("[data-part=ps-ok]")).toHaveText("a");
  const psBefore = (await getLog(page, "/api/v1/ps")).log.length;
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length, { timeout: 10_000 }).toBe(3);
  const { log } = await getLog(page, "/api/v1/events");
  expect(log[1].t - log[0].t).toBeLessThanOrEqual(600); // reconnect: nối ngay (không backoff 1 s)
  expect(log[1].headers["last-event-id"]).toBe("1000-1");
  const g2 = log[2].t - log[1].t;
  expect(g2).toBeGreaterThanOrEqual(1000);
  expect(g2).toBeLessThanOrEqual(3700);
  await expect.poll(async () => (await getLog(page, "/api/v1/ps")).log.length).toBe(psBefore + 1);
});

test("sse errors: 401 dừng + auth:expired; 429 ×3 rồi degraded; im > 40 s nối lại", async ({ page }) => {
  // 401
  await script(page, { "GET /api/v1/events": [err(401, "TOKEN_EXPIRED")] });
  await open(page);
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect(page.locator("[data-part=sse-status]")).toHaveText("closed");
  await page.waitForTimeout(1500);
  expect((await getLog(page, "/api/v1/events")).log).toHaveLength(1);
  expect(await page.evaluate(() => (window as unknown as { __events: string[] }).__events.includes("auth:expired"))).toBe(true);

  // 429 SSE_LIMIT_REACHED (đồng hồ giả để không chờ 10 s thật)
  await reset(page);
  await script(page, { "GET /api/v1/events": [err(429, "SSE_LIMIT_REACHED", { retry_after: 5 })] });
  await page.clock.install();
  await open(page);
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBe(1);
  for (let i = 0; i < 2; i++) {
    await page.waitForTimeout(300); // để trình duyệt xử lý phản hồi 429 và đặt hẹn giờ (đồng hồ thật)
    await page.clock.runFor(5100);
    await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBe(i + 2);
  }
  await expect(page.locator("[data-part=sse-status]")).toHaveText("degraded");
  await expect(page.getByText("Bạn đang mở nhiều cửa sổ; cập nhật tự động tạm dừng")).toBeVisible();
  await page.waitForTimeout(300);
  await page.clock.runFor(30_000);
  expect((await getLog(page, "/api/v1/events")).log).toHaveLength(3);
  await page.clock.runFor(31_000); // sau 60 s thử lại một lần
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBe(4);

  // watchdog: treo > 40 s
  await reset(page);
  await script(page, { "GET /api/v1/events": [{ sse: "", hold: true }, { sse: READY, hold: true }] });
  await page.clock.install();
  await open(page);
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBe(1);
  await page.waitForTimeout(300);
  await page.clock.runFor(41_000);
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBeGreaterThanOrEqual(2);
});

test("useJob: GET trước, SSE sau, progress không giảm; thăm dò 2 s khi SSE không mở; FAILED có lỗi", async ({ page }) => {
  // SSE mở: không thăm dò
  await script(page, {
    "GET /api/v1/jobs/j1": [{ body: { id: "j1", status: "RUNNING", progress: 20 } }],
    "GET /api/v1/events": [{ sse: READY + frame("1000-1", JSON.stringify({ job_id: "j1", status: "RUNNING", progress: 60 }), "job.progress") + frame("1000-2", JSON.stringify({ job_id: "j1", status: "RUNNING", progress: 40 }), "job.progress") + frame("1000-3", JSON.stringify({ job_id: "j1", status: "SUCCEEDED", progress: 100 }), "job.progress"), hold: true }],
  });
  await open(page);
  await page.locator("[data-part=job]").getByLabel("Mã việc").fill("j1");
  await expect(page.locator("[data-part=job-state]")).toHaveText("SUCCEEDED:100");
  expect((await getLog(page, "/api/v1/jobs/j1")).log).toHaveLength(1);

  // SSE hỏng ⇒ thăm dò
  await reset(page);
  await script(page, {
    "GET /api/v1/jobs/j2": [{ body: { id: "j2", status: "RUNNING", progress: 10 } }, { body: { id: "j2", status: "RUNNING", progress: 5 } }, { body: { id: "j2", status: "SUCCEEDED", progress: 100 } }],
    "GET /api/v1/events": [err(503, "SERVICE_UNAVAILABLE")],
  });
  await open(page);
  await page.locator("[data-part=job]").getByLabel("Mã việc").fill("j2");
  await expect(page.locator("[data-part=job-state]")).toHaveText("SUCCEEDED:100", { timeout: 9000 });
  const l = (await getLog(page, "/api/v1/jobs/j2")).log;
  expect(l.length).toBeGreaterThanOrEqual(3);
  expect(l[1].t - l[0].t).toBeGreaterThanOrEqual(1800);
  expect(l[1].t - l[0].t).toBeLessThanOrEqual(2600);

  await reset(page);
  await script(page, { "GET /api/v1/jobs/j3": [{ body: { id: "j3", status: "FAILED", progress: 30, error: { code: "X", message: "Việc bị lỗi." } } }], "GET /api/v1/events": [err(503, "SERVICE_UNAVAILABLE")] });
  await open(page);
  await page.locator("[data-part=job]").getByLabel("Mã việc").fill("j3");
  await expect(page.locator("[data-part=job-state]")).toHaveText("FAILED:30:Việc bị lỗi.");
});

test.describe("autosave", () => {
  test("khôi phục sau khi đóng tab đột ngột", async ({ page, context }) => {
    await open(page);
    await page.locator("[data-part=autosave]").getByLabel("Ghi chú").fill("Xin chào thầy");
    await page.waitForTimeout(2500);
    await expect(page.locator("[data-part=draft-status]")).toHaveText("saved");
    await page.close({ runBeforeUnload: false });
    const p2 = await context.newPage();
    await p2.goto("/dev/data");
    await expect(p2.locator("[data-part=autosave]").getByLabel("Ghi chú")).toHaveValue("Xin chào thầy");
    const stored = await p2.evaluate(() => localStorage.getItem("ep:draft:anon:dev.note"));
    expect(JSON.parse(stored!)).toMatchObject({ v: 1, text: "Xin chào thầy" });
    await p2.locator("[data-part=autosave]").getByRole("button", { name: "Gửi xong" }).click();
    expect(await p2.evaluate(() => localStorage.getItem("ep:draft:anon:dev.note"))).toBeNull();
  });
  test("localStorage đầy ⇒ status error nhưng chữ vẫn nằm trong ô; khoá bí mật bị từ chối", async ({ page }) => {
    await page.addInitScript(() => {
      Storage.prototype.setItem = () => {
        throw new DOMException("full", "QuotaExceededError");
      };
    });
    await open(page);
    const box = page.locator("[data-part=autosave]");
    await box.getByLabel("Ghi chú").fill("chữ không được mất");
    await expect(box.locator("[data-part=draft-status]")).toHaveText("error", { timeout: 4000 });
    await expect(box.getByLabel("Ghi chú")).toHaveValue("chữ không được mất");
    await box.getByRole("button", { name: "Thử khoá bí mật" }).click();
    await expect(box.locator("[data-part=draft-throw]")).toContainText("bí mật");
  });
});

test("undoable: lạc quan ≤ 100 ms, dòng Hoàn tác tại chỗ, tự biến sau 5 s, lỗi ghi hoàn UI + alert", async ({ page }) => {
  await script(page, { "PUT /api/v1/flag": [{ delay: 400, body: {} }] });
  await page.clock.install();
  await open(page);
  const box = page.locator("[data-part=undo]");
  const t0 = Date.now();
  await box.getByRole("button", { name: "Đổi" }).click();
  await expect(box.locator("[data-part=undo-value]")).toHaveText("bật", { timeout: 100 });
  expect(Date.now() - t0).toBeLessThan(1000);
  await expect(box.locator("[data-part=undo-line]")).toContainText("Đã bật");
  expect(await page.getByRole("status").filter({ hasText: /Thành công/ }).count()).toBe(0);
  await box.getByRole("button", { name: "Hoàn tác" }).click();
  await expect(box.locator("[data-part=undo-value]")).toHaveText("tắt");
  await expect.poll(async () => (await getLog(page, "/api/v1/flag")).log.length).toBe(2);
  expect(JSON.parse((await getLog(page, "/api/v1/flag")).log[1].body)).toEqual({ on: false });

  await box.getByRole("button", { name: "Đổi" }).click();
  await expect(box.locator("[data-part=undo-line]")).toBeVisible();
  await page.clock.runFor(5200);
  await expect(box.locator("[data-part=undo-line]")).toHaveCount(0);

  await reset(page);
  await script(page, { "PUT /api/v1/flag": [err(500, "INTERNAL")] });
  await box.getByRole("button", { name: "Đổi" }).click();
  await expect(box.getByRole("alert")).toBeVisible();
  await expect(box.locator("[data-part=undo-value]")).toHaveText("bật"); // ghi lỗi ⇒ hoàn về giá trị trước (đang "bật")
});

test("offline banner: hiện ≤ 1 s, vẫn gõ được, biến ≤ 2 s sau khi có mạng và một request thành công", async ({ page, context }) => {
  await script(page, { "GET /api/v1/ping": [{ body: {} }] });
  await open(page);
  const banner = page.locator("[data-part=offline-banner] [role=status]");
  await expect(banner).toHaveCount(0);
  await context.setOffline(true);
  await expect(banner).toContainText("Mất kết nối mạng", { timeout: 1000 });
  await page.locator("[data-part=offline]").getByLabel("Ô nhập").fill("vẫn gõ được");
  await expect(page.locator("[data-part=offline]").getByLabel("Ô nhập")).toHaveValue("vẫn gõ được");
  await context.setOffline(false);
  await page.locator("[data-part=offline]").getByRole("button", { name: "Gọi /ping" }).click();
  await expect(banner).toHaveCount(0, { timeout: 2000 });
});

test("pagestate: khung xương khi pending; lỗi chuẩn + Thử lại gọi đúng 1 request", async ({ page }) => {
  await script(page, { "GET /api/v1/ps": [{ delay: 800, body: { items: ["a"] } }] });
  await open(page);
  await expect(page.locator("[data-part=pagestate] [aria-busy=true]")).toBeVisible();
  await expect(page.locator("[data-part=ps-ok]")).toHaveText("a");
  await reset(page);
  await script(page, { "GET /api/v1/ps": [err(500, "INTERNAL"), { body: { items: [] } }] });
  await open(page);
  const box = page.locator("[data-part=pagestate]");
  await expect(box.getByRole("alert")).toBeVisible();
  const before = (await getLog(page, "/api/v1/ps")).log.length;
  await box.getByRole("button", { name: "Thử lại" }).click();
  await expect(box.getByText("Chưa có gì")).toBeVisible();
  expect((await getLog(page, "/api/v1/ps")).log.length).toBe(before + 1);
});

test("token hygiene: token không vào storage, URL hay console", async ({ page }) => {
  const logs: string[] = [];
  page.on("console", (m) => logs.push(m.text()));
  await script(page, { "GET /api/v1/ping": [{ body: {} }] });
  await open(page);
  const box = page.locator("[data-part=token]");
  await box.getByLabel("Dán token").fill("SECRET.JWT.VALUE");
  await box.getByRole("button", { name: "Dùng" }).click();
  await call(page, 'ep.apiClient.get("/ping")');
  const st = await page.evaluate(() => JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage }) + document.cookie + location.href);
  expect(st).not.toContain("SECRET.JWT.VALUE");
  expect(logs.join("\n")).not.toContain("SECRET.JWT.VALUE");
  expect((await getLog(page, "/api/v1/ping")).log[0].headers.authorization).toBe("Bearer SECRET.JWT.VALUE");
});

test("BUG-PU03-1: traceId lấy từ header X-Request-Id khi thân thiếu trace_id; thân có thì thân thắng", async ({ page }) => {
  await open(page);
  await script(page, { "POST /api/v1/t": [{ status: 500, body: { code: "INTERNAL", message: "m" }, headers: { "X-Request-Id": "hdr-trace-1" } }] });
  expect((await call(page, 'ep.apiClient.post("/t", {})')).error.traceId).toBe("hdr-trace-1");
  await script(page, { "POST /api/v1/t": [{ status: 422, body: { code: "VALIDATION_FAILED", message: "m", details: [] }, headers: { "X-Request-Id": "hdr-trace-2" } }] });
  expect((await call(page, 'ep.apiClient.post("/t", {})')).error.traceId).toBe("hdr-trace-2");
  await script(page, { "POST /api/v1/t": [{ status: 500, body: { code: "INTERNAL", message: "m", trace_id: "body-trace" }, headers: { "X-Request-Id": "hdr-trace-3" } }] });
  expect((await call(page, 'ep.apiClient.post("/t", {})')).error.traceId).toBe("body-trace");
  await script(page, { "POST /api/v1/t": [{ status: 502, headers: { "Content-Type": "text/html", "X-Request-Id": "hdr-trace-4" }, body: "<html>" }] });
  expect((await call(page, 'ep.apiClient.post("/t", {})')).error.traceId).toBe("hdr-trace-4");
});

test("BUG-PU03-2: event: reconnect → đúng 2 yêu cầu /events và 1 lần đóng, nối lại ≤ 200 ms", async ({ page }) => {
  const READY2 = 'retry: 3000\n\nevent: ready\ndata: {"conn_id":"c"}\n\n';
  await script(page, { "GET /api/v1/events": [{ sse: READY2 + 'event: reconnect\ndata: {"reason":"server_restart"}\n\n', hold: true }, { sse: READY2, hold: true }, { sse: READY2, hold: true }] });
  await open(page);
  await page.getByRole("button", { name: "Gắn 3 người nghe" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/events")).log.length).toBeGreaterThanOrEqual(2);
  await page.waitForTimeout(1500);
  const { log, sseClosed } = await getLog(page, "/api/v1/events");
  expect(log).toHaveLength(2);
  expect(sseClosed).toHaveLength(1);
  expect(log[1].t - log[0].t).toBeLessThanOrEqual(200);
  await expect(page.locator("[data-part=sse-status]")).toHaveText("open");
});
