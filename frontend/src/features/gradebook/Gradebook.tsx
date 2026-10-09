"use client";

import { useMemo, useState } from "react";
import { FileDown, History, Lock, NotebookPen, Scale, SquareCheckBig } from "lucide-react";
import {
  Button,
  ButtonLink,
  ConfirmIrreversible,
  type Column,
  DataTable,
  Drawer,
  EmptyState,
  InlineNotice,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  Skeleton,
  StatusText,
  Toolbar,
} from "@/shared/ui";
import { COURSE_2, fmtScore, type Student } from "@/mock/core";
import { GRADE_AUDIT_KEY, fmtAuditAt, gradeAuditOf, logGrade, type GradeAudit } from "@/mock/audit";
import { rosterOf } from "@/mock/roster";
import { attendanceStats, btScoresFor, calcQt, SCHEME_1, SCHEME_2_DRAFT } from "@/mock/grades";
import { BT03_SEED } from "@/mock/assess";
import { ATTENDANCE_SEED, KEYS, MEMBERS_SEED, SCHEMES_SEED, type AttendanceState, type Bt03State, type MembersState, type SchemesState } from "@/mock/state";
import { CELL_HISTORY, CONFLICT_CELL, OFFICIAL_GRADE_NOTE } from "@/mock/gradebook";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useProgressiveCount } from "@/shared/lib/useProgressiveCount";
import s from "./Gradebook.module.css";

const EDITABLE = ["bt01", "bt02", "ck"] as const;
type EditCol = (typeof EDITABLE)[number];
type Edits = Record<string, number | null>;

const COL_LABEL: Record<EditCol, string> = { bt01: "Bài tập 01", bt02: "Bài tập 02", ck: "Cuối kỳ" };

/** "8,5" / "8.5" / "" → số hoặc null; ngoài thang 0–10 thì bỏ qua. */
function parseScore(raw: string): number | null | undefined {
  const t = raw.trim().replace(",", ".");
  if (t === "") return null;
  const n = Number(t);
  if (!Number.isFinite(n) || n < 0 || n > 10) return undefined;
  return Math.round(n * 100) / 100;
}

type Row = {
  st: Student;
  bt01: number | null;
  bt02: number | null;
  bt03: number | null;
  ck: number | null;
  bonus: number;
  penalty: number;
  avg: number | null;
  qt: number | null;
  speaks: number;
  absences: number;
};

