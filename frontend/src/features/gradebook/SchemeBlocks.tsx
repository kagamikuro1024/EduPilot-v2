"use client";

import { FileText, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button, IconButton, Input, Select, StatusText } from "@/shared/ui";
import type { Citation, PutRule, RuleDoc, RuleKind, SchemeDoc, Unclear, Working } from "./gradeApi";
import s from "./GradeScheme.module.css";

export type BlockId = "components" | "BONUS" | "PENALTY" | "ROUNDING" | "LETTER_SCALE" | "FAIL_CONDITION";
export const BLOCKS: Array<{ id: BlockId; title: string; short: string }> = [
  { id: "components", title: "Thành phần", short: "thành phần" },
  { id: "BONUS", title: "Điểm cộng", short: "điểm cộng" },
  { id: "PENALTY", title: "Điểm trừ", short: "điểm trừ" },
  { id: "ROUNDING", title: "Làm tròn", short: "làm tròn" },
  { id: "LETTER_SCALE", title: "Thang chữ", short: "thang chữ" },
  { id: "FAIL_CONDITION", title: "Liệt", short: "liệt" },
];

const vi = (v: unknown) => String(v ?? "").replace(".", ",");
const NUM = /^\d{1,3}([.,]\d{1,2})?$/;
const dot = (t: string) => t.trim().replace(",", ".");

/** Mục chưa rõ của khối: `components…` → Thành phần; `rules.KIND` → khối của luật đó. */
export const unclearOf = (items: Unclear[], id: BlockId) => items.filter((u) => (id === "components" ? u.target.startsWith("components") : u.target === `rules.${id}`));

/** Ô nhập giữ chữ của chính nó: sai ⇒ lỗi tại ô và GIỮ chữ đã gõ; đúng ⇒ rời ô là lưu. */
function Cell({ label, value, onCommit, check, disabled, width = 96, suffix, placeholder, plain }: { label: string; value: string; onCommit: (v: string) => void; check?: (t: string) => string | null; disabled?: boolean; width?: number; suffix?: string; placeholder?: string; /** chữ, không phải số: không đổi dấu chấm thành dấu phẩy */ plain?: boolean }) {
  const [text, setText] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const view = plain ? value : vi(value);
  const shown = text ?? view;
  return (
    <label className={s.cell}>
      <span className={s.cellLabel}>{label}</span>
      <span className={s.cellRow}>
        <Input
          style={{ width }}
          value={shown}
          placeholder={placeholder}
          disabled={disabled}
          inputMode="decimal"
          aria-invalid={err ? true : undefined}
          onChange={(e) => {
            setText(e.target.value);
            setErr(null);
          }}
          onBlur={() => {
            if (text === null || text === view) {
              setText(null);
              return;
            }
            const bad = check?.(text) ?? null;
            if (bad) return setErr(bad);
            onCommit(plain ? text.trim() : dot(text));
            setText(null);
          }}
        />
        {suffix && <span className={s.cellSuffix}>{suffix}</span>}
      </span>
      {err && <span className={s.cellError} role="alert">{err}</span>}
    </label>
  );
}

const weightCheck = (t: string) => (NUM.test(t.trim()) && Number(dot(t)) <= 100 ? null : "Nhập số từ 0 đến 100");
const scoreCheck = (t: string) => (NUM.test(t.trim()) && Number(dot(t)) <= 10 ? null : "Nhập số từ 0 đến 10");
const decCheck = (t: string) => (NUM.test(t.trim()) ? null : "Nhập một số");
const intCheck = (t: string) => (/^\d{1,3}$/.test(t.trim()) ? null : "Nhập số nguyên");

type Params = Record<string, unknown>;
const str = (p: Params | undefined, k: string) => (p && typeof p[k] === "string" ? (p[k] as string) : "");
const rulesOf = (w: Working, kind: RuleKind) => w.rules.filter((r) => r.kind === kind);

