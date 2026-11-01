package submission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// TopicResolved: bài chưa khớp vừa được gán tay / bỏ qua / hoàn tác → xoá cache "Hôm nay" của nhân viên (mục UNMATCHED_SUBMISSION) và của sinh viên được gán.
const TopicResolved = "submission.resolved"

// Staff là người gọi route của nhân viên.
type Staff struct {
	UserID  uuid.UUID
	IsTA    bool // TA không gán / bỏ qua được bài đã có điểm công bố (Q5)
	TraceID string
}

// ReasonText là lời giải thích bằng chữ của lý do không khớp (không dùng từ kỹ thuật).
func ReasonText(reason string) string {
	switch reason {
	case ReasonNoAssignmentCode:
		return "Không có mã bài tập trong tiêu đề hoặc tên tệp"
	case ReasonAmbiguousAssignment:
		return "Có nhiều mã bài tập"
	case ReasonNoStudentSignal:
		return "Không có MSSV trong tên tệp"
	case ReasonEmailCodeConflict:
		return "Email và MSSV chỉ về hai sinh viên khác nhau"
	case ReasonAmbiguousCode:
		return "MSSV trùng với nhiều sinh viên"
	case ReasonStudentNotInCourse:
		return "MSSV không thuộc lớp"
	case ReasonCodeNotVerified:
		return "MSSV do sinh viên tự khai, chưa có trong danh sách lớp"
	case ReasonSenderUnverified:
		return "Thư chưa được xác thực người gửi"
	case ReasonAlreadyReviewed:
		return "Sinh viên đã có bài được duyệt"
	}
	return ""
}

// Unmatched là một dòng của tab `Chưa khớp`. `Version` là khoá lạc quan của dòng (updated_at theo micro-giây).
type Unmatched struct {
	ID           uuid.UUID  `json:"id"`
	FileName     string     `json:"file_name"`
	Source       string     `json:"source"`
	ReceivedAt   time.Time  `json:"received_at"`
	Reason       string     `json:"reason"`
	ReasonText   string     `json:"reason_text"`
	AssignmentID *uuid.UUID `json:"assignment_id"`
	Suggestion   *Suggested `json:"suggestion"`
	Version      int64      `json:"version"`
}

// Suggested là gợi ý sinh viên cho giảng viên.
type Suggested struct {
	StudentID uuid.UUID `json:"student_id"`
	FullName  string    `json:"full_name"`
}

