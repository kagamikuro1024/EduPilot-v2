// QC US-P4-02: nối GET /api/v1/events, ghi mỗi khung: <epoch_ms>\t<id>\t<event>\t<data>. Dùng: node sse.mjs <token> <out> [<Last-Event-ID>] [<GW>]
import { appendFileSync, writeFileSync } from "node:fs";
import https from "node:https";
const [token, out, lastId, gw = "https://localhost:543"] = process.argv.slice(2);
writeFileSync(out, "");
const u = new URL(gw + "/api/v1/events");
const headers = { Authorization: `Bearer ${token}`, Accept: "text/event-stream" };
if (lastId) headers["Last-Event-ID"] = lastId;
const req = https.request({ host: u.hostname, port: u.port, path: u.pathname, headers, rejectUnauthorized: false }, (res) => {
  appendFileSync(out, `${Date.now()}\t-\tHTTP\t${res.statusCode}\n`);
  let buf = "";
  res.on("data", (c) => {
    buf += c.toString();
    let i;
    while ((i = buf.indexOf("\n\n")) >= 0) {
      const frame = buf.slice(0, i); buf = buf.slice(i + 2);
      const f = Object.fromEntries(frame.split("\n").filter((l) => l.includes(":")).map((l) => [l.slice(0, l.indexOf(":")), l.slice(l.indexOf(":") + 1).trim()]));
      if (f.event || f.data) appendFileSync(out, `${Date.now()}\t${f.id ?? "-"}\t${f.event ?? "message"}\t${f.data ?? ""}\n`);
    }
  });
  res.on("end", () => appendFileSync(out, `${Date.now()}\t-\tEND\t\n`));
});
req.on("error", (e) => appendFileSync(out, `${Date.now()}\t-\tERR\t${e.message}\n`));
req.end();