/** Chữ đọc được của một luật (TA / bản đã xác nhận). */
export function summary(kind: BlockId, r: RuleDoc | PutRule | undefined): string {
  if (!r) return "Chưa có";
  if (!r.applicable && kind !== "FAIL_CONDITION") return "Không áp dụng";
  const p = r.params as Params;
  switch (kind) {
    case "BONUS":
      return `+${vi(str(p, "per_speak"))} mỗi lần phát biểu · tối đa +${vi(str(p, "cap"))}`;
    case "PENALTY":
      return `−${vi(str(p, "per"))} mỗi buổi vắng · miễn ${vi(p.free)} buổi`;
    case "ROUNDING":
      return `Làm tròn đến ${vi(str(p, "step"))}`;
    case "LETTER_SCALE":
      return ((p.bands as Array<{ min: string; letter: string }> | undefined) ?? []).map((b) => `${b.letter} từ ${vi(b.min)}`).join(" · ");
    case "FAIL_CONDITION":
      return str(p, "kind") === "FINAL_BELOW" ? `Thi cuối kỳ dưới ${vi(str(p, "value"))}` : `Vắng từ ${vi(str(p, "value"))} buổi`;
    default:
      return "";
  }
}

const quoteOf = (cs: Citation[] | undefined) => cs?.[0];