// UnmatchedPage là một trang tab `Chưa khớp`.
type UnmatchedPage struct {
	Items      []Unmatched `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}

// ListUnmatched: GET …/submissions?match_status=UNMATCHED (TA, TEACHER), phân trang con trỏ.
func (s *Service) ListUnmatched(ctx context.Context, course uuid.UUID, p httpx.PageParams) (UnmatchedPage, error) {
	arg := store.SubmissionUnmatchedListParams{CourseID: course, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).SubmissionUnmatchedList(ctx, arg)
	if err != nil {
		return UnmatchedPage{}, fmt.Errorf("submission: danh sách chưa khớp: %w", err)
	}
	out := UnmatchedPage{Items: make([]Unmatched, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].ReceivedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		u := Unmatched{ID: r.ID, FileName: r.FileName, Source: r.Source, ReceivedAt: r.ReceivedAt, AssignmentID: r.AssignmentID, Version: r.RowVersion}
		if r.UnmatchedReason != nil {
			u.Reason, u.ReasonText = *r.UnmatchedReason, ReasonText(*r.UnmatchedReason)
		}
		if r.CandidateStudentID != nil {
			u.Suggestion = &Suggested{StudentID: *r.CandidateStudentID, FullName: r.CandidateName}
		}
		out.Items = append(out.Items, u)
	}
	return out, nil
}

func (s *Service) staffTx(ctx context.Context, fn func(tx pgx.Tx, q *store.Queries) error) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error { return fn(tx, store.New(tx)) })
}

func (s *Service) audit(ctx context.Context, q *store.Queries, st Staff, course, id uuid.UUID, action string, after any) error {
	raw, _ := json.Marshal(after)
	var tr *string
	if st.TraceID != "" {
		tr = &st.TraceID
	}
	_, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &st.UserID, Entity: "submissions", EntityID: id.String(), Action: action, After: raw, TraceID: tr})
	return err
}

func versionConflict() *apierr.Error { return apierr.New(http.StatusConflict, apierr.VersionConflict) }

// taBlocked: TA không sửa sau khi đã có điểm công bố của bài (Q5) → 403.
func (s *Service) taBlocked(ctx context.Context, q *store.Queries, st Staff, course uuid.UUID, aid *uuid.UUID) error {
	if !st.IsTA || aid == nil {
		return nil
	}
	has, err := q.SubmissionAssignmentHasPublished(ctx, store.SubmissionAssignmentHasPublishedParams{CourseID: course, AssignmentID: *aid})
	if err != nil {
		return fmt.Errorf("submission: kiểm điểm công bố: %w", err)
	}
	if has {
		return apierr.New(http.StatusForbidden, apierr.Forbidden)
	}
	return nil
}

// MatchIn là thân PUT …/submissions/{id}/match.
type MatchIn struct {
	AssignmentID uuid.UUID `json:"assignment_id"`
	StudentID    uuid.UUID `json:"student_id"`
	Version      int64     `json:"version"`
}

// MatchOut là kết quả gán tay.
type MatchOut struct {
	ID           uuid.UUID `json:"id"`
	AssignmentID uuid.UUID `json:"assignment_id"`
	StudentID    uuid.UUID `json:"student_id"`
	Version      int       `json:"version"`
}

// Match gán tay (SRS 4.4.4): sinh viên phải ACTIVE (422 STUDENT_NOT_IN_COURSE); bản mới nhất của họ đã REVIEWED / PUBLISHED → 409 ALREADY_REVIEWED; sai version → 409 VERSION_CONFLICT;
// version bài nộp = max + 1 dưới khoá nộp chung; audit; outbox `submission.resolved`.
func (s *Service) Match(ctx context.Context, st Staff, course, id uuid.UUID, in MatchIn) (MatchOut, error) {
	var out MatchOut
	err := s.staffTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		row, err := q.SubmissionGetForUpdate(ctx, store.SubmissionGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return fmt.Errorf("submission: đọc bài nộp: %w", err)
		}
		if row.MatchStatus != "UNMATCHED" || row.DismissedAt != nil || row.RowVersion != in.Version {
			return versionConflict()
		}
		if _, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: in.AssignmentID}); errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
		} else if err != nil {
			return fmt.Errorf("submission: đọc bài tập: %w", err)
		}
		if err := s.taBlocked(ctx, q, st, course, &in.AssignmentID); err != nil {
			return err
		}
		active, err := q.SubmissionStudentActive(ctx, store.SubmissionStudentActiveParams{CourseID: course, UserID: in.StudentID})
		if err != nil {
			return fmt.Errorf("submission: kiểm sinh viên: %w", err)
		}
		if !active {
			return apierr.New(http.StatusUnprocessableEntity, apierr.StudentNotInCourse)
		}
		if err := q.SubmissionLockStudent(ctx, store.SubmissionLockStudentParams{AssignmentID: in.AssignmentID.String(), StudentID: in.StudentID.String()}); err != nil {
			return fmt.Errorf("submission: khoá nộp: %w", err)
		}
		if cur, err := q.SubmissionLatestMatched(ctx, store.SubmissionLatestMatchedParams{CourseID: course, AssignmentID: in.AssignmentID, StudentID: in.StudentID}); err == nil {
			if cur.ResultStatus == "REVIEWED" || cur.ResultStatus == "PUBLISHED" {
				return apierr.New(http.StatusConflict, apierr.AlreadyReviewed)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("submission: bản mới nhất: %w", err)
		}
		m, err := q.SubmissionMatchManual(ctx, store.SubmissionMatchManualParams{CourseID: course, ID: id, AssignmentID: &in.AssignmentID, StudentID: &in.StudentID})
		if err != nil {
			return fmt.Errorf("submission: gán tay: %w", err)
		}
		if err := s.audit(ctx, q, st, course, id, "match", map[string]any{"assignment_id": in.AssignmentID, "student_id": in.StudentID, "method": MethodManual}); err != nil {
			return fmt.Errorf("submission: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicResolved, map[string]any{"course_id": course.String(), "user_id": in.StudentID.String(), "submission_id": id.String()}); err != nil {
			return fmt.Errorf("submission: outbox: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicCreated, map[string]any{"course_id": course.String(), "assignment_id": in.AssignmentID.String(), "student_id": in.StudentID.String(), "user_id": in.StudentID.String(),
			"submission_id": id.String(), "version": m.Version, "source": "MANUAL", "matched": true}); err != nil {
			return fmt.Errorf("submission: outbox: %w", err)
		}
		v := 0
		if m.Version != nil {
			v = int(*m.Version)
		}
		out = MatchOut{ID: id, AssignmentID: in.AssignmentID, StudentID: in.StudentID, Version: v}
		return nil
	})
	return out, err
}

// Dismiss: `Bỏ qua` một bài chưa khớp (1–200 ký tự lý do; không xoá bản ghi / tệp).
func (s *Service) Dismiss(ctx context.Context, st Staff, course, id uuid.UUID, reason string, version int64) error {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 1 || n > 200 {
		return apierr.Validation(apierr.FieldError{Field: "reason", Code: "length", Message: "Lý do dài 1–200 ký tự."})
	}
	return s.staffTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		row, err := q.SubmissionGetForUpdate(ctx, store.SubmissionGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return fmt.Errorf("submission: đọc bài nộp: %w", err)
		}
		if row.MatchStatus != "UNMATCHED" || row.DismissedAt != nil || row.RowVersion != version {
			return versionConflict()
		}
		if err := s.taBlocked(ctx, q, st, course, row.AssignmentID); err != nil {
			return err
		}
		if _, err := q.SubmissionDismiss(ctx, store.SubmissionDismissParams{CourseID: course, ID: id, Actor: &st.UserID, Reason: &reason}); err != nil {
			return fmt.Errorf("submission: bỏ qua: %w", err)
		}
		if err := s.audit(ctx, q, st, course, id, "dismiss", map[string]any{"reason": reason}); err != nil {
			return fmt.Errorf("submission: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicResolved, map[string]any{"course_id": course.String(), "submission_id": id.String()}); err != nil {
			return fmt.Errorf("submission: outbox: %w", err)
		}
		return nil
	})
}

// Undismiss hoàn tác `Bỏ qua` (giao diện cho 5 giây).
func (s *Service) Undismiss(ctx context.Context, st Staff, course, id uuid.UUID) error {
	return s.staffTx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		row, err := q.SubmissionGetForUpdate(ctx, store.SubmissionGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return fmt.Errorf("submission: đọc bài nộp: %w", err)
		}
		if row.DismissedAt == nil {
			return apierr.New(http.StatusConflict, apierr.Conflict)
		}
		if err := s.taBlocked(ctx, q, st, course, row.AssignmentID); err != nil {
			return err
		}
		if _, err := q.SubmissionUndismiss(ctx, store.SubmissionUndismissParams{CourseID: course, ID: id}); err != nil {
			return fmt.Errorf("submission: hoàn tác bỏ qua: %w", err)
		}
		if err := s.audit(ctx, q, st, course, id, "undismiss", nil); err != nil {
			return fmt.Errorf("submission: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicResolved, map[string]any{"course_id": course.String(), "submission_id": id.String()}); err != nil {
			return fmt.Errorf("submission: outbox: %w", err)
		}
		return nil
	})
}

// FileURL: `GET …/submissions/{id}/file` → URL ký 5 phút (attachment), chỉ phát sau khi route đã kiểm vai.
func (s *Service) FileURL(ctx context.Context, course, id uuid.UUID) (string, error) {
	row, err := store.New(s.Pool).SubmissionFileKey(ctx, store.SubmissionFileKeyParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return "", fmt.Errorf("submission: đọc khoá tệp: %w", err)
	}
	if s.Blob == nil {
		return "", apierr.New(http.StatusServiceUnavailable, apierr.ServiceUnavailable)
	}
	u, err := s.Blob.PresignGet(ctx, row.BlobKey, row.FileName)
	if err != nil {
		return "", fmt.Errorf("submission: ký URL tải: %w", err)
	}
	return u, nil
}
