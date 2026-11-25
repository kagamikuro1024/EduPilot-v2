import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { expect, test } from "@playwright/test";
import { earlyText } from "../src/shared/data/earlyText";
import { parseInline, parseMarkdown, stripComments, type Block, type Inline } from "../src/shared/lib/markdown";

// US-PE-03 AC1 — bộ render Markdown tối thiểu: HTML thô chỉ là CHỮ, không bao giờ thành thẻ. Test thuần (không cần trình duyệt).

const text = (nodes: Inline[]): string => nodes.map((n) => (n.t === "text" || n.t === "code" ? n.v : text(n.c))).join("");
const flat = (b: Block): string => (b.t === "p" || b.t === "h" ? text(b.c) : b.t === "code" ? b.v : b.t === "table" ? [b.head, ...b.rows].flat().map(text).join("|") : b.items.map(text).join("|"));

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

// BUG-S9-02 (proposals #17): chú thích HTML `<!-- … -->` trong tệp đề bài là ghi chú nội bộ, không bao giờ in ra thành chữ.
test("markdown: chú thích HTML không hiện thành chữ (một dòng, nhiều dòng, giữa dòng, chưa đóng), khối mã giữ nguyên", () => {
  const all = (src: string) => parseMarkdown(src).map(flat).join("\n");
  const hidden = "NỘI DUNG GỐC của Bài tập 03. BA sở hữu. Dữ liệu mô phỏng (D44).";
  const doc = `<!-- ${hidden} -->\n\n# Bài tập 03\n\nĐoạn <!-- ghi chú giữa dòng --> tiếp.\n\n<!-- chú thích\nnhiều dòng\n  BA sở hữu -->\n\n- ý một\n\n<!-- chưa đóng\nvẫn ẩn`;
  const text = all(doc);
  for (const w of ["NỘI DUNG GỐC", "BA sở hữu", "D44", "ghi chú giữa dòng", "chú thích", "chưa đóng", "vẫn ẩn", "<!--", "-->"]) expect(text, w).not.toContain(w);
  expect(text).toContain("Bài tập 03");
  expect(text).toContain("Đoạn  tiếp.");
  expect(text).toContain("ý một");
  expect(parseMarkdown(`<!-- ${hidden} -->`)).toEqual([]);
  expect(all("```\n<!-- giữ nguyên -->\n```")).toContain("<!-- giữ nguyên -->");
  expect(stripComments("a\r\nb")).toBe("a\r\nb"); // không đổi khi không có chú thích
});

// LCP-3 (US-P7-03; SRS FEAT-assignments 4.2.1 «description Markdown»): tiêu đề `#`…`######` là khối riêng (không hiện nguyên dấu #); chưa có dấu cách hoặc quá 6 dấu thì vẫn là đoạn.
test("markdown: tiêu đề # … ###### thành khối h; «#x», «####### x» vẫn là đoạn; HTML trong tiêu đề chỉ là chữ", () => {
  const b = parseMarkdown("# Bài tập 03\n\n## Đề bài ##\nĐoạn một\n### Mục **nhỏ**\n#x\n####### bảy\n# <script>alert(1)</script>");
  expect(b.map((x) => (x.t === "h" ? `h${x.level}` : x.t))).toEqual(["h1", "h2", "p", "h3", "p", "h1"]);
  expect(flat(b[1])).toBe("Đề bài");
  expect(flat(b[2])).toBe("Đoạn một");
  expect(flat(b[3])).toBe("Mục nhỏ");
  expect(flat(b[4])).toBe("#x\n####### bảy");
  expect(flat(b[5])).toBe("<script>alert(1)</script>");
});

// Ô sớm của earlyPaint: chữ thô đọc được, KHÔNG còn dấu Markdown (#, **, `, hàng gạch bảng, rào mã, chú thích HTML).
test("earlyText: bỏ dấu tiêu đề, nhấn, rào mã, chú thích; mục danh sách thành «• »", () => {
  const md = "<!-- nội bộ -->\n\n# Bài tập 03 — Phân tích\n\n## Đề bài\nĐọc **kỹ** mã `nc -l`.\n\n- Nêu ba dấu hiệu\n* Đề xuất\n\n| a | b |\n| --- | --- |\n| 1 | 2 |\n\n```\nls\n```\n\n\n\nHết";
  expect(earlyText(md)).toBe("Bài tập 03 — Phân tích\n\nĐề bài\nĐọc kỹ mã nc -l.\n\n• Nêu ba dấu hiệu\n• Đề xuất\n\n| a | b |\n| 1 | 2 |\n\nls\n\nHết");
  expect(earlyText("x\r\n\r\ny")).toBe("x\n\ny");
});
