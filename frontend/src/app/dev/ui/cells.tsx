"use client";

// Ô trạng thái của /dev/ui: mỗi khối có một hàm nhận trạng thái và trả về nội dung mẫu (SRS 7.2).
// Ô hover / focus chứa cùng phần tử với ô default — Playwright tự rê chuột / bấm Tab để đo (SRS 7.3).
import { useState, type ReactNode } from "react";
import { Box, Download, Search } from "lucide-react";
import {
  ActionList, ActionRow, Button, Checkbox, Composer, ConfirmIrreversible, DataTable, DefinitionList, Dialog, Drawer, EmptyState, FilterChips, Input, InlineNotice,
  MenuList, Page, Popover, SegmentedControl, Section, Select, Skeleton, StatusText, Switch, Tabs, Textarea, Toolbar, UndoLine, Field, type Column,
} from "@/shared/ui";
import { CommandPalette } from "@/shared/ui/CommandPalette";
import { CitationList, VerificationState } from "@/shared/domain";
import { navFor } from "@/shared/shell/nav";
import type { UiState } from "@/shared/ui/registry";
import { FIX } from "./fixtures";

export type Cell = (st: UiState, act: () => void, role: string) => ReactNode;

type Row = { id: string; name: string; score: string };
const ROWS: Row[] = [
  { id: "a", name: "Trần Minh Anh", score: "8,5" },
  { id: "b", name: "Lê Quốc Bảo", score: "7,0" },
  { id: "c", name: FIX.nameLong, score: "9,25" },
];
const COLS: Column<Row>[] = [
  { key: "name", header: "Sinh viên", render: (r) => r.name, primary: true },
  { key: "score", header: "Điểm", render: (r) => r.score, align: "end" },
];
const retry = (act: () => void) => (
  <Button size="sm" onClick={act}>
    Thử lại
  </Button>
);
const errorNote = (act: () => void, what = "Không tải được danh sách.") => (
  <InlineNotice tone="danger" title={what} action={retry(act)}>
    Dữ liệu của bạn vẫn an toàn. Kiểm tra kết nối rồi thử lại.
  </InlineNotice>
);
const emptyOne = (act: () => void, title = "Chưa có bài nộp nào", text = "Khi sinh viên nộp bài, bài sẽ hiện ở đây.", label = "Giao bài đầu tiên") => (
  <EmptyState title={title} action={<Button onClick={act}>{label}</Button>}>
    {text}
  </EmptyState>
);
const noop = () => {};

function OpenTrigger({ label, children }: { label: string; children: (open: boolean, close: () => void) => ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button data-open-overlay onClick={() => setOpen(true)}>
        {label}
      </Button>
      {children(open, () => setOpen(false))}
    </>
  );
}

function ConfirmDemo({ st }: { st: UiState }) {
  const [open, setOpen] = useState(false);
  const loading = st === "loading";
  return (
    <>
      <Button data-open-overlay onClick={() => setOpen(true)}>
        Chốt điểm
      </Button>
      <ConfirmIrreversible
        open={open}
        onClose={() => setOpen(false)}
        onConfirm={noop}
        title="Chốt điểm cuối kỳ?"
        consequence="Chốt 30 sinh viên; 2 em chưa có điểm cuối kỳ. Sau khi chốt không sửa được."
        confirmLabel="Chốt điểm"
        disabledReason={st === "disabled" ? "Còn 2 sinh viên chưa có điểm cuối kỳ." : undefined}
        loading={st === "loading" ? true : st === "error" ? false : undefined}
        error={st === "error" ? "Chưa chốt được: máy chủ không phản hồi. Điểm chưa bị thay đổi." : undefined}
      />
      {loading && null}
    </>
  );
}

function PaletteDemo({ st, role }: { st: UiState; role: string }) {
  const [open, setOpen] = useState(false);
  const groups = navFor(role === "student" ? "student" : "teacher");
  const items = groups.flatMap((g) => g.items);
  return (
    <>
      <Button data-open-overlay onClick={() => setOpen(true)} icon={<Search aria-hidden />}>
        Tìm nhanh
      </Button>
      <CommandPalette open={open} onClose={() => setOpen(false)} items={items} loading={st === "loading"} initialQuery={st === "empty" ? "zzzz" : ""} />
    </>
  );
}

