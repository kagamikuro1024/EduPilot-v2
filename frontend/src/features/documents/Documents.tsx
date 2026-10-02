"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { FileText, Trash2, Upload } from "lucide-react";
import {
  Button,
  ConfirmIrreversible,
  type Column,
  DataTable,
  DefinitionList,
  Drawer,
  EmptyState,
  InlineNotice,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  useRouteState,
  Skeleton,
  StatusText,
} from "@/shared/ui";
import { isDocOfCourse } from "@/mock/docs";
import { ANSWER_KEY_NOTE, DOCUMENTS, DOC_KIND_LABEL, FAILING_UPLOAD, titleFromFile, type DocRow, type DocStatus } from "@/mock/documents";
import { fmtShortDate } from "@/mock/core";
import { useSession } from "@/shared/session/session";
import { simNowMs } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import s from "./Documents.module.css";

type Flags = Record<string, { forAi: boolean; forStudents: boolean }>;
type Progress = { id: string; file: string; progress: number };

const STATUS_LABEL: Record<DocStatus, string> = { READY: "Sẵn sàng", PROCESSING: "Đang xử lý", FAILED: "Lỗi xử lý" };
const STATUS_TONE: Record<DocStatus, "green" | "amber" | "red"> = { READY: "green", PROCESSING: "amber", FAILED: "red" };

