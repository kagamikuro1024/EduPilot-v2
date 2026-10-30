package grade

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/store"
)

// Student là một sinh viên ACTIVE của lớp; Code là MSSV do giảng viên xác lập (`enrollments.student_code_snapshot`), không phải `users.student_code` tự khai (TLR-3).
type Student struct {
	ID       uuid.UUID
	FullName string
	Code     string
}

// ActiveStudents trả sinh viên ACTIVE của lớp, sắp theo tên bỏ dấu rồi id.
func ActiveStudents(ctx context.Context, q *store.Queries, courseID uuid.UUID) ([]Student, error) {
	rows, err := q.GradeActiveStudents(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("grade: sinh viên của lớp: %w", err)
	}
	out := make([]Student, len(rows))
	for i, r := range rows {
		out[i] = Student{ID: r.StudentID, FullName: r.FullName}
		if r.StudentCode != nil {
			out[i].Code = *r.StudentCode
		}
	}
	return out, nil
}

// LoadItems đọc item của một công thức kèm trạng thái nguồn (`Closed`, thang bài thi). Thứ tự theo `position`.
func LoadItems(ctx context.Context, q *store.Queries, courseID, schemeID uuid.UUID) ([]Item, error) {
	rows, err := q.GradeSchemeItemsMeta(ctx, store.GradeSchemeItemsMetaParams{CourseID: courseID, SchemeID: schemeID})
	if err != nil {
		return nil, fmt.Errorf("grade: item của công thức: %w", err)
	}
	out := make([]Item, len(rows))
	for i, r := range rows {
		it := Item{ID: r.ID, ComponentID: r.ComponentID, Key: r.Key, Name: r.Name, Max: r.MaxScore, Source: ItemSource(r.Source), ExamID: r.ExamID, Position: int(r.Position), Published: r.PublishedAt != nil}
		switch it.Source {
		case SourceExam:
			it.Closed = r.ExamPublished
			it.ExamMax = r.ExamMax.Decimal
		case SourceManual:
			it.Closed = it.Published
		case SourceAssignment:
			// P7 nối điểm bài tập; chưa có nguồn đọc → luôn thiếu và chưa chốt.
		}
		out[i] = it
	}
	return out, nil
}

// LoadInputs đọc đầu vào điểm của một lô sinh viên ACTIVE (đúng ba truy vấn cho cả lô, không N+1):
// item MANUAL / IMPORT từ `grade_entries`; item EXAM từ lượt GRADED của bài thi PUBLISHED (COALESCE(adjusted, auto), quy thang item); điểm cộng / trừ / vắng của P5.
// Sinh viên không ACTIVE không có trong kết quả. Không đọc study_time_daily, student_risk, student_observations.
func LoadInputs(ctx context.Context, q *store.Queries, courseID, schemeID uuid.UUID, studentIDs []uuid.UUID) (map[uuid.UUID]Input, error) {
	out := make(map[uuid.UUID]Input, len(studentIDs))
	if len(studentIDs) == 0 {
		return out, nil
	}
	strs := idStrings(studentIDs)
	p5, err := q.GradeInputP5(ctx, store.GradeInputP5Params{CourseID: courseID, StudentIds: strs})
	if err != nil {
		return nil, fmt.Errorf("grade: đầu vào P5: %w", err)
	}
	for _, r := range p5 {
		out[r.StudentID] = Input{Scores: map[uuid.UUID]decimal.Decimal{}, BonusPoints: r.BonusPoints, PenaltyEventPoints: r.PenaltyEventPoints, Absent: int(r.Absent)}
	}
	ent, err := q.GradeInputEntries(ctx, store.GradeInputEntriesParams{CourseID: courseID, SchemeID: schemeID, StudentIds: strs})
	if err != nil {
		return nil, fmt.Errorf("grade: điểm đã nhập: %w", err)
	}
	for _, r := range ent {
		if in, ok := out[r.StudentID]; ok {
			in.Scores[r.ItemID] = r.Score
		}
	}
	ex, err := q.GradeInputExams(ctx, store.GradeInputExamsParams{CourseID: courseID, SchemeID: schemeID, StudentIds: strs})
	if err != nil {
		return nil, fmt.Errorf("grade: điểm bài thi: %w", err)
	}
	for _, r := range ex {
		if in, ok := out[r.StudentID]; ok && r.ExamMax.Sign() > 0 {
			in.Scores[r.ItemID] = r.Score.Mul(r.ItemMax).DivRound(r.ExamMax, DivPrecision)
		}
	}
	return out, nil
}

// idStrings đổi danh sách uuid sang chuỗi: truy vấn ép lại bằng `::text[]::uuid[]` (PgBouncer).
func idStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
