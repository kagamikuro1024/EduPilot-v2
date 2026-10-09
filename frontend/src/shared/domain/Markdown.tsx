import type { ReactNode } from "react";
import { parseMarkdown, type Inline } from "@/shared/lib/markdown";
import s from "./Markdown.module.css";

const inline = (nodes: Inline[]): ReactNode[] =>
  nodes.map((n, i) => {
    switch (n.t) {
      case "strong":
        return <strong key={i}>{inline(n.c)}</strong>;
      case "em":
        return <em key={i}>{inline(n.c)}</em>;
      case "code":
        return <code key={i}>{n.v}</code>;
      default:
        return n.v; // chuỗi → React thoát ký tự: `<script>` hiện thành chữ
    }
  });

/** Đề / giải thích Markdown tối thiểu, dựng thẳng React node (không chèn HTML thô). */
export function Markdown({ source, className }: { source: string; className?: string }) {
  return (
    <div className={[s.md, className ?? ""].join(" ")} data-part="markdown">
      {parseMarkdown(source).map((b, i) => {
        switch (b.t) {
          case "ul":
            return <ul key={i}>{b.items.map((it, k) => <li key={k}>{inline(it)}</li>)}</ul>;
          case "ol":
            return <ol key={i}>{b.items.map((it, k) => <li key={k}>{inline(it)}</li>)}</ol>;
          case "code":
            return <pre key={i} className={s.pre}><code>{b.v}</code></pre>;
          case "table":
            // div + vai ARIA (không dùng thẻ bảng thô — bảng dữ liệu của ứng dụng đi qua DataTable); lưới theo số cột của dòng tiêu đề
            return (
              <div key={i} className={s.tableWrap}>
                <div role="table" className={s.table} style={{ ["--cols" as string]: b.head.length }}>
                  <div role="row" className={s.tr}>{b.head.map((h, k) => <div role="columnheader" className={s.th} key={k}>{inline(h)}</div>)}</div>
                  {b.rows.map((r, k) => <div role="row" className={s.tr} key={k}>{r.map((c, j) => <div role="cell" className={s.td} key={j}>{inline(c)}</div>)}</div>)}
                </div>
              </div>
            );
          default:
            return <p key={i}>{inline(b.c)}</p>;
        }
      })}
    </div>
  );
}
