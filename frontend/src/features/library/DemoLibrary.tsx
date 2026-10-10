"use client";

import { FileText } from "lucide-react";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { agoLabel } from "@/mock/derive";
import { DOC_KIND_LABEL, docById, libraryDocs, uploadedAt, type DocKind } from "@/mock/library";
import { useSession } from "@/shared/session/session";
import { useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { ActionList, ActionRow, Button, ButtonLink, DefinitionList, Drawer, EmptyState, Input, OverflowMenu, Page, PageHeader, PageState, Panel, SegmentedControl, Skeleton, Toolbar } from "@/shared/ui";
import s from "./Library.module.css";

type Filter = "all" | DocKind;

/** Thư viện của sinh viên: tìm tài liệu và dùng được ngay (DESIGN §14.15). */
export function DemoLibraryScreen() {
  const router = useRouter();
  const { course } = useSession();
  const now = useSimNow();
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");
  const [query, setQuery] = useState("");
  const [kind, setKind] = useState<Filter>("all");
  const [open, setOpen] = useState<string | null>(null);

  const docs = useMemo(() => libraryDocs(course.id), [course.id]);
  const q = query.trim().toLowerCase();
  const rows = docs.filter((d) => (kind === "all" ? true : d.kind === kind)).filter(
    (d) => !q || d.title.toLowerCase().includes(q) || d.topic.toLowerCase().includes(q) || d.summary.toLowerCase().includes(q),
  );
  const doc = open ? docById(open) : undefined;

  function askAi(title: string) {
    setChatDraft(`Về tài liệu “${title}”: `);
    router.push("/chat");
  }

  return (
    <Page>
      <PageHeader title="Thư viện" description="Bài giảng, quy chế và đề cũ của lớp. Tìm theo tên, chủ đề hoặc tuần học." meta={`${docs.length} tài liệu`} />

      <Toolbar
        end={
          <SegmentedControl
            label="Lọc theo loại tài liệu"
            value={kind}
            onChange={setKind}
            options={[
              { value: "all", label: "Tất cả", count: docs.length },
              { value: "lecture", label: "Bài giảng", count: docs.filter((d) => d.kind === "lecture").length },
              { value: "regulation", label: "Quy chế", count: docs.filter((d) => d.kind === "regulation").length },
              { value: "exam", label: "Đề cũ", count: docs.filter((d) => d.kind === "exam").length },
            ]}
          />
        }
      >
        <Input
          className={s.search}
          type="search"
          value={query}
          aria-label="Tìm tài liệu"
          placeholder="Tìm tài liệu: AES, chữ ký số, quy chế…"
          onChange={(e) => setQuery(e.target.value)}
        />
      </Toolbar>

      <PageState
        loading={<Panel><Skeleton lines={7} /></Panel>}
        empty={
          <Panel><EmptyState title="Chưa có tài liệu nào được chia sẻ" action={<ButtonLink href="/" variant="primary">Về Hôm nay</ButtonLink>}>
            Khi giảng viên đăng bài giảng hoặc quy chế, tài liệu sẽ hiện ở đây.
          </EmptyState></Panel>
        }
      >
        {rows.length === 0 ? (
          <Panel><EmptyState title={`Không có tài liệu nào khớp “${query}”`} action={<Button variant="primary" onClick={() => { setQuery(""); setKind("all"); }}>Xoá bộ lọc</Button>}>
            Thử từ khoá ngắn hơn, ví dụ “AES” hoặc “quy chế”.
          </EmptyState></Panel>
        ) : (
          <Panel>
          <ActionList label="Tài liệu của lớp">
            {rows.map((d) => (
              <ActionRow
                key={d.id}
                lead={<FileText className={s.file} aria-hidden />}
                title={d.title}
                context={d.summary}
                meta={`${DOC_KIND_LABEL[d.kind]}${d.week ? ` · tuần ${d.week}` : ""} · ${d.pages} trang · ${d.size} · cập nhật ${agoLabel(uploadedAt(d), now)}`}
                onSelect={() => setOpen(d.id)}
                action={
                  <span className={s.rowActions}>
                    {d.practiceAttemptId && <ButtonLink href={`/practice/${d.practiceAttemptId}`} size="sm">Luyện đề này</ButtonLink>}
                    <OverflowMenu
                      items={[
                        { label: "Xem tài liệu", onSelect: () => setOpen(d.id) },
                        { label: "Hỏi AI về tài liệu", onSelect: () => askAi(d.title) },
                      ]}
                    />
                  </span>
                }
              />
            ))}
          </ActionList>
          </Panel>
        )}
      </PageState>

      <Drawer
        open={Boolean(doc)}
        onClose={() => setOpen(null)}
        title={doc?.title ?? ""}
        description={doc?.summary}
        footer={
          doc ? (
            <>
              {doc.practiceAttemptId && <ButtonLink href={`/practice/${doc.practiceAttemptId}`}>Luyện đề này</ButtonLink>}
              <Button variant="primary" onClick={() => askAi(doc.title)}>
                Hỏi AI về tài liệu
              </Button>
            </>
          ) : undefined
        }
      >
        {doc && (
          <>
            <DefinitionList
              items={[
                { term: "Loại", value: DOC_KIND_LABEL[doc.kind] },
                { term: "Chủ đề", value: doc.topic },
                { term: "Tuần", value: doc.week ? `Tuần ${doc.week}` : "Không gắn tuần" },
                { term: "Tệp", value: `${doc.file} · ${doc.pages} trang · ${doc.size}` },
                { term: "Cập nhật", value: `${doc.uploaded} · ${agoLabel(uploadedAt(doc), now)}` },
              ]}
            />
            <div className={s.preview} aria-label="Trang xem trước">
              <p className={s.previewTitle}>{doc.title}</p>
              <p className={s.previewLine} />
              <p className={s.previewLine} />
              <p className={s.previewLineShort} />
              <p className={s.previewNote}>Bản mô phỏng hiện trang mẫu thay cho nội dung PDF.</p>
            </div>
          </>
        )}
      </Drawer>
    </Page>
  );
}