export function Gradebook() {
  const { role, course, user } = useSession();
  const [audit] = useDemoSlice<GradeAudit[]>(GRADE_AUDIT_KEY, []);
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [edits, setEdits] = useDemoSlice<Edits>("gradebook.edits", {});
  const [editing, setEditing] = useState<{ id: string; col: EditCol } | null>(null);
  const [draft, setDraft] = useState("");
  const [invalid, setInvalid] = useState<string | null>(null);
  const [conflict, setConflict] = useState<{ id: string; col: EditCol; mine: number | null } | null>(null);
  const [conflictDone, setConflictDone] = useState(false);
  const [explain, setExplain] = useState<string | null>(null);
  const [historyOf, setHistoryOf] = useState<string | null>(null);
  const [finalizing, setFinalizing] = useState(false);
  const undo = useUndoLine();

  const canEdit = role === "teacher";
  const schemeConfirmed = (schemes[course.id]?.status ?? "none") === "confirmed";
  const scheme = course.id === COURSE_2 ? SCHEME_2_DRAFT : SCHEME_1;
  const hasGrades = course.id !== COURSE_2;

  const rows = useMemo<Row[]>(
    () =>
      rosterOf(course.id, members).map((st) => {
        const base = btScoresFor(st, bt03);
        const at = attendanceStats(st, attendance);
        const bt01 = edits[`${st.id}:bt01`] !== undefined ? edits[`${st.id}:bt01`] : base.bt01;
        const bt02 = edits[`${st.id}:bt02`] !== undefined ? edits[`${st.id}:bt02`] : base.bt02;
        const ck = edits[`${st.id}:ck`] ?? null;
        const q = calcQt({ scores: [bt01, bt02, base.bt03], speaks: at.speaks, absences: at.absences }, scheme);
        return { st, bt01, bt02, bt03: base.bt03, ck, bonus: q.bonus, penalty: q.penalty, avg: q.avg, qt: q.qt, speaks: at.speaks, absences: at.absences };
      }),
    [course.id, members, bt03, attendance, edits, scheme],
  );

  const shown = useProgressiveCount(rows.length);
  const missingCk = rows.filter((r) => r.ck === null).length;
  const explainRow = rows.find((r) => r.st.id === explain);
  const historyStudent = rows.find((r) => r.st.id === historyOf);

  /** Trả `false` khi giá trị không hợp lệ: ô ở lại, báo lỗi tại ô, không lặng lẽ quay về giá trị cũ (03-9). */
  function commit(id: string, col: EditCol, raw: string): boolean {
    const parsed = parseScore(raw);
    const key = `${id}:${col}`;
    const was = rows.find((r) => r.st.id === id)?.[col] ?? null;
    if (parsed === undefined) {
      setInvalid(key);
      return false;
    }
    setInvalid(null);
    if (parsed !== was) logGrade(id, user.name, `Sửa ${COL_LABEL[col]}: ${was === null ? "—" : fmtScore(was)} → ${parsed === null ? "—" : fmtScore(parsed)}`);
    const before = edits[key];
    setEdits((prev) => ({ ...prev, [key]: parsed }));
    if (id === CONFLICT_CELL.studentId && col === CONFLICT_CELL.column && !conflictDone) {
      setConflict({ id, col, mine: parsed });
      return true;
    }
    undo.push(`Đã sửa ${COL_LABEL[col]} của ${rows.find((r) => r.st.id === id)?.st.name ?? ""}`, () =>
      setEdits((prev) => ({ ...prev, [key]: before === undefined ? null : before })),
    );
    return true;
  }

  function startEdit(id: string, col: EditCol, value: number | null) {
    setInvalid(null);
    setEditing({ id, col });
    setDraft(value === null ? "" : fmtScore(value));
  }

  function moveFrom(id: string, col: EditCol, dir: "down" | "right") {
    const i = rows.findIndex((r) => r.st.id === id);
    if (dir === "down") {
      const next = rows[i + 1];
      if (next) startEdit(next.st.id, col, next[col]);
      else setEditing(null);
      return;
    }
    const c = EDITABLE.indexOf(col);
    if (c < EDITABLE.length - 1) {
      const nextCol = EDITABLE[c + 1];
      startEdit(id, nextCol, rows[i][nextCol]);
    } else if (rows[i + 1]) {
      startEdit(rows[i + 1].st.id, EDITABLE[0], rows[i + 1][EDITABLE[0]]);
    } else setEditing(null);
  }

  function cell(row: Row, col: EditCol | "bt03") {
    const value = row[col];
    const text = value === null ? "—" : fmtScore(value);
    if (!canEdit || col === "bt03") {
      return (
        <span className={s.cellRead} data-empty={value === null}>
          {text}
        </span>
      );
    }
    const on = editing?.id === row.st.id && editing.col === col;
    if (on) {
      const bad = invalid === `${row.st.id}:${col}`;
      return (
        <>
          <input
            className={s.input}
            autoFocus
            inputMode="decimal"
            aria-label={`${COL_LABEL[col]} của ${row.st.name}`}
            aria-invalid={bad || undefined}
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value);
              setInvalid(null);
            }}
            onBlur={() => {
              if (commit(row.st.id, col, draft)) setEditing(null);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                if (commit(row.st.id, col, draft)) moveFrom(row.st.id, col, "down");
              } else if (e.key === "Tab") {
                e.preventDefault();
                if (commit(row.st.id, col, draft)) moveFrom(row.st.id, col, "right");
              } else if (e.key === "Escape") {
                setInvalid(null);
                setEditing(null);
              }
            }}
          />
          {bad && (
            <span role="alert" className={s.invalid}>
              Điểm là số từ 0 đến 10, ví dụ 7,5
            </span>
          )}
        </>
      );
    }
    return (
      <button type="button" className={s.cell} data-empty={value === null} onClick={() => startEdit(row.st.id, col, value)}>
        {text}
      </button>
    );
  }

  const columns: Column<Row>[] = [
    {
      key: "student",
      header: "Sinh viên",
      frozen: true,
      width: "208px",
      mobileWidth: "132px",
      render: (r) => (
        <span className={s.who}>
          <span className={s.name}>{r.st.name}</span>
          <span className={s.code}>{r.st.code}</span>
        </span>
      ),
    },
    { key: "qt", header: "QT tạm tính", align: "end", width: "104px", mobileWidth: "76px", part: "col-qt", render: (r) => <span className={s.qt}>{r.qt === null ? "—" : fmtScore(r.qt)}</span> },
    {
      key: "state",
      header: "Trạng thái",
      width: "150px",
      mobileWidth: "112px",
      part: "col-status",
      render: (r) =>
        r.bt01 === null || r.bt02 === null ? (
          <StatusText tone="red">Thiếu bài tập</StatusText>
        ) : r.ck === null ? (
          <StatusText tone="amber">Chưa có điểm cuối kỳ</StatusText>
        ) : (
          <StatusText tone="green">Đủ điểm, chờ chốt</StatusText>
        ),
    },
    { key: "bt01", header: "BT01", align: "end", width: "88px", render: (r) => cell(r, "bt01") },
    { key: "bt02", header: "BT02", align: "end", width: "88px", render: (r) => cell(r, "bt02") },
    { key: "bt03", header: "BT03", align: "end", width: "88px", render: (r) => cell(r, "bt03") },
    { key: "bonus", header: "Cộng", align: "end", width: "80px", render: (r) => <span className={s.delta}>{r.bonus > 0 ? `+${fmtScore(r.bonus, 2)}` : "—"}</span> },
    { key: "penalty", header: "Trừ", align: "end", width: "80px", render: (r) => <span className={s.delta}>{r.penalty > 0 ? `−${fmtScore(r.penalty, 2)}` : "—"}</span> },
    { key: "ck", header: "Cuối kỳ", align: "end", width: "96px", render: (r) => cell(r, "ck") },
    {
      key: "menu",
      header: "",
      width: "48px",
      align: "end",
      render: (r) => (
        <OverflowMenu
          label={`Hành động với ${r.st.name}`}
          items={[
            { label: "Xem giải trình điểm", icon: <Scale aria-hidden />, onSelect: () => setExplain(r.st.id) },
            { label: "Lịch sử sửa điểm", icon: <History aria-hidden />, onSelect: () => setHistoryOf(r.st.id) },
          ]}
        />
      ),
    },
  ];

  const finalizeReason =
    missingCk > 0
      ? `Chốt ${rows.length} sinh viên; ${missingCk} chưa có điểm cuối kỳ — chưa chốt được. Nhập đủ cột Cuối kỳ rồi chốt lại.`
      : undefined;

  return (
    <Page width="wide">
      <PageHeader
        title="Sổ điểm"
        description={`${course.label} · ${course.schedule}`}
        meta={
          <>
            <span>{rows.length} sinh viên</span>
            <span>Công thức: quá trình {scheme.qtWeight}% · cuối kỳ {100 - scheme.qtWeight}%</span>
            <span>{schemeConfirmed ? "Công thức đã xác nhận" : "Công thức chưa xác nhận"}</span>
          </>
        }
        actions={
          <>
            <ButtonLink href="/gradebook/scheme" icon={<NotebookPen aria-hidden />}>
              Xem công thức
            </ButtonLink>
            {role === "teacher" &&
              (schemeConfirmed ? (
                <Button variant="primary" icon={<SquareCheckBig aria-hidden />} onClick={() => setFinalizing(true)}>
                  Chốt điểm
                </Button>
              ) : (
                <span className={s.lock}>
                  <Button variant="primary" className={s.lockBtn} aria-disabled="true" icon={<Lock aria-hidden />} onClick={() => undefined}>
                    Chốt điểm
                  </Button>
                  <span role="note" className={s.lockWhy}>
                    Chưa chốt được: công thức điểm của lớp này chưa được xác nhận. Mở Công thức điểm để xác nhận trước.
                  </span>
                </span>
              ))}
          </>
        }
      />

      {!schemeConfirmed && (
        <InlineNotice
          tone="warning"
          title="Công thức điểm chưa xác nhận"
          action={
            <ButtonLink size="sm" href="/gradebook/scheme">
              Mở công thức điểm
            </ButtonLink>
          }
        >
          Điểm quá trình ở bảng này chỉ là tạm tính theo bản nháp trích từ quy chế. Chưa chốt được điểm cho tới khi công thức được xác nhận.
        </InlineNotice>
      )}

      <PageState
        loading={<Skeleton lines={12} />}
        empty={
          <EmptyState title="Lớp chưa có điểm" icon={<NotebookPen aria-hidden />} action={<ButtonLink href="/grading">Mở hàng chờ chấm bài</ButtonLink>}>
            Lớp này chưa có bài tập nào được công bố điểm. Khi bạn công bố điểm một bài tập, cột điểm và điểm quá trình tạm tính sẽ xuất hiện ở đây.
          </EmptyState>
        }
        error={{
          problem: "Không tải được sổ điểm của lớp này.",
          recovery: "Điểm đã lưu không bị mất. Thử lại, hoặc mở lại từ Hôm nay sau ít phút.",
        }}
        state={hasGrades ? undefined : "empty"}
      >

        <Toolbar
          end={
            <OverflowMenu
              label="Thêm hành động với sổ điểm"
              items={[
                {
                  label: "Xuất XLSX",
                  icon: <FileDown aria-hidden />,
                  onSelect: () => undo.push("Đã tạo file sổ điểm (mô phỏng) · gồm cột điểm thành phần và dòng ghi chú điểm chính thức"),
                },
              ]}
            />
          }
        >
          <span className={s.hint}>
            {canEdit ? "Bấm vào ô điểm để sửa tại chỗ · Enter xuống dòng, Tab sang cột" : "Trợ giảng xem sổ điểm, không sửa ô và không chốt điểm"}
          </span>
        </Toolbar>

        {conflict && (
          <InlineNotice
            tone="warning"
            title={`${CONFLICT_CELL.by} vừa sửa ô này thành ${CONFLICT_CELL.otherValue}`}
            action={
              <>
                <Button
                  size="sm"
                  onClick={() => {
                    setConflictDone(true);
                    setConflict(null);
                  }}
                >
                  Giữ của tôi
                </Button>
                <Button
                  size="sm"
                  variant="primary"
                  onClick={() => {
                    setEdits((prev) => ({ ...prev, [`${conflict.id}:${conflict.col}`]: 8 }));
                    setConflictDone(true);
                    setConflict(null);
                  }}
                >
                  Dùng bản mới
                </Button>
              </>
            }
          >
            Ô {COL_LABEL[conflict.col]} của {rows.find((r) => r.st.id === conflict.id)?.st.name} đã được người khác sửa lúc {CONFLICT_CELL.at}. Bản của bạn là{" "}
            {conflict.mine === null ? "trống" : fmtScore(conflict.mine)}.
          </InlineNotice>
        )}

        {undo.node}

        <DataTable
          caption={`Sổ điểm ${course.label}`}
          columns={columns}
          rows={rows.slice(0, shown)}
          rowKey={(r) => r.st.id}
          rowAttrs={(r) => ({ "data-part": "student-row", "data-student-id": r.st.id })}
          mobile="scroll"
          scrollHint="Kéo ngang để xem BT01–BT03 và Cuối kỳ"
          empty={<EmptyState title="Lớp chưa có điểm">Chưa có sinh viên nào trong lớp này.</EmptyState>}
        />
        <p className={s.note}>{OFFICIAL_GRADE_NOTE}</p>
      </PageState>

      <Drawer
        open={Boolean(explainRow)}
        onClose={() => setExplain(null)}
        title={explainRow ? `Giải trình điểm · ${explainRow.st.name}` : ""}
        description={explainRow ? `${explainRow.st.code} · ${course.label}` : undefined}
      >
        {explainRow && <Explain row={explainRow} published={bt03.status === "published"} />}
      </Drawer>

      <Drawer
        open={Boolean(historyStudent)}
        onClose={() => setHistoryOf(null)}
        title={historyStudent ? `Lịch sử sửa điểm · ${historyStudent.st.name}` : ""}
        description="Mỗi thay đổi điểm đều ghi lại người sửa và thời điểm."
      >
        <div className={s.history}>
          {[...gradeAuditOf(audit, historyStudent?.st.id ?? "").map((a) => ({ at: fmtAuditAt(a.atMs), by: a.by, text: a.text })), ...CELL_HISTORY].map((h) => (
            <div key={`${h.at}-${h.text}`} className={s.historyRow}>
              <span>{h.text}</span>
              <span className={s.historyMeta}>
                {h.at} · {h.by}
              </span>
            </div>
          ))}
        </div>
      </Drawer>

      {role === "teacher" && (
        <ConfirmIrreversible
          open={finalizing}
          onClose={() => setFinalizing(false)}
          onConfirm={() => undo.push(`Đã chốt điểm ${rows.length} sinh viên (mô phỏng)`)}
          title="Chốt điểm học phần"
          consequence={`Chốt ${rows.length} sinh viên. Sau khi chốt, điểm không sửa được trong EduPilot và được gửi sang hệ thống quản lý đào tạo của trường.`}
          disabledReason={finalizeReason}
          confirmLabel="Chốt điểm"
        />
      )}
    </Page>
  );
}

