"use client";

import { Search } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useCursorList } from "@/shared/data/useCursorList";
import { Button, type Column, DataTable, EmptyState, FilterChips, Input, Page, PageHeader, Panel, PrivateMark, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import { FILTERS, isFilter, signedPoints, type StudentFilter, type StudentRow } from "./studentsApi";
import s from "./StudentsView.module.css";

const SEARCH_DELAY_MS = 300;
const TABLE_HEIGHT = 560;

/**
 * `/students` của lớp thật (US-P5-04, DESIGN §14.6): tìm + chip lọc, bảng ảo hoá phân trang con trỏ, bấm dòng mở hồ sơ.
 * Ô tìm và chip nằm trong URL (`?q=&filter=`) nên quay lại từ hồ sơ vẫn còn; `?filter=` từ "Hôm nay" bật chip tương ứng.
 */
export function RealStudents({ courseId, classCode }: { courseId: string; classCode: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const urlQ = params.get("q") ?? "";
  const chips = useMemo(() => (params.get("filter") ?? "").split(",").filter(isFilter), [params]);
  const [text, setText] = useState(urlQ);

  function setUrl(next: { q?: string; chips?: StudentFilter[] }) {
    const p = new URLSearchParams(params.toString());
    const q = next.q ?? urlQ;
    const c = next.chips ?? chips;
    if (q) p.set("q", q);
    else p.delete("q");
    if (c.length) p.set("filter", c.join(","));
    else p.delete("filter");
    router.replace(`${pathname}?${p.toString()}`);
  }

  useEffect(() => {
    if (text.trim() === urlQ) return;
    const t = setTimeout(() => setUrl({ q: text.trim() }), SEARCH_DELAY_MS);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [text]);

  const filter = chips.join(",");
  const list = useCursorList<StudentRow>(["students", courseId, urlQ, filter], `/courses/${courseId}/students`, {
    limit: 100,
    query: { q: urlQ || undefined, filter: filter || undefined },
    idOf: (r) => r.student.id,
  });
  const first = list.data?.pages[0];
  const total = first && "total" in first && typeof first.total === "number" ? first.total : null;
  const filtered = urlQ !== "" || chips.length > 0;

  const columns: Column<StudentRow>[] = [
    {
      key: "name",
      header: "Sinh viên",
      frozen: true,
      width: "26%",
      render: (r) => (
        <span className={s.name}>
          <span className="ep-item-title">{r.student.full_name}</span>
          <span className="ep-meta">{r.student.student_code ?? ""}</span>
        </span>
      ),
    },
    { key: "att", header: "Chuyên cần", render: (r) => (r.attendance.recorded === 0 ? "—" : `Vắng ${r.attendance.absent}/${r.attendance.recorded}`) },
    { key: "speak", header: "Phát biểu", align: "end", render: (r) => `${r.participation.speak_count} · ${signedPoints(r.participation.points_total)}` },
    {
      key: "act",
      header: <span title="Phút: 7 ngày gần nhất. Câu hỏi và bài đăng: tuần trước (T2–CN).">Hoạt động học</span>,
      align: "end",
      render: (r) => `${r.study.tracked ? `${r.study.minutes_7d} phút` : "—"} · ${r.activity.asks_last_week} câu hỏi${r.activity.posts_last_week > 0 ? ` · ${r.activity.posts_last_week} bài` : ""}`,
    },
    {
      key: "risk",
      header: "Rủi ro",
      render: (r) => (r.risk.level === "AT_RISK" ? <StatusText tone="red">Cần chú ý</StatusText> : r.risk.level === "WATCH" ? <StatusText tone="amber">Theo dõi</StatusText> : null),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader title="Sinh viên" description={total === null ? classCode : `${total} sinh viên · ${classCode}`} meta={<PrivateMark>Chuyên cần và nhãn rủi ro chỉ giảng viên/TA thấy</PrivateMark>} />
      <Panel>
        <Toolbar end={<FilterChips label="Lọc nhanh" value={chips} onChange={(next) => setUrl({ chips: next })} options={FILTERS} />}>
          <label className={s.search}>
            <Search aria-hidden />
            <span className="ep-sr-only">Tìm theo tên hoặc mã số sinh viên</span>
            <Input type="search" value={text} placeholder="Tìm theo tên hoặc MSSV" onChange={(e) => setText(e.target.value)} />
          </label>
        </Toolbar>
        {list.isError && list.items.length === 0 ? (
          <ApiErrorNotice error={list.error} showTechnical onRetry={() => void list.refetch()} title="Không tải được danh sách sinh viên." />
        ) : list.isPending ? (
          <Skeleton lines={10} />
        ) : (
          <DataTable
            virtual={{ height: TABLE_HEIGHT }}
            caption={`Sinh viên lớp ${classCode}`}
            columns={columns}
            rows={list.items}
            rowKey={(r) => r.student.id}
            onRowClick={(r) => router.push(`/students/${r.student.id}`)}
            pagination={{ nextCursor: list.hasNextPage ? (list.data?.pages.at(-1)?.next_cursor ?? null) : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage, error: list.isFetchNextPageError ? "Chưa tải được trang kế." : undefined }}
            empty={
              filtered ? (
                <EmptyState title="Không có sinh viên nào khớp." action={<Button onClick={() => { setText(""); setUrl({ q: "", chips: [] }); }}>Bỏ lọc</Button>} />
              ) : (
                <EmptyState title="Lớp này chưa có sinh viên nào." />
              )
            }
          />
        )}
      </Panel>
    </Page>
  );
}
