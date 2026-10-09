"use client";

import { useState } from "react";
import { ExternalLink } from "lucide-react";
import { NOW, fmtScore, fmtTime } from "@/mock/core";
import { COURSE_SUMMARY, REQUESTS, STATUS_LABEL, STATUS_STRIP, type LlmRequest } from "@/mock/system";
import { useSession } from "@/shared/session/session";
import { Button, type Column, DataTable, DefinitionList, Drawer, EmptyState, Field, InlineNotice, Page, PageHeader, PageState, Panel, Section, SegmentedControl, Skeleton, StatusText, Textarea } from "@/shared/ui";
import s from "./observability.module.css";

/** Dải trạng thái gọn: nhãn nhỏ, số lớn, ngăn bằng đường kẻ dọc — không phải thẻ số liệu. */
export function StatusStrip({ items }: { items: Array<{ label: string; value: string; hint?: string }> }) {
  return (
    <dl className={s.strip}>
      {items.map((m) => (
        <div key={m.label} className={s.stripItem}>
          <dt className="ep-meta">{m.label}</dt>
          <dd>
            <span className="ep-data">{m.value}</span>
            {m.hint && <span className={s.stripHint}>{m.hint}</span>}
          </dd>
        </div>
      ))}
    </dl>
  );
}

type Filter = "all" | "problem" | "privacy";

