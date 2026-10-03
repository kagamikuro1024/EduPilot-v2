// Ma trận 24 khối × 8 trạng thái (docs/specs/FEAT-ui-foundation/SRS.md 7.2): 117 ô áp dụng + 75 ô N/A có lý do.
// /dev/ui vẽ đúng theo bảng này; thêm / bớt ô phải qua proposals.md.

export type UiState = "default" | "hover" | "focus" | "selected" | "disabled" | "loading" | "empty" | "error";
export const STATES: readonly UiState[] = ["default", "hover", "focus", "selected", "disabled", "loading", "empty", "error"];

/** Mã lý do N/A (SRS 7.2). */
export const NA_REASONS = {
  A: "Thành phần không có khái niệm “được chọn”.",
  B: "Thành phần không có khái niệm “rỗng” riêng; nơi dùng quyết định.",
  C: "Lỗi do nơi dùng báo qua InlineNotice / Field.",
  D: "Thành phần không có chờ tải riêng.",
  E: "Không có vùng rê chuột, hoặc rê chuột thuộc phần tử con đã được kiểm.",
  F: "Khoá được ở mức phần tử con (nút bên trong tự có trạng thái vô hiệu).",
  G: "Thành phần không tương tác.",
  H: "Biến thể ngữ nghĩa nằm trong trạng thái mặc định (bốn lời của VerificationState).",
} as const;
export type NaCode = keyof typeof NA_REASONS;

export type BlockSpec = {
  name: string;
  /** các trạng thái được vẽ */
  states: UiState[];
  /** các trạng thái N/A → mã lý do */
  na: Partial<Record<UiState, NaCode>>;
};

const D = "default", H = "hover", F = "focus", S = "selected", X = "disabled", L = "loading", E = "empty", R = "error";

export const REGISTRY: BlockSpec[] = [
  { name: "Button", states: [D, H, F, X, L], na: { selected: "A", empty: "B", error: "C" } },
  { name: "Field", states: [D, H, F, X, E, R], na: { selected: "A", loading: "D" } },
  { name: "Checkbox", states: [D, H, F, S, X, R], na: { loading: "D", empty: "B" } },
  { name: "Switch", states: [D, H, F, S, X, L], na: { empty: "B", error: "C" } },
  { name: "Tabs", states: [D, H, F, S, X], na: { loading: "D", empty: "B", error: "C" } },
  { name: "SegmentedControl", states: [D, H, F, S, X], na: { loading: "D", empty: "B", error: "C" } },
  { name: "FilterChips", states: [D, H, F, S, X], na: { loading: "D", empty: "B", error: "C" } },
  { name: "ActionList", states: [D, H, F, S, X, L, E, R], na: {} },
  { name: "DataTable", states: [D, H, F, S, X, L, E, R], na: {} },
  { name: "Popover", states: [D, F, X, L, E], na: { hover: "E", selected: "A", error: "C" } },
  { name: "Menu", states: [D, H, F, S, X], na: { loading: "D", empty: "B", error: "C" } },
  { name: "Dialog", states: [D, F, L, R], na: { hover: "E", selected: "A", disabled: "F", empty: "B" } },
  { name: "Drawer", states: [D, F, L, R], na: { hover: "E", selected: "A", disabled: "F", empty: "B" } },
  { name: "ConfirmIrreversible", states: [D, F, X, L, R], na: { hover: "E", selected: "A", empty: "B" } },
  { name: "Composer", states: [D, H, F, X, L, E, R], na: { selected: "A" } },
  { name: "InlineNotice", states: [D, F, R], na: { hover: "E", selected: "A", disabled: "F", loading: "D", empty: "B" } },
  { name: "StatusText", states: [D, R], na: { hover: "G", focus: "G", selected: "G", disabled: "G", loading: "G", empty: "G" } },
  { name: "EmptyState", states: [D, F], na: { hover: "G", selected: "G", disabled: "G", loading: "G", empty: "G", error: "G" } },
  { name: "Skeleton", states: [D, L], na: { hover: "G", focus: "G", selected: "G", disabled: "G", empty: "G", error: "G" } },
  { name: "UndoLine", states: [D, H, F, R], na: { selected: "A", disabled: "F", loading: "D", empty: "B" } },
  { name: "Layout", states: [D, L, R], na: { hover: "E", focus: "E", selected: "A", disabled: "F", empty: "B" } },
  { name: "CommandPalette", states: [D, H, F, S, L, E], na: { disabled: "F", error: "C" } },
  { name: "CitationList", states: [D, H, F, S, L, E, R], na: { disabled: "F" } },
  { name: "VerificationState", states: [D, H, F, X], na: { selected: "H", loading: "C", empty: "C", error: "C" } },
] as BlockSpec[];

export const APPLICABLE = REGISTRY.reduce((n, b) => n + b.states.length, 0); // 117
export const NOT_APPLICABLE = REGISTRY.reduce((n, b) => n + Object.keys(b.na).length, 0); // 75
