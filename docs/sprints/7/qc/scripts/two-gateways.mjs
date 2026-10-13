import http from "node:http";
const [A, B, tt] = process.argv.slice(2, 5); const pairs = process.argv.slice(5).reduce((a, v, i, r) => (i % 2 ? a : [...a, [v, r[i + 1]]]), []);
function sse(host, token, label) { return new Promise((resolve) => { const frames = []; const req = http.request({ host, port: 8080, path: "/api/v1/events", headers: { Authorization: `Bearer ${token}`, Accept: "text/event-stream" } }, (res) => { res.on("data", (c) => { for (const m of c.toString().matchAll(/event: (\w+)\ndata: (.*)/g)) frames.push([Date.now(), m[1], m[2]]); }); resolve({ frames, close: () => req.destroy(), status: res.statusCode }); }); req.on("error", (e) => resolve({ frames, err: e.message, close() {} })); req.end(); }); }
function post(host, token, mid) { return new Promise((resolve) => { const t0 = Date.now(); const req = http.request({ host, port: 8080, path: `/api/v1/chat/messages/${mid}/escalate`, method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" } }, (res) => { res.resume(); res.on("end", () => resolve([res.statusCode, t0])); }); req.on("error", (e) => resolve([e.message, t0])); req.end(); }); }
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const tests = [["nối A, sinh ở B", A, B, pairs[0]], ["nối B, sinh ở A", B, A, pairs[1]]];
for (const [name, listen, produce, [stoken, mid]] of tests) {
  const s = await sse(listen, tt, name); await sleep(1500); const before = s.frames.length;
  const [code, t0] = await post(produce, stoken, mid); await sleep(4000);
  const got = s.frames.slice(before).filter((f) => f[1] === "notification");
  console.log(`${name}: POST=${code}, nhận ${got.length} khung notification, độ trễ ${got[0] ? got[0][0] - t0 : "-"} ms (SSE gateway ${listen}, POST gateway ${produce}, sse status ${s.status} ${s.err ?? ""})`); s.close();
}