export function Documents() {
  const { role, course } = useSession();
  const [added, setAdded] = useDemoSlice<DocRow[]>("documents.new", []);
  const [removed, setRemoved] = useDemoSlice<string[]>("documents.removed", []);
  const [flags, setFlags] = useDemoSlice<Flags>("documents.flags", {});
  const [uploads, setUploads] = useState<Progress[]>([]);
  const [over, setOver] = useState(false);
  const [deleting, setDeleting] = useState<DocRow | null>(null);
  const [detail, setDetail] = useState<DocRow | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const undo = useUndoLine();
  // `?state=empty` minh hoạ danh sách rỗng: số đếm ở đầu màn phải khớp (03-8)
  const emptyShown = useRouteState() === "empty";

  const docs = useMemo(
    () =>
      [...added, ...DOCUMENTS]
        .filter((d) => isDocOfCourse(d, course.id) && !removed.includes(d.id))
        .map((d) => ({ ...d, forAi: flags[d.id]?.forAi ?? d.forAi, forStudents: flags[d.id]?.forStudents ?? d.forStudents })),
    [added, removed, flags, course.id],
  );

  useEffect(() => {
    if (uploads.length === 0) return;
    const t = window.setInterval(() => setUploads((prev) => prev.map((u) => ({ ...u, progress: Math.min(100, u.progress + 9) }))), 120);
    return () => window.clearInterval(t);
  }, [uploads.length]);

  /** Tải lên: dòng hiện ngay ở trạng thái `PROCESSING` (chưa dùng cho AI, chưa hiện cho SV), xong mới `READY` / `FAILED`. */
  function startUpload(files: string[]) {
    const ups = files.map((file, i) => ({ id: `up-${Date.now()}-${i}`, file, progress: 0 }));
    setUploads((prev) => [...prev, ...ups]);
    setAdded((prev) => [
      ...ups.map<DocRow>((u) => ({
        id: u.id,
        title: titleFromFile(u.file),
        file: u.file,
        kind: "lecture",
        week: null,
        uploaded: fmtShortDate(new Date(simNowMs())),
        pages: 18,
        size: "1,9 MB",
        forStudents: false,
        forAi: false,
        status: "PROCESSING",
      })),
      ...prev,
    ]);
    window.setTimeout(() => {
      setUploads((prev) => prev.filter((u) => !ups.some((x) => x.id === u.id)));
      setAdded((prev) =>
        prev.map((d) => {
          const up = ups.find((u) => u.id === d.id);
          if (!up) return d;
          const failed = up.file === FAILING_UPLOAD.name;
          return failed
            ? { ...d, status: "FAILED", pages: 12, size: "4,8 MB", failReason: FAILING_UPLOAD.failReason, failFix: FAILING_UPLOAD.failFix }
            : { ...d, status: "READY", forAi: true };
        }),
      );
    }, 1500);
  }

  function toggle(d: DocRow, key: "forAi" | "forStudents") {
    const next = !d[key];
    setFlags((prev) => ({ ...prev, [d.id]: { forAi: d.forAi, forStudents: d.forStudents, [key]: next } }));
    undo.push(
      key === "forAi"
        ? `${next ? "Đã dùng" : "Đã ngừng dùng"} “${d.title}” cho AI`
        : `${next ? "Đã hiện" : "Đã ẩn"} “${d.title}” với sinh viên`,
      () => setFlags((prev) => ({ ...prev, [d.id]: { ...prev[d.id], [key]: !next } as Flags[string] })),
    );
  }

  const failedDocs = docs.filter((d) => d.status === "FAILED");

  const columns: Column<DocRow>[] = [
    {
      key: "name",
      header: "Tài liệu",
      frozen: true,
      width: "320px",
      render: (d) => <span className={s.name}>{d.title}</span>,
    },
    { key: "kind", header: "Loại", width: "110px", render: (d) => <span className={s.meta}>{DOC_KIND_LABEL[d.kind]}</span> },
    { key: "week", header: "Tuần", width: "80px", render: (d) => <span className={s.meta}>{d.week ? `Tuần ${d.week}` : "—"}</span> },
    {
      key: "ai",
      header: "Dùng cho AI",
      width: "150px",
      render: (d) =>
        d.kind === "answer" ? (
          <span className={s.locked}>Không dùng cho AI của sinh viên</span>
        ) : d.status === "FAILED" ? (
          <StatusText tone="neutral">Không dùng được</StatusText>
        ) : d.status !== "READY" ? (
          <StatusText tone="amber">Chờ xử lý</StatusText>
        ) : (
          <button type="button" className={s.toggle} aria-pressed={d.forAi} disabled={role !== "teacher"} onClick={() => toggle(d, "forAi")}>
            <StatusText tone={d.forAi ? "green" : "neutral"}>{d.forAi ? "Có" : "Không"}</StatusText>
          </button>
        ),
    },
    {
      key: "students",
      header: "Hiện cho sinh viên",
      width: "170px",
      render: (d) =>
        d.kind === "answer" ? (
          <span className={s.locked}>Không hiển thị cho sinh viên</span>
        ) : (
          <button type="button" className={s.toggle} aria-pressed={d.forStudents} disabled={role !== "teacher" || d.status !== "READY"} onClick={() => toggle(d, "forStudents")}>
            <StatusText tone={d.forStudents ? "green" : "neutral"}>{d.forStudents ? "Có" : "Không"}</StatusText>
          </button>
        ),
    },
    {
      key: "status",
      header: "Trạng thái",
      width: "180px",
      render: (d) => (
        <span className={s.flagCell}>
          <StatusText tone={STATUS_TONE[d.status]}>{STATUS_LABEL[d.status]}</StatusText>
        </span>
      ),
    },
    { key: "updated", header: "Cập nhật", width: "110px", render: (d) => <span className={s.meta}>{d.uploaded}</span> },
    {
      key: "menu",
      header: "",
      width: "48px",
      align: "end",
      render: (d) => (
        <OverflowMenu
          label={`Hành động với ${d.title}`}
          items={[
            { label: "Xem chi tiết", onSelect: () => setDetail(d) },
            ...(role === "teacher"
              ? [{ label: "Xoá tài liệu", icon: <Trash2 aria-hidden />, danger: true, onSelect: () => setDeleting(d) }]
              : []),
          ]}
        />
      ),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Tài liệu"
        description="Nguồn để AI trả lời sinh viên và nội dung hiện trong Thư viện của lớp."
        meta={
          <>
            <span>{emptyShown ? 0 : docs.length} tài liệu</span>
            <span>{emptyShown ? 0 : docs.filter((d) => d.forAi).length} tài liệu đang dùng cho AI</span>
            <span>Đáp án không bao giờ hiện cho sinh viên</span>
          </>
        }
      />

      {role === "teacher" && (
        <div
          className={[s.drop, over ? s.dropOver : ""].join(" ")}
          onDragOver={(e) => {
            e.preventDefault();
            setOver(true);
          }}
          onDragLeave={() => setOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setOver(false);
            startUpload(Array.from(e.dataTransfer.files).map((f) => f.name));
          }}
        >
          <span className={s.dropText}>
            <span className={s.dropTitle}>Kéo tệp vào đây để tải lên</span>
            <span className={s.dropHint}>PDF, DOCX, PPTX · tối đa 50 MB mỗi tệp. Tài liệu được đọc nội dung trước khi dùng cho AI.</span>
          </span>
          <span className={s.dropActions}>
            <input
              ref={fileRef}
              className={s.file}
              type="file"
              multiple
              aria-label="Chọn tệp để tải lên"
              onChange={(e) => {
                startUpload(Array.from(e.target.files ?? []).map((f) => f.name));
                e.target.value = "";
              }}
            />
            <Button variant="ghost" onClick={() => startUpload([FAILING_UPLOAD.name])}>
              Thử tệp mẫu lỗi
            </Button>
            <Button variant="primary" icon={<Upload aria-hidden />} onClick={() => fileRef.current?.click()}>
              Tải tài liệu
            </Button>
          </span>
        </div>
      )}

      {uploads.length > 0 && (
        <div className={s.uploads}>
          {uploads.map((u) => (
            <div key={u.id} className={s.upload}>
              <div className={s.uploadHead}>
                <span className={s.uploadName}>{titleFromFile(u.file)}</span>
                <span className={s.meta}>{u.progress < 60 ? "Đang tải lên…" : "Đang đọc nội dung…"}</span>
              </div>
              <span className={s.bar}>
                <span className={s.barFill} style={{ width: `${u.progress}%` }} />
              </span>
            </div>
          ))}
        </div>
      )}

      <PageState
        loading={<Skeleton lines={10} />}
        empty={
          <EmptyState
            title="Chưa có tài liệu"
            icon={<FileText aria-hidden />}
            action={
              role === "teacher" ? (
                <Button variant="primary" icon={<Upload aria-hidden />} onClick={() => fileRef.current?.click()}>
                  Tải tài liệu
                </Button>
              ) : undefined
            }
          >
            Tải bài giảng, quy chế môn học và đề cũ lên để AI trả lời sinh viên theo đúng nội dung lớp bạn dạy. Đáp án tải lên vẫn được giữ kín với sinh viên.
          </EmptyState>
        }
        error={{ problem: "Không tải được danh sách tài liệu.", recovery: "Tệp đã tải lên vẫn được giữ. Thử lại sau ít phút." }}
      >
        {failedDocs.map((d) => (
          <InlineNotice key={d.id} tone="danger" title={`${d.title}: ${d.failReason}`}>
            {d.failFix}
          </InlineNotice>
        ))}

        <InlineNotice tone="privacy" compact>
          Tài liệu loại Đáp án: {ANSWER_KEY_NOTE}.
        </InlineNotice>

        {undo.node}

        <DataTable
          caption="Tài liệu của học phần"
          columns={columns}
          rows={docs}
          rowKey={(d) => d.id}
          empty={<EmptyState title="Chưa có tài liệu">Tải tài liệu đầu tiên lên để bắt đầu.</EmptyState>}
        />
      </PageState>

      <Drawer open={Boolean(detail)} onClose={() => setDetail(null)} title={detail?.title ?? ""} description={detail ? DOC_KIND_LABEL[detail.kind] : undefined}>
        {detail && (
          <DefinitionList
            items={[
              { term: "Tên tệp gốc", value: detail.file },
              { term: "Tuần", value: detail.week ? `Tuần ${detail.week}` : "Không gắn tuần" },
              { term: "Tải lên", value: detail.uploaded },
              { term: "Dung lượng", value: `${detail.pages} trang · ${detail.size}` },
              { term: "Trạng thái", value: <StatusText tone={STATUS_TONE[detail.status]}>{STATUS_LABEL[detail.status]}</StatusText> },
              {
                term: "Hiện cho sinh viên",
                value: detail.kind === "answer" ? "Không hiển thị cho sinh viên" : detail.forStudents ? "Có" : "Không",
              },
              {
                term: "Dùng cho AI",
                value:
                  detail.kind === "answer"
                    ? "Không dùng cho AI của sinh viên"
                    : detail.status === "FAILED"
                      ? "Không dùng được"
                      : detail.status !== "READY"
                        ? "Chờ xử lý"
                        : detail.forAi
                          ? "Có"
                          : "Không",
              },
            ]}
          />
        )}
      </Drawer>

      <ConfirmIrreversible
        open={Boolean(deleting)}
        onClose={() => setDeleting(null)}
        onConfirm={() => {
          if (!deleting) return;
          const doc = deleting;
          setRemoved((prev) => [...prev, doc.id]);
          undo.push(`Đã xoá “${doc.title}”`, () => setRemoved((prev) => prev.filter((x) => x !== doc.id)));
        }}
        title="Xoá tài liệu khỏi lớp"
        consequence={
          deleting
            ? `Xoá “${deleting.title}” (${deleting.pages} trang). AI sẽ không dùng tài liệu này để trả lời nữa${deleting.forStudents ? " và sinh viên không còn thấy trong Thư viện" : ""}.`
            : ""
        }
        confirmLabel="Xoá tài liệu"
      />
    </Page>
  );
}
