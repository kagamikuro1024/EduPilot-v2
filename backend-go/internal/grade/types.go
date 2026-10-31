// Package grade: Grade Engine (hàm thuần, decimal), nguồn điểm, công thức, sổ điểm, chốt / mở khoá, nhập / xuất XLSX, điểm của sinh viên
// (FEAT-gradebook). Luật 5: mọi phép tính điểm là Go + shopspring/decimal, không float, không LLM, không thời gian học (D17).
// Gói KHÔNG import activity, crm, llm, grading, privacy, agent (kiểm bằng TestForbiddenImports).
package grade

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MaxScale là thang điểm tối đa của mọi điểm thành phần / học phần (10) và TotalWeight là tổng trọng số (100): hai hằng DUY NHẤT của công thức PRD M7.
// Mọi giá trị mang nghĩa quy chế (trọng số, trần, bước, ngưỡng) đến từ `grade_rules.params` / `grade_components.weight`.
var (
	MaxScale    = decimal.NewFromInt(10)  //nolint:gochecknoglobals // hằng decimal bất biến
	TotalWeight = decimal.NewFromInt(100) //nolint:gochecknoglobals // hằng decimal bất biến
)

// DivPrecision: mọi phép chia dùng DivRound(·, 20) (TLR-1); NormalizeDigits: HP thô được chuẩn hoá Round(10) ngay trước bước làm tròn nghiệp vụ.
const (
	DivPrecision    = 20
	NormalizeDigits = 10
)

// Group là nhóm điểm: QT (quá trình) hay CK (cuối kỳ).
type Group string

const (
	GroupQT Group = "QT"
	GroupCK Group = "CK"
)

// ItemSource là nguồn điểm của một item.
type ItemSource string

const (
	SourceManual     ItemSource = "MANUAL"
	SourceExam       ItemSource = "EXAM"
	SourceAssignment ItemSource = "ASSIGNMENT"
)

// Citation là một đoạn trích quy chế (nguyên văn) kèm số trang.
type Citation struct {
	Page  int    `json:"page"`
	Quote string `json:"quote"`
}

// Item là một cột điểm của thành phần. Max là thang của item (điểm nhập ∈ [0; Max]).
// Closed: nguồn đã chốt sổ cho item (bài thi PUBLISHED; cột MANUAL/IMPORT có published_at) — chỉ khi đó "thiếu điểm" mới được coi là 0 theo MISSING_SCORE (TLR-2).
type Item struct {
	ID          uuid.UUID
	Key, Name   string
	Max         decimal.Decimal
	Source      ItemSource
	ExamID      *uuid.UUID
	Closed      bool
	Published   bool // published_at IS NOT NULL (cột nhập tay)
	ExamMax     decimal.Decimal
	Position    int
	ComponentID uuid.UUID
}

// Component là một thành phần điểm: trọng số là % của điểm học phần.
type Component struct {
	ID           uuid.UUID
	Key, Name    string
	Group        Group
	Weight       decimal.Decimal
	Items        []Item
	Citations    []Citation
	FilledByUser bool
}

// RuleMeta là căn cứ quy chế của một luật.
type RuleMeta struct {
	Quote        string
	Page         int
	FilledByUser bool
}

// BonusRule: điểm cộng = Σ điểm SPEAK + BONUS (P5) tối đa Cap. PerSpeak chỉ để hiển thị / đối chiếu (Q6).
type BonusRule struct {
	RuleMeta
	Applicable bool
	PerSpeak   decimal.Decimal
	Cap        decimal.Decimal
}

// PenaltyRule: trừ Per mỗi buổi ABSENT vượt Free, cộng điểm sự kiện PENALTY.
type PenaltyRule struct {
	RuleMeta
	Applicable bool
	Free       int
	Per        decimal.Decimal
}

// RoundingRule: bước làm tròn điểm học phần (nửa lên).
type RoundingRule struct {
	RuleMeta
	Step decimal.Decimal
}

// Band là một mức của thang chữ: điểm học phần đã làm tròn ≥ Min → Letter.
type Band struct {
	Min    decimal.Decimal
	Letter string
}

// LetterScale: các mức giảm dần theo Min; mức cuối có Min = 0.
type LetterScale struct {
	RuleMeta
	Bands []Band
}

// FailKind là loại điều kiện liệt.
type FailKind string

const (
	FailFinalBelow    FailKind = "FINAL_BELOW"
	FailAbsentAtLeast FailKind = "ABSENT_AT_LEAST"
)

// FailCondition: sinh viên thoả điều kiện bị điểm chữ F (HP vẫn giữ số đã tính).
type FailCondition struct {
	RuleMeta
	Kind  FailKind
	Value decimal.Decimal
	Text  string // chữ gốc của value ("3.0"), in lại ở verdict
}

// MissingScoreRule: các thành phần (theo key) được tính 0 khi thiếu điểm mà nguồn đã chốt (TLR-2).
type MissingScoreRule struct {
	RuleMeta
	ZeroComponents []string
}

// Rules gom mọi luật của một công thức. Con trỏ nil = công thức không có luật đó.
type Rules struct {
	Bonus        *BonusRule
	Penalty      *PenaltyRule
	Rounding     *RoundingRule
	Letters      *LetterScale
	Fails        []FailCondition
	MissingScore *MissingScoreRule
}

// Scheme là một công thức điểm (bản nháp hay đã xác nhận) ở dạng engine đọc.
type Scheme struct {
	ID         uuid.UUID
	No         int
	Version    int
	Components []Component
	Rules      Rules
}

// Input là đầu vào điểm của MỘT sinh viên. Scores: item → điểm theo thang của item; thiếu khoá = thiếu điểm (không phải 0).
type Input struct {
	Scores map[uuid.UUID]decimal.Decimal
	// ItemClosed / ItemReason: trạng thái THEO SINH VIÊN của item ASSIGNMENT thiếu điểm (SRS FEAT-assignments 4.9 mục 3): đã đóng (không nộp → 0 nếu thành phần trong zero_components)
	// hay còn chờ (lý do: còn bài chưa khớp / chưa chấm xong / chưa đến hạn). Item không ở đây theo `Item.Closed` như cũ.
	ItemClosed         map[uuid.UUID]bool
	ItemReason         map[uuid.UUID]string
	BonusPoints        decimal.Decimal
	PenaltyEventPoints decimal.Decimal
	Absent             int
}