export function ObservabilityScreen() {
  const { role, course, user } = useSession();
  const isAdmin = role === "admin";
  const [filter, setFilter] = useState<Filter>("all");
  const [open, setOpen] = useState<LlmRequest | null>(null);
  const [reason, setReason] = useState("");
  const [reasonTouched, setReasonTouched] = useState(false);
  // báo lỗi TẠI Ô (không chỉ làm mờ nút): trống sau khi rời ô, hoặc đã gõ mà còn ngắn (04-1)
  const reasonError =
    reason.trim().length >= 10 ? undefined : reason.trim().length > 0 ? "Viết rõ hơn một chút (ít nhất 10 ký tự) để người đọc nhật ký sau này hiểu được." : reasonTouched || reason.length > 0 ? "Cần nhập lý do trước khi mở nội dung." : undefined;
  const [unlocked, setUnlocked] = useState(false);
  const [trace, setTrace] = useState(false);

  const base = isAdmin ? REQUESTS : REQUESTS.filter((r) => r.courseId === course.id);
  const rows = base.filter((r) => (filter === "problem" ? r.status !== "ok" : filter === "privacy" ? r.privacy !== null : true));

  const columns: Column<LlmRequest>[] = [
    { key: "at", header: "Lúc", width: "76px", render: (r) => <span className="ep-num">{fmtTime(r.at)}</span> },
    {
      key: "task",
      header: "Tác vụ",
      frozen: true,
      render: (r) => (
        <>
          <span className={s.task}>{r.task}</span>
          <span className={s.sub}>Lớp {r.courseId}</span>
        </>
      ),
    },
    { key: "model", header: "Model", render: (r) => <span className={s.mono}>{r.model}</span> },
    { key: "latency", header: "Độ trễ", align: "end", render: (r) => <span className="ep-num">{r.latencyMs >= 1000 ? `${fmtScore(r.latencyMs / 1000)} s` : `${r.latencyMs} ms`}</span> },
    {
      key: "confidence",
      header: "Độ tin cậy",
      align: "end",
     
      render: (r) => (r.task === "Đánh chỉ mục tài liệu" ? <span className={s.sub}>—</span> : <span className="ep-num">{fmtScore(r.confidence, 2)}</span>),
    },
    {
      key: "privacy",
      header: "Bảo vệ thông tin",
     
      render: (r) => (r.privacy ? <StatusText tone="blue">{r.privacy}</StatusText> : <span className={s.sub}>Không có</span>),
    },
    {
      key: "status",
      header: "Kết quả",
      render: (r) => <StatusText tone={r.status === "error" ? "red" : r.status === "fallback" ? "amber" : "green"}>{STATUS_LABEL[r.status]}</StatusText>,
    },
  ];

  function closeDrawer() {
    setOpen(null);
    setUnlocked(false);
    setReason("");
    setReasonTouched(false);
    setTrace(false);
  }

  return (
    <Page width="wide">
      <PageHeader
        title="Quan sát AI"
        description={
          isAdmin
            ? "Hệ thống đang chạy thế nào, và khi có yêu cầu hỏng thì hỏng ở đâu."
            : "Số liệu AI của lớp bạn. Nội dung câu hỏi chỉ quản trị viên hệ thống mở được, và mỗi lần mở đều bị ghi nhật ký."
        }
        meta={
          <>
            <span>Thứ Năm, 29 tháng 10 · 09:20</span>
            <span>{isAdmin ? "Toàn hệ thống" : course.label}</span>
          </>
        }
      />

      <PageState
        loading={
          <Panel>
            <div className={s.loading}>
              <Skeleton lines={2} />
              <Skeleton lines={8} />
            </div>
          </Panel>
        }
        empty={<Panel><EmptyState title="Chưa có yêu cầu nào trong hôm nay">Khi sinh viên hỏi hoặc việc nền chạy, mỗi yêu cầu gửi tới model sẽ xuất hiện ở đây kèm độ trễ và kết quả.</EmptyState></Panel>}
        error={{
          problem: "Không đọc được số liệu quan sát trong 5 phút gần nhất.",
          recovery: "Việc ghi nhật ký vẫn chạy, không mất dữ liệu. Thử lại; nếu vẫn lỗi, xem hàng chờ xử lý lỗi trước khi đổi cấu hình.",
        }}
      >
        <Panel>
          <StatusStrip items={STATUS_STRIP} />
        </Panel>

        {!isAdmin && (
          <Section panel title={`Lớp ${course.code} trong 7 ngày`} description="Số tổng hợp của lớp bạn phụ trách.">
            <DefinitionList items={(COURSE_SUMMARY[course.id] ?? []).map((m) => ({ term: m.label, value: m.hint ? `${m.value} · ${m.hint}` : m.value }))} />
          </Section>
        )}

        <Section panel
          title={`${rows.length} yêu cầu gần nhất`}
          description={isAdmin ? "Chọn một hàng để xem nội dung đã che, công cụ đã gọi và độ tin cậy." : "Giảng viên xem được số liệu; nội dung yêu cầu không mở được từ đây."}
          action={
            <SegmentedControl
              label="Lọc yêu cầu"
              value={filter}
              onChange={setFilter}
              options={[
                { value: "all", label: "Tất cả" },
                { value: "problem", label: "Lỗi và dự phòng" },
                { value: "privacy", label: "Có che thông tin" },
              ]}
            />
          }
        >
          <DataTable
            caption="Yêu cầu gửi tới model, mới nhất trước"
            columns={columns}
            rows={rows}
            rowKey={(r) => r.id}
            dense
            onRowClick={isAdmin ? (r) => setOpen(r) : undefined}
            activeKey={open?.id}
            empty={<p className={s.sub}>Không có yêu cầu nào khớp bộ lọc này.</p>}
          />
        </Section>

        {isAdmin && open && (
          <Drawer
            open
            onClose={closeDrawer}
            title={`${open.task} · ${open.id}`}
            description={`${fmtTime(open.at)} · ${open.model} · lớp ${open.courseId}`}
            footer={
              unlocked ? (
                <Button variant="ghost" iconEnd={<ExternalLink aria-hidden />} onClick={() => setTrace(true)}>
                  Chi tiết yêu cầu
                </Button>
              ) : undefined
            }
          >
            {!unlocked ? (
              <div className={s.gate}>
                <InlineNotice tone="privacy" title="Nội dung yêu cầu là dữ liệu của sinh viên">
                  Tên và mã số sinh viên đã được thay bằng nhãn, nhưng nội dung vẫn là câu hỏi của người học. Nhập lý do trước khi mở; lý do và tên bạn được ghi vào nhật ký kiểm toán.
                </InlineNotice>
                <Field
                  label="Lý do xem nội dung"
                  required
                  error={reasonError}
                  helper={reasonError ? undefined : "Ví dụ: xử lý khiếu nại của giảng viên về câu trả lời sai lúc 08:40."}
                >
                  {(id, describedBy) => (
                    <Textarea
                      id={id}
                      aria-describedby={describedBy}
                      rows={3}
                      value={reason}
                      invalid={Boolean(reasonError)}
                      onChange={(e) => setReason(e.target.value)}
                      onBlur={() => setReasonTouched(true)}
                      placeholder="Vì sao bạn cần xem yêu cầu này?"
                    />
                  )}
                </Field>
                <Button variant="primary" disabled={reason.trim().length < 10} onClick={() => setUnlocked(true)}>
                  Mở nội dung
                </Button>
              </div>
            ) : (
              <div className={s.detail}>
                <InlineNotice tone="success" compact>
                  Đã ghi nhật ký kiểm toán · {fmtTime(NOW)} · {user.name} · lý do: “{reason.trim()}”
                </InlineNotice>
                <h3 className={s.detailHead}>Nội dung đã che</h3>
                <p className={s.masked}>{open.masked}</p>
                <p className={s.sub}>Nhãn [[SV_1]] thay cho một sinh viên cụ thể. Muốn biết là ai, phải mở hồ sơ lớp và có lý do riêng.</p>
                <h3 className={s.detailHead}>Công cụ đã gọi</h3>
                <ul className={s.tools}>
                  {open.tools.map((t) => (
                    <li key={t} className={s.mono}>
                      {t}
                    </li>
                  ))}
                </ul>
                <h3 className={s.detailHead}>Số liệu</h3>
                <DefinitionList
                  items={[
                    { term: "Độ tin cậy", value: open.task === "Đánh chỉ mục tài liệu" ? "—" : fmtScore(open.confidence, 2) },
                    { term: "Độ trễ", value: `${fmtScore(open.latencyMs / 1000)} s` },
                    { term: "Token vào / ra", value: `${open.tokensIn} / ${open.tokensOut}` },
                    { term: "Kết quả", value: STATUS_LABEL[open.status] },
                    { term: "Sự kiện bảo vệ thông tin", value: open.privacy ?? "Không có" },
                  ]}
                />
                {trace && <p className={s.sub}>Mã theo dõi {open.id}-trace. Hệ thống theo dõi chi tiết nằm ngoài EduPilot và chưa được nối trong bản mô phỏng này.</p>}
              </div>
            )}
          </Drawer>
        )}
      </PageState>
    </Page>
  );
}
