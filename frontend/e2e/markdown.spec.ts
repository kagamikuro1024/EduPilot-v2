import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { expect, test } from "@playwright/test";
import { parseInline, parseMarkdown, type Block, type Inline } from "../src/shared/lib/markdown";

// US-PE-03 AC1 — bộ render Markdown tối thiểu: HTML thô chỉ là CHỮ, không bao giờ thành thẻ. Test thuần (không cần trình duyệt).

const text = (nodes: Inline[]): string => nodes.map((n) => (n.t === "text" || n.t === "code" ? n.v : text(n.c))).join("");
const flat = (b: Block): string => (b.t === "p" ? text(b.c) : b.t === "code" ? b.v : b.t === "table" ? [b.head, ...b.rows].flat().map(text).join("|") : b.items.map(text).join("|"));

const HOSTILE = [
  "<script>alert(1)</script>",
  "<img src=x onerror=alert(1)>",
  "<a href=\"javascript:alert(1)\">bấm</a>",
  "<iframe src=//evil></iframe>",
  "[bấm](javascript:alert(1))",
  "![x](http://evil/x.png)",
  "http://evil.example/auto-link",
  "<svg onload=alert(1)>",
];

test("markdown: HTML thô, liên kết và ảnh chỉ là chữ (không có kiểu nút nào ngoài đoạn / nhấn / mã / danh sách / bảng)", () => {
  for (const h of HOSTILE) {
    const blocks = parseMarkdown(`Đề: ${h}\n\n- ${h}\n\n| a | b |\n| --- | --- |\n| ${h} | x |\n\n\`\`\`\n${h}\n\`\`\``);
    expect(blocks.map((b) => b.t).sort()).toEqual(["code", "p", "table", "ul"]);
    const all = blocks.map(flat).join("\n");
    expect(all, h).toContain(h.replace(/`/g, "")); // nguyên văn, còn nguyên dấu `<`
  }
});

test("markdown: cú pháp tối thiểu", () => {
  expect(parseInline("a **đậm** và *nghiêng* và `mã`")).toEqual([
    { t: "text", v: "a " }, { t: "strong", c: [{ t: "text", v: "đậm" }] }, { t: "text", v: " và " }, { t: "em", c: [{ t: "text", v: "nghiêng" }] }, { t: "text", v: " và " }, { t: "code", v: "mã" },
  ]);
  const b = parseMarkdown("Đoạn 1\ndòng 2\n\n1. một\n2. hai\n\n- x\n- y\n\n| Tên | Điểm |\n| --- | --- |\n| An | 9 |");
  expect(b.map((x) => x.t)).toEqual(["p", "ol", "ul", "table"]);
  expect(b[1]).toMatchObject({ t: "ol", items: [[{ t: "text", v: "một" }], [{ t: "text", v: "hai" }]] });
  expect(b[3]).toMatchObject({ t: "table" });
  // khối mã giữ nguyên chữ, kể cả dấu * _ | bên trong
  expect(parseMarkdown("```\n*a* _b_ | c\n```")).toEqual([{ t: "code", v: "*a* _b_ | c" }]);
  // dấu * lẻ không thành nhấn
  expect(parseInline("2 * 3 = 6")).toEqual([{ t: "text", v: "2 * 3 = 6" }]);
});

test("markdown: bộ vẽ không dùng dangerouslySetInnerHTML và không dựng thẻ từ chuỗi", () => {
  const src = readFileSync(resolve(__dirname, "../src/shared/domain/Markdown.tsx"), "utf8");
  expect(src).not.toMatch(/dangerouslySetInnerHTML|innerHTML|insertAdjacentHTML|createContextualFragment/);
  const feature = readFileSync(resolve(__dirname, "../src/features/questions/QuestionReview.tsx"), "utf8");
  expect(feature).not.toMatch(/dangerouslySetInnerHTML/);
  expect(feature).toContain("<Markdown source={d.stem}");
});
