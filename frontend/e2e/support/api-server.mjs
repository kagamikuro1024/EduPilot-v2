// Gateway GIẢ cho test lớp dữ liệu (US-PU-03). Mỗi route có một "kịch bản" (hàng đợi phản hồi, phản hồi cuối lặp lại);
// mọi request đều được ghi lại (thời điểm, header) để test đếm / đo khoảng cách. Điều khiển qua /__ctl/*.
import http from "node:http";

const PORT = Number(process.env.API_PORT ?? 3312);
/** @type {Map<string, any[]>} */
const scripts = new Map();
/** @type {any[]} */
let log = [];
const sseClosed = [];

const send = (res, status, body, headers = {}) => {
  res.writeHead(status, { "Content-Type": "application/json", ...headers });
  res.end(typeof body === "string" ? body : JSON.stringify(body));
};

const cors = (req, res) => {
  const o = req.headers.origin;
  if (o) {
    res.setHeader("Access-Control-Allow-Origin", o);
    res.setHeader("Access-Control-Allow-Credentials", "true");
    res.setHeader("Vary", "Origin");
    res.setHeader("Access-Control-Allow-Headers", "authorization,content-type,x-request-id,idempotency-key,if-none-match,if-match,last-event-id,accept");
    res.setHeader("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS");
    res.setHeader("Access-Control-Expose-Headers", "etag,retry-after,idempotent-replayed,x-request-id");
  }
};

const readBody = (req) => new Promise((r) => { let b = ""; req.on("data", (c) => (b += c)); req.on("end", () => r(b)); });

http
  .createServer(async (req, res) => {
    cors(req, res);
    const url = new URL(req.url, "http://x");
    if (req.method === "OPTIONS") return res.writeHead(204).end();

    if (url.pathname === "/__ctl/ping") return send(res, 200, { ok: true });
    if (url.pathname === "/__ctl/reset") { scripts.clear(); log = []; sseClosed.length = 0; return send(res, 200, {}); }
    if (url.pathname === "/__ctl/script") { // { "GET /api/v1/x": [ {status, body, headers, delay, destroy, sse, hold} ] }
      const s = JSON.parse(await readBody(req));
      for (const [k, v] of Object.entries(s)) scripts.set(k, v);
      return send(res, 200, {});
    }
    if (url.pathname === "/__ctl/log") return send(res, 200, { log, sseClosed });

    const body = await readBody(req);
    const key = `${req.method} ${url.pathname}`;
    const entry = { t: Date.now(), method: req.method, path: url.pathname, search: url.search, headers: req.headers, body };
    log.push(entry);

    const q = scripts.get(key);
    if (!q) return send(res, 404, { code: "NOT_FOUND", message: "Không tìm thấy.", trace_id: "0".repeat(32) });
    const n = log.filter((l) => l.method === req.method && l.path === url.pathname).length;
    const step = q[Math.min(n - 1, q.length - 1)];
    if (step.delay) await new Promise((r) => setTimeout(r, step.delay));
    if (step.destroy) return req.socket.destroy();

    if (step.sse !== undefined) {
      res.writeHead(step.status ?? 200, { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" });
      res.write(step.sse);
      res.on("close", () => sseClosed.push({ t: Date.now(), n }));
      if (!step.hold) setTimeout(() => res.end(), step.closeAfter ?? 20);
      return;
    }
    const headers = { ...(step.headers ?? {}) };
    if (step.status === 304) return res.writeHead(304, headers).end();
    return send(res, step.status ?? 200, step.body ?? {}, headers);
  })
  .listen(PORT, () => console.log(`api-server giả: :${PORT}`));

// Máy chủ giả không được chết giữa lượt chạy vì một socket bị huỷ (kịch bản destroy / hold): ghi lại và sống tiếp.
process.on("uncaughtException", (e) => console.error("api-server giả: bỏ qua lỗi", e?.code ?? e?.message));