function CitationDemo({ st, act }: { st: UiState; act: () => void }) {
  if (st === "loading") return <Skeleton lines={3} />;
  if (st === "error") return errorNote(act, "Không tải được nguồn tham khảo.");
  if (st === "empty") return <CitationList items={[]} />;
  return (
    <CitationList
      defaultOpenId={st === "selected" ? "1" : undefined}
      items={[
        { id: "1", title: "Chương 4 · Kế thừa và đa hình.pdf", page: 12, excerpt: "Lớp con thừa hưởng thuộc tính và phương thức của lớp cha, và có thể ghi đè phương thức." },
        { id: "2", title: FIX.urlLong, page: 3, excerpt: "Đa hình cho phép gọi cùng một phương thức trên các đối tượng khác kiểu." },
      ]}
    />
  );
}

function ComposerDemo({ st, act }: { st: UiState; act: () => void }) {
  const [v, setV] = useState(st === "empty" || st === "disabled" ? "" : "Em muốn hỏi về bài tập lớn thứ hai");
  const [failed, setFailed] = useState(false);
  // mất mạng thì gửi lỗi: chữ GIỮ NGUYÊN trong ô, hiện "Gửi lại" (US-PU-02 AC13)
  const send = () => {
    if (typeof navigator !== "undefined" && !navigator.onLine) setFailed(true);
    else {
      setFailed(false);
      act();
    }
  };
  return (
    <Composer
      value={v}
      onChange={setV}
      onSubmit={send}
      placeholder="Hỏi trợ lý về học phần…"
      busy={st === "loading"}
      onStop={st === "loading" ? act : undefined}
      disabled={st === "disabled"}
      hint={st === "empty" ? "Nhập câu hỏi để gửi." : undefined}
      error={st === "error" || failed ? "Chưa gửi được. Câu hỏi của bạn vẫn còn trong ô." : undefined}
      onRetry={st === "error" || failed ? send : undefined}
      tools={<Button variant="ghost" size="sm" icon={<Download aria-hidden />}>Đính kèm</Button>}
    />
  );
}

