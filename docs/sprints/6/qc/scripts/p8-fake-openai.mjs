#!/usr/bin/env node
// Máy chủ OpenAI-compatible GIẢ của QC cho sprint 6 (P8). Khác provider `fake` trong tiến trình:
// máy này GHI LẠI toàn bộ payload gửi tới provider ra JSONL để quét canary / MSSV / họ tên roster.
//
// Dùng:
//   node docs/sprints/6/qc/scripts/p8-fake-openai.mjs [port=8099] [out=/tmp/p8-llm.jsonl]
//   DIMS=512 node … p8-fake-openai.mjs      # trả sai chiều để kiểm EMBED_FAILED (US-P8-01 AC7/AC10)
//   node … p8-fake-openai.mjs --selftest    # tự kiểm, không mở cổng
// Rồi ADMIN thêm provider type=openai_compatible, base_url=http://host.docker.internal:8099/v1
// và định tuyến task EMBEDDING + CHAT sang nó (FEAT-llm-gateway SRS 4.2).
//
// Quét payload: grep -c 'CANARY-7Q2X' /tmp/p8-llm.jsonl
//               jq -r '.body.input[]?, (.body.messages[]?.content)' /tmp/p8-llm.jsonl | grep -cE '20[0-9]{6}|@edupilot'

import { createServer } from "node:http";
import { appendFileSync } from "node:fs";

const DIMS = Number(process.env.DIMS || 1536);

// Vectơ xác định theo chuỗi vào: PRNG gieo bằng FNV-1a rồi chuẩn hoá L2.
function embed(text) {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < text.length; i++) { h ^= text.charCodeAt(i); h = Math.imul(h, 16777619) >>> 0; }
  const v = new Array(DIMS);
  let norm = 0;
  for (let i = 0; i < DIMS; i++) {
    h ^= h << 13; h >>>= 0; h ^= h >>> 17; h ^= h << 5; h >>>= 0;
    const x = h / 4294967296 - 0.5;
    v[i] = x; norm += x * x;
  }
  norm = Math.sqrt(norm) || 1;
  return v.map((x) => x / norm);
}

function selftest() {
  const a = embed("xin chào"), b = embed("xin chào"), c = embed("khác");
  console.assert(a.length === DIMS, "sai số chiều");
  console.assert(a.every((x, i) => x === b[i]), "không xác định");
  console.assert(!a.every((x, i) => x === c[i]), "không phân biệt chuỗi");
  const n = Math.sqrt(a.reduce((s, x) => s + x * x, 0));
  console.assert(Math.abs(n - 1) < 1e-9, "chưa chuẩn hoá L2");
  console.log(`selftest ok (dims=${DIMS})`);
}

if (process.argv.includes("--selftest")) { selftest(); process.exit(0); }

const port = Number(process.argv[2] || 8099);
const out = process.argv[3] || "/tmp/p8-llm.jsonl";

createServer((req, res) => {
  let raw = "";
  req.on("data", (c) => (raw += c));
  req.on("end", () => {
    let body = null;
    try { body = JSON.parse(raw); } catch { body = raw; }
    appendFileSync(out, JSON.stringify({ ts: new Date().toISOString(), path: req.url, body }) + "\n");

    const json = (o) => { res.writeHead(200, { "content-type": "application/json" }); res.end(JSON.stringify(o)); };

    if (req.url.startsWith("/v1/embeddings")) {
      const inputs = Array.isArray(body?.input) ? body.input : [String(body?.input ?? "")];
      return json({
        object: "list",
        model: body?.model ?? "fake-embed",
        data: inputs.map((t, i) => ({ object: "embedding", index: i, embedding: embed(String(t)) })),
        usage: { prompt_tokens: inputs.length, total_tokens: inputs.length },
      });
    }

    if (req.url.startsWith("/v1/chat/completions")) {
      const text = "Đây là câu trả lời giả của QC.";
      if (body?.stream) {
        res.writeHead(200, { "content-type": "text/event-stream", "cache-control": "no-cache" });
        for (const w of text.split(" ")) {
          res.write(`data: ${JSON.stringify({ choices: [{ index: 0, delta: { content: w + " " } }] })}\n\n`);
        }
        res.write(`data: ${JSON.stringify({ choices: [{ index: 0, delta: {}, finish_reason: "stop" }] })}\n\n`);
        return res.end("data: [DONE]\n\n");
      }
      return json({
        id: "chatcmpl-qc", object: "chat.completion", model: body?.model ?? "fake-chat",
        choices: [{ index: 0, message: { role: "assistant", content: text }, finish_reason: "stop" }],
        usage: { prompt_tokens: 1, completion_tokens: 1, total_tokens: 2 },
      });
    }

    if (req.url.startsWith("/v1/models")) return json({ object: "list", data: [{ id: "fake-chat" }, { id: "fake-embed" }] });

    res.writeHead(404, { "content-type": "application/json" });
    res.end('{"error":{"message":"not found"}}');
  });
}).listen(port, () => console.log(`p8-fake-openai :${port} dims=${DIMS} → ${out}`));
