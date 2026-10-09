"use client";

import { Suspense, type ReactNode } from "react";
import { ActionList, ActionRow, Button, Composer, DataTable, DefinitionList, EmptyState, PageState, Panel, PanelSection, Section, Tabs } from "@/shared/ui";
import s from "./PanelMatrix.module.css";

type Row = { id: string; name: string; score: string };
const ROWS: Row[] = [1, 2, 3].map((i) => ({ id: `r${i}`, name: `Sinh viên ${i}`, score: `${6 + i},0` }));
const COLS = [{ key: "n", header: "Sinh viên", render: (r: Row) => r.name }, { key: "d", header: "Điểm", render: (r: Row) => r.score, align: "end" as const }];

function Cell({ name, children }: { name: string; children: ReactNode }) {
  return (
    <div className={s.cell} data-part="panel-cell" data-cell={name}>
      <span className={s.label}>{name}</span>
      {children}
    </div>
  );
}

const Strong = ({ n }: { n: number }) => (
  <div className={s.strongRow}>
    {Array.from({ length: n }, (_, i) => (
      <PanelSection key={i} tone="strong">
        <span className="ep-section-title">{12 - i * 4}</span> bài chờ chấm
      </PanelSection>
    ))}
  </div>
);

/** Ca vi phạm cố ý cho phép kiểm STRONG / WALL ở panels.spec.ts: `/dev/ui?fixture=strong4` (4 ô nhấn trong một panel) và `?fixture=wall` (3 Panel bằng nhau cùng hàng ở đầu trang). DevUi chỉ vẽ ca này. */
export function PanelFixture({ kind }: { kind: "strong4" | "wall" }) {
  if (kind === "strong4") {
    return (
      <div data-part="strong4-fixture">
        <Panel aria-label="Ca vi phạm: 4 ô nhấn">
          <Strong n={4} />
        </Panel>
      </div>
    );
  }
  return (
    <div className={s.wall} data-part="wall-fixture">
      {[1, 2, 3].map((i) => (
        <Panel key={i} aria-label={`Ca vi phạm: panel ${i}`}>
          <PanelSection>Số liệu {i}</PanelSection>
        </Panel>
      ))}
    </div>
  );
}

/** US-UI-02 AC7: mục "Panel" của /dev/ui — ma trận ≥ 8 ô; mỗi ô đạt AUDIT sạch ở 1440 và 390 px. */
export function PanelMatrix() {
  return (
    <section id="panel" className={s.root} data-part="panel-matrix" aria-label="Panel">
      <h2 className="ep-section-title">Panel</h2>
      <div className={s.cells}>
        {(["md", "lg", "none"] as const).map((pad) => (
          <Cell key={`c-${pad}`} name={`${pad} · chỉ nội dung`}>
            <Panel padding={pad} aria-label={`${pad} · chỉ nội dung`}>
              <p className={pad === "none" ? s.padText : s.text}>Nội dung một vùng làm việc.</p>
            </Panel>
          </Cell>
        ))}
        {(["md", "lg", "none"] as const).map((pad) => (
          <Cell key={`s-${pad}`} name={`${pad} · hai PanelSection`}>
            <Panel padding={pad} aria-label={`${pad} · hai PanelSection`}>
              <PanelSection title="Nhóm thứ nhất">
                <p className={s.text}>Hai nhóm liền nhau ngăn bằng một đường kẻ 1 px.</p>
              </PanelSection>
              <PanelSection title="Nhóm thứ hai" action={<Button size="sm" variant="secondary">Sửa</Button>}>
                <p className={s.text}>Không viền, không bóng, không bo góc riêng.</p>
              </PanelSection>
            </Panel>
          </Cell>
        ))}
        <Cell name="1 ô nhấn">
          <Panel aria-label="1 ô nhấn">
            <Strong n={1} />
            <PanelSection title="Chi tiết"><p className={s.text}>Ô nhấn chỉ cho số liệu quan trọng.</p></PanelSection>
          </Panel>
        </Cell>
        <Cell name="3 ô nhấn">
          <Panel aria-label="3 ô nhấn">
            <Strong n={3} />
          </Panel>
        </Cell>
        <Cell name="DataTable trong panel">
          <Panel padding="none" aria-label="DataTable trong panel">
            <DataTable caption="Điểm" columns={COLS} rows={ROWS} rowKey={(r) => r.id} />
          </Panel>
        </Cell>
        <Cell name="ActionList trong panel">
          <Panel aria-label="ActionList trong panel">
            <ActionList label="Việc">
              {[1, 2, 3].map((i) => (
                <ActionRow key={i} tone={i === 1 ? "red" : "neutral"} title={`Việc số ${i}`} context="Một dòng bối cảnh ngắn" action={i === 1 ? <Button size="sm" variant="primary">Làm</Button> : undefined} />
              ))}
            </ActionList>
          </Panel>
        </Cell>
        <Cell name="Tabs + DefinitionList trong panel">
          <Panel aria-label="Tabs + DefinitionList trong panel">
            <Tabs label="Mục" value="a" onChange={() => {}} options={[{ value: "a", label: "Thông tin" }, { value: "b", label: "Câu hỏi" }]} />
            <DefinitionList items={[{ term: "Thời lượng", value: "45 phút" }, { term: "Chấm điểm", value: "Từng phần" }]} />
          </Panel>
        </Cell>
        <Cell name="Composer trong panel">
          <Panel aria-label="Composer trong panel">
            <p className={s.text}>Cuộc trò chuyện mẫu.</p>
            <Composer value="" onChange={() => {}} onSubmit={() => {}} placeholder="Nhập câu hỏi" />
          </Panel>
        </Cell>
        <Suspense fallback={null}>
          {(["loading", "empty", "error"] as const).map((st) => (
            <Cell key={st} name={`PageState ${st}`}>
              <Panel aria-label={`PageState ${st}`}>
                <PageState state={st} empty={<EmptyState title="Chưa có gì ở đây.">Khi có dữ liệu, nó hiện ở đây.</EmptyState>}>
                  <Section title="Nội dung"><p className={s.text}>Có dữ liệu.</p></Section>
                </PageState>
              </Panel>
            </Cell>
          ))}
        </Suspense>
      </div>
    </section>
  );
}