export function Blocks({ doc, working, editable, errors, selected, onSelect, onChange }: { doc: SchemeDoc; working: Working; editable: boolean; errors: Record<string, string>; selected: BlockId | null; onSelect: (id: BlockId, c: Citation | null) => void; onChange: (w: Working) => void }) {
  // Mặc định mọi mục là CHỮ (văn bản để duyệt); ô nhập chỉ hiện ở mục chưa rõ hoặc ở mục giảng viên bấm vào để sửa (DESIGN §14.11).
  const [editing, setEditing] = useState<string | null>(null);
  const setRule = (kind: RuleKind, patch: (r: PutRule | undefined) => PutRule | null) => {
    const cur = rulesOf(working, kind)[0];
    const next = patch(cur);
    const rest = working.rules.filter((r) => r !== cur);
    onChange({ ...working, rules: next ? [...rest, next] : rest });
  };
  const setComp = (i: number, patch: Partial<Working["components"][number]>) => onChange({ ...working, components: working.components.map((c, j) => (j === i ? { ...c, ...patch } : c)) });
  const sum = working.components.reduce((a, c) => a + (Number(c.weight) || 0), 0);
  const docRule = (k: RuleKind) => doc.rules.find((r) => r.kind === k);
  const done = (id: string) => (
    <Button size="sm" variant="ghost" onClick={() => setEditing(editing === id ? null : id)}>Xong</Button>
  );

  return (
    <div className={s.blocks}>
      {BLOCKS.map((b) => {
        const un = unclearOf(doc.unclear_items, b.id);
        const rules = b.id === "components" ? [] : doc.rules.filter((r) => r.kind === b.id);
        const cite = b.id === "components" ? quoteOf(doc.components.flatMap((c) => c.citations)) : quoteOf(rules.flatMap((r) => r.citations));
        const byUser = b.id === "components" ? doc.components.some((c) => c.filled_by_user) : rules.some((r) => r.filled_by_user);
        const isSel = selected === b.id;
        const open = editable && (un.length > 0 || editing === b.id); // ô nhập của khối
        const rule = rulesOf(working, b.id as RuleKind)[0];
        const view = (text: string) =>
          editable ? (
            <button type="button" className={s.valueBtn} data-part="edit-value" onClick={() => setEditing(b.id)}>{text}</button>
          ) : (
            <p className={s.value}>{text}</p>
          );
        return (
          <section key={b.id} className={[s.block, isSel ? s.blockSel : ""].join(" ")} aria-labelledby={`blk-${b.id}`} data-block={b.id} data-unclear={un.length > 0 ? "true" : undefined}>
            <div className={s.blockHead}>
              <h3 id={`blk-${b.id}`} className="ep-item-title">{b.title}</h3>
              {b.id === "components" && <StatusText tone={Math.abs(sum - 100) < 0.005 ? "green" : "amber"}>{`Tổng ${vi(Math.round(sum * 100) / 100)}%`}</StatusText>}
              {cite ? (
                <button type="button" className={s.cite} onClick={() => onSelect(b.id, cite)} aria-pressed={isSel}>
                  <FileText aria-hidden />
                  {`Trang ${cite.page}`}
                </button>
              ) : byUser ? (
                <StatusText tone="neutral">Do giảng viên đặt</StatusText>
              ) : null}
            </div>
            {un.map((u, i) => (
              <p key={i} className={s.unclear} role="status">{u.message}</p>
            ))}

            {b.id === "components" && (
              <div className={s.comps}>
                {working.components.map((c, i) => {
                  const text = `${c.name} · ${vi(c.weight)}% · ${c.grp === "QT" ? "Quá trình" : "Cuối kỳ"}`;
                  const rowOpen = editable && (un.length > 0 || editing === `c:${c.key}`);
                  return rowOpen ? (
                    <div key={c.key} className={s.compRow}>
                      <Cell plain label="Tên" value={c.name} width={200} onCommit={(v) => setComp(i, { name: v })} check={(t) => (t.trim() && t.trim().length <= 80 ? null : "Nhập tên 1–80 ký tự")} />
                      <Cell label="Trọng số" value={c.weight} suffix="%" onCommit={(v) => setComp(i, { weight: v })} check={weightCheck} />
                      <label className={s.cell}>
                        <span className={s.cellLabel}>Nhóm</span>
                        <Select value={c.grp} onChange={(e) => setComp(i, { grp: e.target.value as "QT" | "CK" })}>
                          <option value="QT">Quá trình</option>
                          <option value="CK">Cuối kỳ</option>
                        </Select>
                      </label>
                      <IconButton label={`Xoá ${c.name}`} onClick={() => onChange({ ...working, components: working.components.filter((_, j) => j !== i) })}>
                        <Trash2 aria-hidden />
                      </IconButton>
                      {editing === `c:${c.key}` && done(`c:${c.key}`)}
                    </div>
                  ) : editable ? (
                    <button key={c.key} type="button" className={s.valueBtn} data-part="edit-value" onClick={() => setEditing(`c:${c.key}`)}>{text}</button>
                  ) : (
                    <p key={c.key} className={s.value}>{text}</p>
                  );
                })}
                {errors.components && <p className={s.cellError} role="alert">{errors.components}</p>}
                {editable && (
                  <Button
                    size="sm"
                    variant="ghost"
                    icon={<Plus aria-hidden />}
                    onClick={() => {
                      const key = `tp${working.components.length + 1}`;
                      onChange({ ...working, components: [...working.components, { key, name: "Thành phần mới", grp: "QT", weight: "0", items: [] }] });
                      setEditing(`c:${key}`);
                    }}
                  >
                    Thêm thành phần
                  </Button>
                )}
              </div>
            )}

            {(b.id === "BONUS" || b.id === "PENALTY") &&
              (!open || !rule ? (
                !rule && !editable ? <p className={s.value}>Chưa có</p> : !rule ? <RuleApplicable kind={b.id} rule={rule} editable onSet={(applicable) => setRule(b.id as RuleKind, (r) => ({ kind: b.id as RuleKind, applicable, params: applicable ? (r?.params ?? {}) : {} }))}>{null}</RuleApplicable> : view(summary(b.id, rule))
              ) : (
                <>
                  <RuleApplicable kind={b.id} rule={rule} editable onSet={(applicable) => setRule(b.id as RuleKind, (r) => ({ kind: b.id as RuleKind, applicable, params: applicable ? (r?.params ?? {}) : {} }))}>
                    {b.id === "BONUS" ? (
                      <>
                        <Cell label="Mỗi lần phát biểu" value={str(rule?.params, "per_speak")} check={decCheck} onCommit={(v) => setRule("BONUS", (r) => ({ kind: "BONUS", applicable: true, params: { source: "PARTICIPATION", per_speak: v, cap: str(r?.params, "cap") || undefined } }))} />
                        <Cell label="Tối đa" value={str(rule?.params, "cap")} check={decCheck} onCommit={(v) => setRule("BONUS", (r) => ({ kind: "BONUS", applicable: true, params: { source: "PARTICIPATION", per_speak: str(r?.params, "per_speak") || undefined, cap: v } }))} />
                      </>
                    ) : (
                      <>
                        <Cell label="Buổi vắng được miễn" value={String((rule?.params as Params | undefined)?.free ?? "")} check={intCheck} onCommit={(v) => setRule("PENALTY", (r) => ({ kind: "PENALTY", applicable: true, params: { source: "ABSENCE", free: Number(v), per: str(r?.params, "per") || undefined } }))} />
                        <Cell label="Trừ mỗi buổi" value={str(rule?.params, "per")} check={decCheck} onCommit={(v) => setRule("PENALTY", (r) => ({ kind: "PENALTY", applicable: true, params: { source: "ABSENCE", free: (r?.params as Params | undefined)?.free, per: v } }))} />
                      </>
                    )}
                  </RuleApplicable>
                  {editing === b.id && done(b.id)}
                </>
              ))}

            {b.id === "ROUNDING" &&
              (open || (editable && !rule) ? (
                <div className={s.compRow}>
                  <label className={s.cell}>
                    <span className={s.cellLabel}>Bước làm tròn</span>
                    <Select
                      value={str(rule?.params, "step")}
                      aria-invalid={un.length > 0 ? true : undefined}
                      onChange={(e) => setRule("ROUNDING", () => (e.target.value ? { kind: "ROUNDING", applicable: true, params: { step: e.target.value, mode: "HALF_UP" } } : null))}
                    >
                      <option value="">Chọn bước</option>
                      {["0.01", "0.1", "0.25", "0.5", "1"].map((v) => (
                        <option key={v} value={v}>{`Làm tròn đến ${vi(v)}`}</option>
                      ))}
                    </Select>
                  </label>
                  {editing === b.id && done(b.id)}
                </div>
              ) : (
                view(summary("ROUNDING", rule ?? docRule("ROUNDING")))
              ))}

            {b.id === "LETTER_SCALE" && (
              open || (editable && !rule) ? (
                <>
                  <Letters rule={rule} shown={docRule("LETTER_SCALE")} editable onSet={(bands) => setRule("LETTER_SCALE", () => ({ kind: "LETTER_SCALE", applicable: true, params: { bands } }))} />
                  {editing === b.id && done(b.id)}
                </>
              ) : (
                view(summary("LETTER_SCALE", rule ?? docRule("LETTER_SCALE")))
              )
            )}

            {b.id === "FAIL_CONDITION" && (
              <div className={s.comps}>
                {rulesOf(working, "FAIL_CONDITION").map((r, i) => {
                  const k = `f:${i}`;
                  return editable && (un.length > 0 || editing === k) ? (
                    <div key={i} className={s.compRow}>
                      <label className={s.cell}>
                        <span className={s.cellLabel}>Điều kiện</span>
                        <Select value={str(r.params, "kind")} onChange={(e) => onChange({ ...working, rules: working.rules.map((x) => (x === r ? { ...x, params: { ...x.params, kind: e.target.value } } : x)) })}>
                          <option value="FINAL_BELOW">Thi cuối kỳ dưới</option>
                          <option value="ABSENT_AT_LEAST">Số buổi vắng từ</option>
                        </Select>
                      </label>
                      <Cell label="Giá trị" value={str(r.params, "value")} check={decCheck} onCommit={(v) => onChange({ ...working, rules: working.rules.map((x) => (x === r ? { ...x, params: { ...x.params, value: v } } : x)) })} />
                      <IconButton label="Xoá điều kiện" onClick={() => onChange({ ...working, rules: working.rules.filter((x) => x !== r) })}>
                        <Trash2 aria-hidden />
                      </IconButton>
                      {editing === k && done(k)}
                    </div>
                  ) : editable ? (
                    <button key={i} type="button" className={s.valueBtn} data-part="edit-value" onClick={() => setEditing(k)}>{summary("FAIL_CONDITION", r)}</button>
                  ) : (
                    <p key={i} className={s.value}>{summary("FAIL_CONDITION", r)}</p>
                  );
                })}
                {rulesOf(working, "FAIL_CONDITION").length === 0 && <p className={s.value}>Không có</p>}
                {editable && (
                  <Button size="sm" variant="ghost" icon={<Plus aria-hidden />} onClick={() => { onChange({ ...working, rules: [...working.rules, { kind: "FAIL_CONDITION", applicable: true, params: { kind: "FINAL_BELOW", value: "3" } }] }); setEditing(`f:${rulesOf(working, "FAIL_CONDITION").length}`); }}>
                    Thêm điều kiện
                  </Button>
                )}
              </div>
            )}

            {cite && (
              <blockquote className={s.quote}>
                <button type="button" className={s.quoteBtn} onClick={() => onSelect(b.id, cite)}>{cite.quote}</button>
              </blockquote>
            )}
          </section>
        );
      })}
    </div>
  );
}