function Explain({ row, published }: { row: Row; published: boolean }) {
  const got = [
    { label: "Bài tập 01", v: row.bt01 },
    { label: "Bài tập 02", v: row.bt02 },
    { label: "Bài tập 03", v: row.bt03 },
  ].filter((x) => x.v !== null);
  const qtRaw = (row.avg ?? 0) + row.bonus - row.penalty;
  const rawBonus = row.speaks * SCHEME_1.speakBonus;

  return (
    <div className={s.calc}>
      <div className={s.calcRow}>
        <span className={s.calcLabel}>
          Trung bình bài tập đã công bố
          <span className={s.calcFormula}>
            ({got.map((x) => fmtScore(x.v)).join(" + ")}) ÷ {got.length}
          </span>
        </span>
        <span className={s.calcValue}>{row.avg === null ? "—" : fmtScore(row.avg, 2)}</span>
      </div>
      <div className={s.calcRow}>
        <span className={s.calcLabel}>
          Điểm cộng phát biểu
          <span className={s.calcFormula}>
            {row.speaks} lần × {fmtScore(SCHEME_1.speakBonus, 2)}
            {rawBonus > SCHEME_1.speakCap ? ` = ${fmtScore(rawBonus, 2)}, chạm trần +${fmtScore(SCHEME_1.speakCap, 1)}` : ""}
          </span>
        </span>
        <span className={s.calcValue}>+{fmtScore(row.bonus, 2)}</span>
      </div>
      <div className={s.calcRow}>
        <span className={s.calcLabel}>
          Trừ điểm vắng
          <span className={s.calcFormula}>
            Vắng {row.absences} buổi · trừ {fmtScore(SCHEME_1.absencePenalty, 1)} mỗi buổi từ buổi vắng thứ {SCHEME_1.absenceFrom}
          </span>
        </span>
        <span className={s.calcValue}>−{fmtScore(row.penalty, 2)}</span>
      </div>
      <div className={s.calcRow}>
        <span className={s.calcLabel}>
          Điểm quá trình (làm tròn 0,1)
          <span className={s.calcFormula}>
            {row.avg === null ? "—" : `${fmtScore(row.avg, 2)} + ${fmtScore(row.bonus, 2)} − ${fmtScore(row.penalty, 2)} = ${fmtScore(qtRaw, 2)}`}
          </span>
        </span>
        <span className={s.calcTotal}>{row.qt === null ? "—" : fmtScore(row.qt)}</span>
      </div>
      <div className={s.calcRow}>
        <span className={s.calcLabel}>
          Điểm cuối kỳ
          <span className={s.calcFormula}>Chưa có điểm thi kết thúc học phần (tuần 16)</span>
        </span>
        <span className={s.calcValue}>{row.ck === null ? "—" : fmtScore(row.ck)}</span>
      </div>
      {!published && row.bt03 === null && (
        <InlineNotice tone="info" compact>
          Bài tập 03 chưa công bố điểm nên chưa tính vào trung bình. Công bố ở màn Chấm bài, điểm quá trình sẽ tự tính lại.
        </InlineNotice>
      )}
      <p className={s.note}>{OFFICIAL_GRADE_NOTE}</p>
    </div>
  );
}
