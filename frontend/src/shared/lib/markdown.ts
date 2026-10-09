/**
 * Bộ phân tích Markdown TỐI THIỂU (SRS FEAT-weekly-exam 4.1.2, Q21): đoạn, nhấn (`**đậm**`, `*nghiêng*`), mã (`inline` và khối ```), danh sách, bảng.
 * Chỉ sinh cây dữ liệu — bộ vẽ (`shared/domain/Markdown.tsx`) dựng React node. HTML thô KHÔNG có cú pháp riêng: `<script>`, `<img onerror=…>` chỉ là chữ.
 * Không ảnh, không liên kết (kể cả tự nhận URL). An toàn nằm ở việc không bao giờ đưa chuỗi vào `dangerouslySetInnerHTML`, không ở việc cắt chuỗi lúc lưu.
 */
export type Inline = { t: "text"; v: string } | { t: "strong"; c: Inline[] } | { t: "em"; c: Inline[] } | { t: "code"; v: string };
export type Block =
  | { t: "p"; c: Inline[] }
  | { t: "ul" | "ol"; items: Inline[][] }
  | { t: "code"; v: string }
  | { t: "table"; head: Inline[][]; rows: Inline[][][] };

export function parseInline(s: string): Inline[] {
  const out: Inline[] = [];
  let buf = "";
  const flush = () => {
    if (buf) out.push({ t: "text", v: buf });
    buf = "";
  };
  let i = 0;
  while (i < s.length) {
    const ch = s[i];
    if (ch === "`") {
      const end = s.indexOf("`", i + 1);
      if (end > i + 1) {
        flush();
        out.push({ t: "code", v: s.slice(i + 1, end) });
        i = end + 1;
        continue;
      }
    }
    if (ch === "*" && s[i + 1] === "*") {
      const end = s.indexOf("**", i + 2);
      if (end > i + 2) {
        flush();
        out.push({ t: "strong", c: parseInline(s.slice(i + 2, end)) });
        i = end + 2;
        continue;
      }
    }
    if (ch === "*" || ch === "_") {
      const end = s.indexOf(ch, i + 1);
      if (end > i + 1 && s[i + 1] !== " " && s[end - 1] !== " ") {
        flush();
        out.push({ t: "em", c: parseInline(s.slice(i + 1, end)) });
        i = end + 1;
        continue;
      }
    }
    buf += ch;
    i++;
  }
  flush();
  return out;
}

const LIST_UL = /^\s*[-*]\s+(.*)$/;
const LIST_OL = /^\s*\d+[.)]\s+(.*)$/;
const cells = (line: string) =>
  line
    .trim()
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((c) => c.trim());
const isSep = (line: string) => /^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$/.test(line);

export function parseMarkdown(src: string): Block[] {
  const lines = src.replace(/\r\n?/g, "\n").split("\n");
  const blocks: Block[] = [];
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line.trim() === "") {
      i++;
      continue;
    }
    if (line.trimStart().startsWith("```")) {
      const body: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trimStart().startsWith("```")) body.push(lines[i++]);
      i++; // dòng đóng (có hoặc không)
      blocks.push({ t: "code", v: body.join("\n") });
      continue;
    }
    if (line.includes("|") && i + 1 < lines.length && isSep(lines[i + 1])) {
      const head = cells(line).map(parseInline);
      i += 2;
      const rows: Inline[][][] = [];
      while (i < lines.length && lines[i].includes("|") && lines[i].trim() !== "") rows.push(cells(lines[i++]).map(parseInline));
      blocks.push({ t: "table", head, rows });
      continue;
    }
    const ul = LIST_UL.exec(line);
    const ol = LIST_OL.exec(line);
    if (ul || ol) {
      const re = ul ? LIST_UL : LIST_OL;
      const items: Inline[][] = [];
      while (i < lines.length) {
        const m = re.exec(lines[i]);
        if (!m) break;
        items.push(parseInline(m[1]));
        i++;
      }
      blocks.push({ t: ul ? "ul" : "ol", items });
      continue;
    }
    const para: string[] = [];
    while (i < lines.length && lines[i].trim() !== "" && !lines[i].trimStart().startsWith("```") && !LIST_UL.test(lines[i]) && !LIST_OL.test(lines[i])) para.push(lines[i++]);
    blocks.push({ t: "p", c: parseInline(para.join("\n")) });
  }
  return blocks;
}
