package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/store"
)

// Loại nội dung chia sẻ được ở P2. `questions` (P9) và `grade_scheme` (P6) chưa có bảng ⇒ NOT_AVAILABLE kèm tên phase sở hữu.
// ponytail: switch thay cho bộ đăng ký Sharer — thêm một case khi P6 / P9 có bảng; chưa cần cơ chế đăng ký cho hai ca.
const (
	ShareDocuments   = "documents"
	ShareQuestions   = "questions"
	ShareGradeScheme = "grade_scheme"
)

// ShareSourceKindError: loại nội dung không hợp lệ / chưa có (422 VALIDATION_FAILED, details.code NOT_AVAILABLE hoặc INVALID_KIND).
func shareKindError(kind string) error {
	switch kind {
	case ShareQuestions:
		return &InvalidError{Field: "what", Code: "NOT_AVAILABLE", Message: "Ngân hàng câu hỏi chưa dùng lại được (phase P9)."}
	case ShareGradeScheme:
		return &InvalidError{Field: "what", Code: "NOT_AVAILABLE", Message: "Công thức điểm chưa dùng lại được (phase P6)."}
	}
	return &InvalidError{Field: "what", Code: "INVALID_KIND", Message: "Loại nội dung không hợp lệ."}
}

// ErrSourceCourse: người gọi không phải giảng viên của lớp nguồn (hoặc lớp nguồn không có) — cùng một câu trả lời, không lộ lớp có hay không.
var ErrSourceCourse = errors.New("course: không phải giảng viên của lớp nguồn")

// ErrSubjectMismatch: hai lớp khác học phần.
func errSubjectMismatch() error {
	return &InvalidError{Field: "source_course_id", Code: "SUBJECT_MISMATCH", Message: "Hai lớp phải cùng học phần."}
}

// Skipped là phần không chia sẻ và lý do (để giảng viên hiểu vì sao số tài liệu ít hơn tổng).
type Skipped struct{ Kind, Reason string }

// ShareResult: Documents = số tài liệu của lớp nguồn hiện dùng được ở lớp đích (ổn định khi gọi lại).
type ShareResult struct {
	Documents int
	Skipped   []Skipped
}

// ShareFrom chia sẻ nội dung từ lớp nguồn sang lớp đích (targetID đã qua guard Teacher). Một chiều: đích không đọc / sửa nguồn.
func (s Service) ShareFrom(ctx context.Context, targetID, teacherID, sourceID uuid.UUID, what []string) (ShareResult, error) {
	if len(what) == 0 {
		return ShareResult{}, &InvalidError{Field: "what", Code: "REQUIRED", Message: "Chọn ít nhất một loại nội dung."}
	}
	for _, k := range what { // kiểm hết loại TRƯỚC khi ghi bất cứ gì
		if k != ShareDocuments {
			return ShareResult{}, shareKindError(k)
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return ShareResult{}, fmt.Errorf("course: mở transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	// Khoá hai lớp theo thứ tự id tăng: hai lần chia sẻ ngược chiều không bế tắc.
	ids := []uuid.UUID{targetID, sourceID}
	if sourceID.String() < targetID.String() {
		ids[0], ids[1] = sourceID, targetID
	}
	cs := map[uuid.UUID]store.Course{}
	for _, id := range ids {
		c, err := q.LockCourse(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			if id == sourceID {
				return ShareResult{}, ErrSourceCourse
			}
			return ShareResult{}, ErrNotFound
		}
		if err != nil {
			return ShareResult{}, fmt.Errorf("course: khoá lớp: %w", err)
		}
		cs[id] = c
	}
	if sourceID == targetID {
		return ShareResult{}, &InvalidError{Field: "source_course_id", Code: "SAME_COURSE", Message: "Hãy chọn một lớp khác."}
	}
	// Quyền trên lớp nguồn: giảng viên ĐANG HOẠT ĐỘNG. ADMIN không là giảng viên của lớp nào ⇒ không tới được đây.
	m, err := q.GetMembership(ctx, store.GetMembershipParams{CourseID: sourceID, UserID: teacherID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (m.RoleInCourse != "TEACHER" || m.Status != store.EnrollmentStatusACTIVE)) {
		return ShareResult{}, ErrSourceCourse
	}
	if err != nil {
		return ShareResult{}, fmt.Errorf("course: tra lớp nguồn: %w", err)
	}
	src, dst := cs[sourceID], cs[targetID]
	if src.SubjectCode != dst.SubjectCode {
		return ShareResult{}, errSubjectMismatch()
	}
	if src.Status != store.CourseStatusACTIVE || dst.Status != store.CourseStatusACTIVE {
		return ShareResult{}, ErrArchived
	}

	if _, err := q.ShareCourseDocuments(ctx, store.ShareCourseDocumentsParams{TargetID: targetID, SharedBy: &teacherID, SourceID: sourceID}); err != nil {
		return ShareResult{}, fmt.Errorf("course: chia sẻ tài liệu: %w", err)
	}
	if err := q.AddTargetToSharedChunks(ctx, store.AddTargetToSharedChunksParams{TargetID: targetID, SourceID: sourceID}); err != nil {
		return ShareResult{}, fmt.Errorf("course: thêm lớp vào chunk: %w", err)
	}
	n, err := q.CountSharedDocuments(ctx, store.CountSharedDocumentsParams{TargetID: targetID, SourceID: sourceID})
	if err != nil {
		return ShareResult{}, fmt.Errorf("course: đếm tài liệu: %w", err)
	}
	res := ShareResult{Documents: int(n.Shared), Skipped: []Skipped{}}
	if n.Policy > 0 {
		res.Skipped = append(res.Skipped, Skipped{Kind: ShareDocuments, Reason: "Quy chế môn học không dùng lại được: mỗi lớp có quy chế riêng."})
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &targetID, ActorID: &teacherID, Entity: "course", EntityID: targetID.String(), Action: "content_shared",
		After: []byte(fmt.Sprintf(`{"source":%q,"documents":%d}`, sourceID, res.Documents))}); err != nil {
		return ShareResult{}, fmt.Errorf("course: ghi audit_log: %w", err)
	}
	if err := emitChanged(ctx, tx, targetID); err != nil {
		return ShareResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ShareResult{}, fmt.Errorf("course: commit chia sẻ: %w", err)
	}
	return res, nil
}

// ShareSource là một lớp có thể lấy nội dung về.
type ShareSource struct {
	ID                        uuid.UUID
	ClassCode, Name, Semester string
	Documents                 int
}

// ShareSources liệt kê lớp cùng học phần mà giảng viên là TEACHER ACTIVE của cả hai, chưa lưu trữ.
func (s Service) ShareSources(ctx context.Context, targetID, teacherID uuid.UUID) ([]ShareSource, error) {
	rows, err := store.New(s.Pool).ListShareSources(ctx, store.ListShareSourcesParams{UserID: teacherID, TargetID: targetID})
	if err != nil {
		return nil, fmt.Errorf("course: nguồn chia sẻ: %w", err)
	}
	out := make([]ShareSource, len(rows))
	for i, r := range rows {
		out[i] = ShareSource{ID: r.ID, ClassCode: r.ClassCode, Name: r.Name, Semester: r.Semester, Documents: int(r.Documents)}
	}
	return out, nil
}
