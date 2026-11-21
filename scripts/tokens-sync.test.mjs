// Chạy: node --test scripts/tokens-sync.test.mjs
// DEF-R3-02 (TC-PU01-01): kit `docs/design/DESIGN_TOKENS.css` ⊆ `frontend/src/shared/styles/tokens.css` — mọi khai báo `--ep-*: giá trị` của kit có mặt y nguyên ở code
// (code thắng khi DESIGN.md / US đổi giá trị: góp ý #25 hạ --ep-ink-3, uiux-91 US-04 đặt --ep-slow 200 ms). Kit lệch = kit cũ, sửa kit.
import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const decls = (file) => {
  const css = readFileSync(file, "utf8").replace(/\/\*[\s\S]*?\*\//g, "");
  const out = new Map(); // tên → tập giá trị (một token có thể khai báo lại ở chủ đề / @media)
  for (const m of css.matchAll(/(--ep-[a-z0-9-]+)\s*:\s*([^;{}]+);/g)) {
    const v = m[2].trim().replace(/\s+/g, " ");
    if (!out.has(m[1])) out.set(m[1], new Set());
    out.get(m[1]).add(v);
  }
  return out;
};

test("mọi khai báo của kit có mặt y nguyên ở tokens.css", () => {
  const kit = decls("docs/design/DESIGN_TOKENS.css"), code = decls("frontend/src/shared/styles/tokens.css");
  assert.ok(kit.size > 50, `kit chỉ có ${kit.size} token`);
  const bad = [];
  for (const [name, vals] of kit) for (const v of vals) if (!code.get(name)?.has(v)) bad.push(`${name}: kit «${v}» ≠ code ${[...(code.get(name) ?? ["(không có)"])].map((x) => `«${x}»`).join(" / ")}`);
  assert.deepEqual(bad, []);
});