/** Luật có thể "Không áp dụng": lựa chọn ngay tại khối; chưa chọn gì ⇒ trống, chờ giảng viên. */
function RuleApplicable({ kind, rule, editable, onSet, children }: { kind: BlockId; rule: PutRule | undefined; editable: boolean; onSet: (applicable: boolean) => void; children: React.ReactNode }) {
  if (!editable) return <p className={s.value}>{summary(kind, rule as PutRule | undefined)}</p>;
  const mode = rule ? (rule.applicable ? "yes" : "no") : "";
  return (
    <div className={s.compRow}>
      <label className={s.cell}>
        <span className={s.cellLabel}>Áp dụng</span>
        <Select value={mode} onChange={(e) => e.target.value && onSet(e.target.value === "yes")}>
          <option value="">Chọn</option>
          <option value="yes">Có</option>
          <option value="no">Không áp dụng</option>
        </Select>
      </label>
      {mode === "yes" && children}
    </div>
  );
}

function Letters({ rule, shown, editable, onSet }: { rule: PutRule | undefined; shown: RuleDoc | undefined; editable: boolean; onSet: (bands: Array<{ min: string; letter: string }>) => void }) {
  const bands = ((rule?.params as Params | undefined)?.bands as Array<{ min: string; letter: string }> | undefined) ?? [];
  if (!editable) return <p className={s.value}>{summary("LETTER_SCALE", shown)}</p>;
  const set = (i: number, patch: Partial<{ min: string; letter: string }>) => onSet(bands.map((b, j) => (j === i ? { ...b, ...patch } : b)));
  return (
    <div className={s.comps}>
      {bands.map((b, i) => (
        <div key={i} className={s.compRow}>
          <Cell plain label="Điểm chữ" value={b.letter} width={64} onCommit={(v) => set(i, { letter: v })} check={(t) => (/^[A-Za-z][A-Za-z+]{0,3}$/.test(t.trim()) ? null : "1–4 ký tự")} />
          <Cell label="Từ điểm" value={b.min} check={scoreCheck} onCommit={(v) => set(i, { min: v })} />
          <IconButton label={`Xoá mức ${b.letter}`} onClick={() => onSet(bands.filter((_, j) => j !== i))}>
            <Trash2 aria-hidden />
          </IconButton>
        </div>
      ))}
      <Button size="sm" variant="ghost" icon={<Plus aria-hidden />} onClick={() => onSet([...bands, { min: "0", letter: "F" }])}>
        Thêm mức
      </Button>
    </div>
  );
}
