"use client";

import { Skeleton } from "@/shared/ui";
import type { Citation } from "./gradeApi";
import { useDocPages, useDocPreview } from "./gradeApi";
import s from "./GradeScheme.module.css";

const squash = (t: string) => t.replace(/\s+/g, " ").trim();

/** Trang tài liệu nguồn với đoạn trích được nhấn mạnh. Không tìm thấy nguyên văn thì hiện đoạn trích riêng (trích dẫn do máy chủ đã kiểm theo trang). */
export function CitationPane({ course, docId, title, cite }: { course: string; docId?: string; title?: string; cite: Citation | null }) {
  const pages = useDocPages(course, docId);
  const preview = useDocPreview(course, docId).data;
  const text = cite ? pages.data?.[cite.page] : undefined;
  const flat = text ? squash(text) : "";
  const at = cite && flat ? flat.indexOf(squash(cite.quote)) : -1;
  return (
    <div className={s.pane} data-part="citation-pane">
      <p className={s.paneHead}>{title ?? "Quy chế"}</p>
      {cite && (
        <>
          <p className={s.paneMeta}>
            <span data-part="citation-page">{`Trang ${cite.page}`}</span>
            {preview && (
              <>
                {" · "}
                <a className={s.paneLink} href={`${preview}#page=${cite.page}`} target="_blank" rel="noopener noreferrer">{`Mở trang ${cite.page}`}</a>
              </>
            )}
          </p>
          {pages.isPending && docId ? (
            <Skeleton lines={6} />
          ) : at >= 0 ? (
            <p className={s.paneText}>
              {flat.slice(0, at)}
              <mark className={s.mark}>{flat.slice(at, at + squash(cite.quote).length)}</mark>
              {flat.slice(at + squash(cite.quote).length)}
            </p>
          ) : (
            <p className={s.paneText}>
              <mark className={s.mark}>{cite.quote}</mark>
            </p>
          )}
        </>
      )}
    </div>
  );
}
