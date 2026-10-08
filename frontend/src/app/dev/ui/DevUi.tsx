"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ButtonLink, Button, Kbd, PageHeader, PrivateMark, Section, SegmentedControl, Drawer, ActionList, ActionRow, DataTable, type Column } from "@/shared/ui";
import { TrendChart } from "@/shared/ui/Chart";
import { VerificationState } from "@/shared/domain";
import { NA_REASONS, REGISTRY, STATES } from "@/shared/ui/registry";
import { CELLS } from "./cells";
import { FIX } from "./fixtures";
import { useQueryParam } from "./useQueryParam";
import s from "./DevUi.module.css";

const WIDTHS = ["375", "900", "1280", "1440"] as const;

function StateCell({ block, st, role }: { block: string; st: (typeof STATES)[number]; role: string }) {
  const [calls, setCalls] = useState(0);
  const act = useCallback(() => setCalls((n) => n + 1), []);
  return (
    <div className={s.cell} data-part="state-cell" data-state={st} data-calls={calls}>
      <span className={s.state}>{st}</span>
      <div className={s.region} data-part="work-region">
        {CELLS[block](st, act, role)}
      </div>
    </div>
  );
}

type Big = { id: string; name: string; score: string };
const BIG: Big[] = Array.from({ length: 1000 }, (_, i) => ({ id: `r${i}`, name: `Sinh viên ${String(i + 1).padStart(4, "0")}`, score: ((i * 37) % 100 / 10).toFixed(1).replace(".", ",") }));
const BIG_COLS: Column<Big>[] = [
  { key: "no", header: "Mã", render: (r) => r.id, width: "20%" },
  { key: "name", header: "Sinh viên", render: (r) => r.name },
  { key: "score", header: "Điểm", render: (r) => r.score, align: "end" },
];

function BigTable() {
  const [open, setOpen] = useState<Big | null>(null);
  return (
    <div data-part="datatable-1000">
      <DataTable caption="1.000 sinh viên" columns={BIG_COLS} rows={BIG} rowKey={(r) => r.id} onRowClick={setOpen} activeKey={open?.id} virtual={{ height: 420 }} />
      <Drawer open={open !== null} onClose={() => setOpen(null)} title={open?.name ?? ""}>
        <p>Điểm: {open?.score}</p>
      </Drawer>
    </div>
  );
}

type Item = { id: string; name: string };
function CursorDemo() {
  const [rows, setRows] = useState<Item[]>(() => Array.from({ length: 20 }, (_, i) => ({ id: `p1-${i}`, name: `Dòng trang 1 · ${i + 1}` })));
  const [next, setNext] = useState<string | null>("c2");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const failed = useRef(false);
  const failTwo = useQueryParam("fail2") === "1";
  const log = useRef<string[]>([]);
  useEffect(() => {
    (window as unknown as { __cursorCalls: string[] }).__cursorCalls = log.current;
  }, []);
  const load = (cursor: string) => {
    log.current.push(cursor);
    setLoading(true);
    setError(null);
    window.setTimeout(() => {
      setLoading(false);
      if (cursor === "c2" && failTwo && !failed.current) {
        failed.current = true;
        setError("Không tải được trang tiếp theo. Các dòng đã có vẫn còn nguyên.");
        return;
      }
      const page = Number(cursor.slice(1));
      setRows((r) => [...r, ...Array.from({ length: 20 }, (_, i) => ({ id: `p${page}-${i}`, name: `Dòng trang ${page} · ${i + 1}` }))]);
      setNext(page >= 3 ? null : `c${page + 1}`);
    }, 250);
  };
  return (
    <div data-part="cursor-demo">
      <DataTable caption="Phân trang con trỏ" columns={[{ key: "n", header: "Tên", render: (r: Item) => r.name }]} rows={rows} rowKey={(r) => r.id} virtual={{ height: 240 }} pagination={{ nextCursor: next, onLoadMore: load, loading, error }} />
    </div>
  );
}

