// Máy chủ gateway GIẢ của QC cho US-PU-03 (Q-type). Dùng trong Eval: const {FAKE}=await import('/abs/pu03-fake-api.mjs'); FAKE.start(3311)
// FAKE.script({'GET /api/v1/r':[{status:503,body:{code:'SERVICE_UNAVAILABLE',message:'m',trace_id:'a'.repeat(32)}},{body:{ok:1}}]})
// step: {status,body,headers,delay,destroy,raw,ctype}; hết kịch bản → 200 {"ok":true}. FAKE.log: [{t,method,path,search,headers,body}].
// SSE: GET /api/v1/events đọc FAKE.sse (mảng kịch bản mỗi lần nối): [{frames:'id: 1-0\ndata: x\n\n', hold:true, closeAfter:ms, status:429, body:{…}}]
export const FAKE = { srv: null, log: [], scripts: {}, sse: [], sseConns: [], sseClosed: [] };
const CORS = { 'access-control-allow-origin': 'http://localhost:3400', 'access-control-allow-credentials': 'true', 'access-control-allow-headers': 'authorization,content-type,x-request-id,idempotency-key,if-none-match,last-event-id,accept', 'access-control-allow-methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS', 'access-control-expose-headers': 'etag,retry-after,idempotent-replayed,x-request-id' };
FAKE.reset = () => { FAKE.log.length = 0; FAKE.scripts = {}; FAKE.sse = []; FAKE.sseConns.length = 0; FAKE.sseClosed.length = 0; };
FAKE.script = (s) => { for (const [k, v] of Object.entries(s)) FAKE.scripts[k] = [...v]; };
FAKE.start = (port = 3311) => {
  FAKE.srv = Bun.serve({ port, idleTimeout: 120, async fetch(req) {
    const url = new URL(req.url); const t = Date.now();
    if (req.method === 'OPTIONS') return new Response(null, { status: 204, headers: CORS });
    const headers = Object.fromEntries(req.headers); const body = await req.text().catch(() => '');
    FAKE.log.push({ t, method: req.method, path: url.pathname, search: url.search, headers, body });
    if (url.pathname === '/api/v1/events') {
      const step = FAKE.sse.shift() || { hold: true, frames: 'event: ready\ndata: {}\n\n' };
      FAKE.sseConns.push({ t, headers });
      if (step.status) return new Response(JSON.stringify(step.body || { code: 'X', message: 'm', trace_id: 'a'.repeat(32) }), { status: step.status, headers: { ...CORS, 'content-type': 'application/json', ...(step.headers || {}) } });
      const enc = new TextEncoder(); let timer;
      return new Response(new ReadableStream({ start(c) { if (step.frames) c.enqueue(enc.encode(step.frames)); if (step.closeAfter !== undefined) timer = setTimeout(() => { try { c.close(); } catch {} }, step.closeAfter); req.signal.addEventListener('abort', () => { clearTimeout(timer); FAKE.sseClosed.push({ t: Date.now() }); }); } }), { headers: { ...CORS, 'content-type': 'text/event-stream', 'cache-control': 'no-cache' } });
    }
    const key = `${req.method} ${url.pathname}`; const q = FAKE.scripts[key]; const step = q && q.length ? q.shift() : { body: { ok: true } };
    if (step.delay) await Bun.sleep(step.delay);
    if (step.destroy) return new Response(new ReadableStream({ start(c) { c.error(new Error('x')); } }), { headers: CORS });
    const raw = step.raw !== undefined ? step.raw : JSON.stringify(step.body ?? {});
    return new Response(step.status === 304 ? null : raw, { status: step.status || 200, headers: { ...CORS, 'content-type': step.ctype || 'application/json', ...(step.headers || {}) } });
  } });
  return FAKE;
};
FAKE.stop = () => FAKE.srv?.stop(true);
