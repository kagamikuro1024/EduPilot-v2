"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError } from "@/shared/data";
import { Button, ButtonLink, EmptyState, InlineNotice, Page, PageHeader, PageState, Panel, Skeleton } from "@/shared/ui";
import { askAboutDoc, downloadUrl, useLibDoc } from "./libraryApi";
import { metaOf } from "./RealLibrary";
import r from "./RealLibrary.module.css";

/** Chi tiết tài liệu thật: PDF xem trước ngay trong trang (URL ký sẵn 5 phút); DOCX / PPTX chỉ có `Tải xuống`; `Hỏi AI về tài liệu` chỉ khi `can_ask_ai`. */
export function RealLibraryDetail({ courseId, id }: { courseId: string; id: string }) {
  const router = useRouter();
  const q = useLibDoc(courseId, id);
  const [err, setErr] = useState<string | null>(null);

  async function download() {
    setErr(null);
    try {
      window.location.assign(await downloadUrl(courseId, id));
    } catch (e) {
      setErr(e instanceof ApiError && e.code === "FILE_GONE" ? "Tệp này không còn nữa." : "Chưa tải được. Thử lại.");
    }
  }
  async function ask() {
    setErr(null);
    try {
      router.push(`/chat?session=${await askAboutDoc(courseId, id)}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.userMessage : "Chưa mở được. Thử lại.");
    }
  }

  return (
    <Page width="full">
      <PageHeader title={q.data?.title ?? "Tài liệu"} actions={<ButtonLink href="/library" variant="ghost">Thư viện</ButtonLink>} />
      {q.error instanceof ApiError && q.error.status === 404 ? (
        <Panel><EmptyState title="Không tìm thấy tài liệu." action={<ButtonLink href="/library" variant="primary">Về thư viện</ButtonLink>} /></Panel>
      ) : (
      <PageState query={q} loading={<Panel><Skeleton lines={5} /></Panel>} >
        {q.data && (
          <Panel>
            <div className={r.detail} data-part="library-detail">
              <p className={r.meta}>{[q.data.file_kind, metaOf(q.data), q.data.page_count ? `${q.data.page_count} trang` : null].filter(Boolean).join(" · ")}</p>
              {q.data.preview_url && <iframe className={r.preview} src={q.data.preview_url} title={`Xem trước ${q.data.title}`} data-part="pdf-preview" />}
              {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
              <div className={r.actions}>
                {q.data.can_ask_ai && <Button variant="primary" onClick={() => void ask()}>Hỏi AI về tài liệu</Button>}
                <Button variant={q.data.can_ask_ai ? "secondary" : "primary"} onClick={() => void download()}>Tải xuống</Button>
              </div>
            </div>
          </Panel>
        )}
      </PageState>
      )}
    </Page>
  );
}