export const CELLS: Record<string, Cell> = {
  Button: (st, act) => (
    <Button variant="primary" disabled={st === "disabled"} loading={st === "loading"} onClick={act}>
      Lưu thay đổi
    </Button>
  ),
  Field: (st, act) => (
    <Field label="Họ và tên" helper={st === "empty" ? "Chưa nhập. Ghi đúng như trên thẻ sinh viên." : undefined} error={st === "error" ? "Họ và tên không được để trống. Nhập tên đầy đủ." : undefined} required>
      {(id, d) => <Input id={id} aria-describedby={d} disabled={st === "disabled"} invalid={st === "error"} defaultValue={st === "empty" || st === "error" ? "" : FIX.nameLong} onChange={act} />}
    </Field>
  ),
  Checkbox: (st, act) => (
    <div>
      <Checkbox label="Nhận thông báo qua email" disabled={st === "disabled"} defaultChecked={st === "selected"} aria-invalid={st === "error" || undefined} onChange={act} />
      {st === "error" && <p role="alert">Cần đồng ý điều khoản để tiếp tục.</p>}
    </div>
  ),
  Switch: (st, act) => <SwitchDemo st={st} act={act} />,
  Tabs: (st, act) => <TabsDemo st={st} act={act} kind="tabs" />,
  SegmentedControl: (st, act) => <TabsDemo st={st} act={act} kind="segmented" />,
  FilterChips: (st, act) => <TabsDemo st={st} act={act} kind="chips" />,
  ActionList: (st, act) => {
    if (st === "loading") return <Skeleton lines={3} />;
    if (st === "empty") return emptyOne(act, "Chưa có việc nào", "Việc cần làm sẽ hiện ở đây khi có.", "Xem lịch");
    if (st === "error") return errorNote(act, "Không tải được danh sách việc.");
    return (
      <ActionList label="Việc cần làm">
        <ActionRow tone="red" title="Chấm bài tập lớn 2" context="12 bài chờ chấm" meta="hạn hôm nay" selected={st === "selected"} onSelect={act} action={<Button size="sm" disabled={st === "disabled"} onClick={act}>Mở</Button>} />
        <ActionRow tone="amber" title={FIX.title120} context={FIX.nameLong} onSelect={act} />
      </ActionList>
    );
  },
  DataTable: (st, act) => {
    if (st === "loading") return <Skeleton lines={4} />;
    if (st === "error") return errorNote(act);
    return (
      <div aria-disabled={st === "disabled" || undefined}>
        <DataTable
          caption="Điểm bài tập lớn"
          columns={COLS}
          rows={st === "empty" ? [] : ROWS}
          rowKey={(r) => r.id}
          onRowClick={st === "disabled" ? undefined : act}
          activeKey={st === "selected" ? "b" : undefined}
          mobile="scroll"
          empty={emptyOne(act, "Chưa có điểm nào", "Điểm sẽ hiện khi bài được chấm.", "Mở hàng chờ chấm")}
        />
      </div>
    );
  },
  Popover: (st, act) => (
    <div style={{ minHeight: 170 }}>
      <Popover
        label="Bộ lọc"
        defaultOpen={st === "loading" || st === "empty"}
        trigger={(p) => (
          <Button disabled={st === "disabled"} aria-expanded={p["aria-expanded"]} aria-haspopup="true" onClick={() => { p.toggle(); act(); }}>
            Bộ lọc
          </Button>
        )}
      >
        {st === "loading" ? <div aria-busy="true" style={{ padding: 12 }}><Skeleton lines={2} /></div> : st === "empty" ? <div style={{ padding: 12 }}>{emptyOne(act, "Không có bộ lọc", "Chưa có bộ lọc nào được lưu.", "Tạo bộ lọc")}</div> : <div style={{ padding: 12 }}>Chỉ hiện bài chưa chấm</div>}
      </Popover>
    </div>
  ),
  Menu: (st, act) => (
    <MenuList
      items={[
        { label: "Đổi tên", onSelect: act },
        { label: "Nhân bản", onSelect: act, selected: st === "selected" ? true : undefined },
        { label: "Lưu trữ", onSelect: act, disabled: st === "disabled" },
      ]}
    />
  ),
  Dialog: (st) => (
    <OpenTrigger label="Mở hộp thoại">
      {(open, close) => (
        <Dialog open={open} onClose={close} title="Xoá tài liệu này?" description="Tài liệu sẽ bị gỡ khỏi tri thức của lớp." loading={st === "loading"} error={st === "error" ? "Chưa xoá được. Tài liệu vẫn còn nguyên." : undefined} footer={<Button variant="primary" onClick={close}>Xoá tài liệu</Button>}>
          {st === "default" || st === "focus" ? <p>Việc này không hoàn tác được.</p> : null}
        </Dialog>
      )}
    </OpenTrigger>
  ),
  Drawer: (st) => (
    <OpenTrigger label="Mở chi tiết">
      {(open, close) => (
        <Drawer open={open} onClose={close} title="Chi tiết bài nộp" loading={st === "loading"} error={st === "error" ? "Chưa tải được chi tiết. Thử lại sau ít phút." : undefined}>
          {st === "default" || st === "focus" ? <p>Bài nộp lúc 09:20, 3 tệp đính kèm.</p> : null}
        </Drawer>
      )}
    </OpenTrigger>
  ),
  ConfirmIrreversible: (st) => <ConfirmDemo st={st} />,
  Composer: (st, act) => <ComposerDemo st={st} act={act} />,
  InlineNotice: (st, act) =>
    st === "error" ? (
      errorNote(act, "Chưa lưu được.")
    ) : (
      <InlineNotice tone="info" title="Bài sẽ tự lưu" action={st === "focus" ? <Button size="sm" onClick={act}>Hiểu rồi</Button> : undefined}>
        Bản nháp được lưu trên máy này mỗi vài giây.
      </InlineNotice>
    ),
  StatusText: (st) => (st === "error" ? <StatusText tone="red">Quá hạn 2 ngày</StatusText> : <StatusText tone="green">Đã chấm</StatusText>),
  EmptyState: (_st, act) => emptyOne(act),
  Skeleton: (st) => <Skeleton lines={st === "loading" ? 4 : 3} />,
  UndoLine: (st, act) => (st === "error" ? <InlineNotice tone="danger" compact action={retry(act)}>Không hoàn tác được. Thử lại.</InlineNotice> : <UndoLine message="Đã đánh vắng 3 sinh viên" onUndo={act} onDone={noop} />),
  Layout: (st, act) => {
    if (st === "loading") return <Skeleton lines={5} />;
    if (st === "error") return errorNote(act, "Không tải được trang.");
    return (
      <Page width="reading">
        <Section title="Thông tin lớp" description="Cập nhật lần cuối hôm nay">
          <Toolbar end={<Button size="sm" onClick={act}>Chỉnh sửa</Button>}>
            <StatusText tone="green">Đang mở</StatusText>
          </Toolbar>
          <DefinitionList items={[{ term: "Học phần", value: "Lập trình hướng đối tượng" }, { term: "Sĩ số", value: FIX.bigNumber }]} />
        </Section>
      </Page>
    );
  },
  CommandPalette: (st, _act, role) => <PaletteDemo st={st} role={role} />,
  CitationList: (st, act) => <CitationDemo st={st} act={act} />,
  VerificationState: (st, act, role) => (
    <VerificationState status="pending" canReview={role !== "student"} disabled={st === "disabled"} onConfirm={act} onEdit={act} onReject={act}>
      <Box className="ep-sr-only" aria-hidden />
      Đáp án gợi ý: dùng kế thừa để chia sẻ phương thức chung.
    </VerificationState>
  ),
};