function ClsDemo() {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const t = window.setTimeout(() => setReady(true), 600);
    return () => window.clearTimeout(t);
  }, []);
  return (
    <div className={s.layer} data-part="cls-demo" data-loaded={ready || undefined}>
      <Section title="Danh sách việc" description={ready ? "3 việc cần làm hôm nay" : "\u00a0"}>
        {ready ? (
          <ActionList label="Việc hôm nay">
            {[1, 2, 3].map((i) => (
              <ActionRow key={i} tone="neutral" title={`Việc số ${i}`} context="Một dòng bối cảnh ngắn" />
            ))}
          </ActionList>
        ) : (
          <ActionList label="Việc hôm nay" loading={3} />
        )}
      </Section>
      <Section title="Điểm">
        <DataTable
          caption="Điểm"
          mobile="scroll"
          columns={[{ key: "n", header: "Sinh viên", render: (r: Item) => r.name }, { key: "d", header: "Điểm", render: () => "8,0", align: "end" }]}
          rows={ready ? [1, 2, 3].map((i) => ({ id: String(i), name: `Sinh viên ${i}` })) : []}
          loading={ready ? undefined : 3}
          rowKey={(r) => r.id}
        />
      </Section>
    </div>
  );
}

export default function DevUi() {
  const role = useQueryParam("as") ?? "teacher";
  const [w, setW] = useState<(typeof WIDTHS)[number]>("1440");
  const long = [FIX.title120, FIX.nameLong, FIX.diacritics, FIX.urlLong, FIX.bigNumber];
  return (
    <div className={s.wrap} style={{ "--w": `${w}px` } as React.CSSProperties}>
      <PageHeader
        title="Thư viện thành phần"
        description="25 khối × 8 trạng thái, chỉ có ở bản dev."
        actions={<SegmentedControl label="Bề rộng xem thử" value={w} onChange={setW} options={WIDTHS.map((x) => ({ value: x, label: `${x}` }))} />}
      />

      <main className={s.sub} data-part="matrix">
        {REGISTRY.map((b) => (
          <section key={b.name} className={s.block} data-part="primitive" data-name={b.name} aria-label={b.name}>
            <h2 className="ep-section-title">{b.name}</h2>
            <div className={s.cells}>
              {b.states.map((st) => (
                <StateCell key={st} block={b.name} st={st} role={role} />
              ))}
            </div>
            <ul className={s.na} aria-label={`${b.name} · không áp dụng`}>
              {STATES.filter((st) => b.na[st]).map((st) => (
                <li key={st} data-part="state-cell" data-na data-state={st} data-reason={NA_REASONS[b.na[st]!]}>
                  {st}: {NA_REASONS[b.na[st]!]}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </main>

      <section className={s.sub} aria-label="Phụ">
        <h2 className="ep-section-title">Phụ</h2>
        <div data-part="work-region" className={s.sub}>
          <ButtonLink href="/login" variant="secondary">Liên kết dạng nút</ButtonLink>
          <p>
            Phím tắt <Kbd>⌘K</Kbd> · <PrivateMark />
          </p>
          <TrendChart label="Điểm trung bình" points={[{ x: "T1", y: 6.5 }, { x: "T2", y: 7.2 }, { x: "T3", y: 7.8 }]} format={(v) => v.toFixed(1)} />
        </div>
      </section>

      <section className={s.sub} aria-label="Bốn lời xác nhận" data-part="verification-four">
        <h2 className="ep-section-title">Bốn lời của VerificationState</h2>
        <div data-part="work-region" className={s.sub}>
          <VerificationState status="pending" canReview={role !== "student"}>Nháp AI: dùng kế thừa để chia sẻ phương thức chung.</VerificationState>
          <VerificationState status="verified" verifier="Lê Thu Hà">Dùng kế thừa để chia sẻ phương thức chung.</VerificationState>
          <VerificationState status="edited" onViewOriginal={() => {}}>Dùng kế thừa khi quan hệ là “là một”.</VerificationState>
          <VerificationState status="awaiting">Câu hỏi của bạn đã được ghi lại.</VerificationState>
        </div>
      </section>

      <section className={s.sub} aria-label="Bảng 1.000 dòng">
        <h2 className="ep-section-title">Bảng 1.000 dòng (ảo hoá)</h2>
        <BigTable />
      </section>

      <section className={s.sub} aria-label="Phân trang con trỏ">
        <h2 className="ep-section-title">Phân trang con trỏ</h2>
        <CursorDemo />
      </section>

      <section className={s.sub} aria-label="Khung xương khớp hình">
        <h2 className="ep-section-title">Khung xương → nội dung</h2>
        <ClsDemo />
      </section>

      <section className={s.long} aria-label="Chuỗi dài" data-part="overflow">
        <h2 className="ep-section-title">Chuỗi Việt dài</h2>
        {long.map((t) => (
          <p key={t}>{t}</p>
        ))}
        <Button variant="secondary">{FIX.title120}</Button>
      </section>
    </div>
  );
}