function SwitchDemo({ st, act }: { st: UiState; act: () => void }) {
  const [on, setOn] = useState(st === "selected");
  return (
    <div aria-busy={st === "loading" || undefined}>
      <Switch label="Cho phép nộp muộn" description={st === "loading" ? "Đang lưu…" : undefined} checked={on} disabled={st === "disabled" || st === "loading"} onChange={(n) => { setOn(n); act(); }} />
    </div>
  );
}

function TabsDemo({ st, act, kind }: { st: UiState; act: () => void; kind: "tabs" | "segmented" | "chips" }) {
  const [v, setV] = useState<"a" | "b" | "c">(st === "selected" ? "b" : "a");
  const [chips, setChips] = useState<Array<"a" | "b" | "c">>(st === "selected" ? ["b"] : []);
  const opts = [
    { value: "a" as const, label: "Tất cả", count: 12 },
    { value: "b" as const, label: "Chưa chấm", count: 4 },
    { value: "c" as const, label: "Đã chấm", disabled: st === "disabled" },
  ];
  const on = (x: "a" | "b" | "c") => { setV(x); act(); };
  if (kind === "tabs") return <Tabs label="Bài nộp" value={v} onChange={on} options={opts} />;
  if (kind === "segmented") return <SegmentedControl label="Chế độ xem" value={v} onChange={on} options={opts} />;
  return <FilterChips label="Lọc" value={chips} onChange={(n) => { setChips(n); act(); }} options={opts} />;
}

// Dùng ở phần "Phụ" của trang: các Textarea / Select không nằm trong 24 khối nhưng thuộc họ Field.
export const FieldExtras = () => (
  <div style={{ display: "grid", gap: 12 }}>
    <Field label="Ghi chú">{(id) => <Textarea id={id} rows={2} defaultValue={FIX.title120} />}</Field>
    <Field label="Lớp">{(id) => <Select id={id} defaultValue="1"><option value="1">LTHDT · Nhóm 1</option></Select>}</Field>
  </div>
);
