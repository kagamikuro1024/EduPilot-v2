package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/regress"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic outbox của phúc khảo (US-P7-10).
const (
	TopicAppealCreated = "assignment.appeal_created"
	TopicAppealDecided = "assignment.appeal_decided"
	appealReasonMin    = 10
	appealReasonMax    = 1000
	appealResponseMin  = 10
	appealResponseMax  = 2000
)

// AppealIn là thân `POST …/assignments/{aid}/appeal`.
type AppealIn struct {
	CriteriaIDs []string `json:"criteria_ids"`
	Reason      string   `json:"reason"`
}

// AppealCreated là phản hồi `201`.
type AppealCreated struct {
	Appeal struct {
		ID        uuid.UUID `json:"id"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"appeal"`
}

// cleanText cắt khoảng trắng và loại ký tự điều khiển (lý do chỉ tới giảng viên: không qua tường lửa PII, không tới LLM).
func cleanText(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s))
}

// FileAppeal gửi phúc khảo (SRS 4.10.1): kết quả của chính sinh viên đã PUBLISHED, còn hạn (giờ máy chủ), tiêu chí ⊆ rubric, lý do 10–1.000, mỗi bài một lần. Không ghi escalation_tickets; không gọi LLM.
func (s *Service) FileAppeal(ctx context.Context, a Actor, course, aid uuid.UUID, in AppealIn) (AppealCreated, error) {
	var out AppealCreated
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		c, err := q.GradeAppealContext(ctx, store.GradeAppealContextParams{CourseID: course, AssignmentID: aid, StudentID: a.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.AssignmentNotFound) // chưa công bố (kể cả đã nộp) → không lộ nháp
		}
		if err != nil {
			return fmt.Errorf("assignment: ngữ cảnh phúc khảo: %w", err)
		}
		switch {
		case c.AppealDays == 0:
			return apierr.New(http.StatusConflict, apierr.AppealDisabled)
		case c.AppealID != uuid.Nil:
			return apierr.New(http.StatusConflict, apierr.AppealAlreadyFiled)
		}
		pub := time.Time{}
		if c.PublishedAt != nil {
			pub = *c.PublishedAt
		}
		closes := pub.Add(time.Duration(c.AppealDays) * 24 * time.Hour)
		if s.now().After(closes) {
			return apierr.New(http.StatusConflict, apierr.AppealWindowClosed).WithDetails(map[string]any{"closes_at": closes})
		}
		var rubric []grading.RubricItem
		_ = json.Unmarshal(c.Rubric, &rubric)
		known := map[string]bool{}
		for _, r := range rubric {
			known[r.ID] = true
		}
		seen := map[string]bool{}
		ids := []string{}
		for _, id := range in.CriteriaIDs {
			if !known[id] {
				return apierr.New(http.StatusUnprocessableEntity, apierr.CriterionUnknown)
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return apierr.Validation(apierr.FieldError{Field: "criteria_ids", Code: "required", Message: "Chọn ít nhất một tiêu chí."})
		}
		reason := cleanText(in.Reason)
		if n := len([]rune(reason)); n < appealReasonMin || n > appealReasonMax {
			return apierr.New(http.StatusUnprocessableEntity, apierr.ReasonLength)
		}
		before := decimal.Zero
		if c.Total.Valid {
			before = c.Total.Decimal
		}
		row, err := q.GradeAppealInsert(ctx, store.GradeAppealInsertParams{CourseID: course, AssignmentID: aid, StudentID: a.UserID, SubmissionID: c.SubmissionID, ResultID: c.ResultID, CriteriaIds: ids, Reason: reason, ScoreBefore: before})
		if isUnique(err) {
			return apierr.New(http.StatusConflict, apierr.AppealAlreadyFiled)
		}
		if err != nil {
			return fmt.Errorf("assignment: ghi phúc khảo: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicAppealCreated, map[string]any{"course_id": course, "appeal_id": row.ID, "assignment_id": aid}); err != nil {
			return fmt.Errorf("assignment: outbox: %w", err)
		}
		out.Appeal.ID, out.Appeal.Status, out.Appeal.CreatedAt = row.ID, "OPEN", row.CreatedAt
		return nil
	})
	return out, err
}

// DecideIn là thân `POST …/appeals/{id}/decide`.
type DecideIn struct {
	Decision       string       `json:"decision"`
	Response       string       `json:"response"`
	CriteriaScores []ScoreEntry `json:"criteria_scores"`
	Version        int32        `json:"version"`
}

// ScoreEntry là điểm mới của một tiêu chí (chuỗi hoặc số JSON).
type ScoreEntry struct {
	ID    string `json:"id"`
	Score Num    `json:"score"`
}

// DecideOut là phản hồi quyết định.
type DecideOut struct {
	ID         uuid.UUID `json:"id"`
	Status     string    `json:"status"`
	ScoreAfter *string   `json:"score_after"`
	Version    int32     `json:"version"`
}

// DecideAppeal quyết định phúc khảo (SRS 4.10.2–4.10.4): giữ nguyên hoặc sửa điểm TẠI CHỖ trên kết quả đã công bố, phản hồi bắt buộc; đóng ngay.
// Giao dịch mở đầu bằng `FOR SHARE` hàng gradebooks (tuần tự với Chốt điểm); sổ LOCKED không chặn.
func (s *Service) DecideAppeal(ctx context.Context, a Actor, course, id uuid.UUID, in DecideIn) (DecideOut, error) {
	var out DecideOut
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if in.Decision != "UPHELD" && in.Decision != "ADJUSTED" {
			return apierr.Validation(apierr.FieldError{Field: "decision", Code: "enum", Message: "Quyết định là UPHELD hoặc ADJUSTED."})
		}
		resp := cleanText(in.Response)
		if n := len([]rune(resp)); n < appealResponseMin || n > appealResponseMax {
			return apierr.New(http.StatusUnprocessableEntity, apierr.ResponseRequired)
		}
		if err := q.GradebookEnsure(ctx, course); err != nil {
			return fmt.Errorf("assignment: sổ điểm: %w", err)
		}
		if _, err := q.GradebookStatusShare(ctx, course); err != nil {
			return fmt.Errorf("assignment: khoá sổ điểm: %w", err)
		}
		ap, err := q.GradeAppealLock(ctx, store.GradeAppealLockParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusNotFound, apierr.NotFound)
		}
		if err != nil {
			return fmt.Errorf("assignment: khoá phúc khảo: %w", err)
		}
		if ap.Status != "OPEN" {
			return apierr.New(http.StatusConflict, apierr.AppealAlreadyDecided)
		}
		if ap.Version != in.Version {
			return apierr.New(http.StatusConflict, apierr.VersionConflict)
		}
		var scoreAfter decimal.NullDecimal
		if in.Decision == "ADJUSTED" {
			var err error
			if scoreAfter, err = s.adjustPublished(ctx, q, course, ap, in.CriteriaScores); err != nil {
				return err
			}
			if err := s.captureAppeal(ctx, q, course, ap); err != nil { // M2: tiêu chí đã sửa ở phúc khảo → ca hồi quy GRADING_APPEAL
				return err
			}
		}
		n, err := q.GradeAppealDecide(ctx, store.GradeAppealDecideParams{Status: in.Decision, Response: &resp, ScoreAfter: scoreAfter, DecidedBy: &a.UserID, CourseID: course, ID: id, Version: in.Version})
		if err != nil {
			return fmt.Errorf("assignment: ghi quyết định: %w", err)
		}
		if n == 0 {
			return apierr.New(http.StatusConflict, apierr.AppealAlreadyDecided)
		}
		after := map[string]any{"status": in.Decision, "score_before": ap.ScoreBefore.StringFixed(2)}
		if scoreAfter.Valid {
			after["score_after"] = scoreAfter.Decimal.StringFixed(2)
		}
		if err := s.appealAudit(ctx, q, a, course, id, "appeal.decide", map[string]any{"status": "OPEN"}, after); err != nil {
			return fmt.Errorf("assignment: audit: %w", err)
		}
		if _, err := outbox.Write(ctx, tx, TopicAppealDecided, map[string]any{"course_id": course, "appeal_id": id, "student_id": ap.StudentID, "user_id": ap.StudentID, "assignment_id": ap.AssignmentID}); err != nil {
			return fmt.Errorf("assignment: outbox: %w", err)
		}
		out = DecideOut{ID: id, Status: in.Decision, Version: in.Version + 1}
		if scoreAfter.Valid {
			v := scoreAfter.Decimal.StringFixed(2)
			out.ScoreAfter = &v
		}
		return nil
	})
	return out, err
}

// adjustPublished đổi điểm tiêu chí của kết quả ĐÃ CÔNG BỐ tại chỗ: ≥ 1 tiêu chí đổi (422 NO_CHANGE), bước 0,25 ≤ max; khoản trừ muộn đang áp giữ nguyên.
func (s *Service) adjustPublished(ctx context.Context, q *store.Queries, course uuid.UUID, ap store.GradeAppealLockRow, scores []ScoreEntry) (decimal.NullDecimal, error) {
	var rubric []grading.RubricItem
	_ = json.Unmarshal(ap.Rubric, &rubric)
	maxOf := map[string]decimal.Decimal{}
	for _, r := range rubric {
		m, _ := decimal.NewFromString(r.MaxScore)
		maxOf[r.ID] = m
	}
	var crits []critRow
	_ = json.Unmarshal(ap.Criteria, &crits)
	byID := map[string]*critRow{}
	for i := range crits {
		byID[crits[i].ID] = &crits[i]
	}
	changed := false
	for _, e := range scores {
		c := byID[e.ID]
		if c == nil {
			return decimal.NullDecimal{}, apierr.New(http.StatusUnprocessableEntity, apierr.CriterionUnknown)
		}
		d, err := parseScore(string(e.Score), maxOf[e.ID], true)
		if err != nil {
			return decimal.NullDecimal{}, err
		}
		ns := d.StringFixed(2)
		if old, err := decimal.NewFromString(deref(c.Score)); err != nil || !old.Equal(d) {
			c.Score, c.Edited, changed = &ns, true, true
		}
	}
	if !changed {
		return decimal.NullDecimal{}, apierr.New(http.StatusUnprocessableEntity, apierr.NoChange)
	}
	sum, ok := criteriaSum(mustJSON(crits))
	if !ok {
		return decimal.NullDecimal{}, apierr.New(http.StatusUnprocessableEntity, apierr.ScoresRequired)
	}
	total := grading.TotalAfterLate(sum, ap.LatePenalty)
	if n, err := q.GradeAppealAdjustResult(ctx, store.GradeAppealAdjustResultParams{Criteria: mustJSON(crits), Total: decimal.NullDecimal{Decimal: total, Valid: true}, CourseID: course, ID: ap.ResultID}); err != nil {
		return decimal.NullDecimal{}, fmt.Errorf("assignment: cập nhật kết quả đã công bố: %w", err)
	} else if n == 0 {
		return decimal.NullDecimal{}, apierr.New(http.StatusConflict, apierr.VersionConflict)
	}
	return decimal.NullDecimal{Decimal: total, Valid: true}, nil
}

// captureAppeal ghi ca GRADING_APPEAL cho các tiêu chí vừa đổi điểm (cờ REGRESSION_CAPTURE bật; so với điểm AI gốc).
func (s *Service) captureAppeal(ctx context.Context, q *store.Queries, course uuid.UUID, ap store.GradeAppealLockRow) error {
	if !s.Guidance.On(course, regress.Feature) {
		return nil
	}
	after, err := q.GradeAppealLock(ctx, store.GradeAppealLockParams{CourseID: course, ID: ap.ID})
	if err != nil {
		return fmt.Errorf("assignment: đọc điểm sau phúc khảo: %w", err)
	}
	ai, err := regress.AIScores(ctx, q, course, ap.ResultID)
	if err != nil {
		return err
	}
	return regress.CaptureGrading(ctx, q, s.Guidance, course, regress.KindGradingAppeal, ap.ResultID, ap.Rubric, after.Criteria, ai, s.RegressMinDelta, regress.ChangedCriteria(ap.Criteria, after.Criteria))
}

func (s *Service) appealAudit(ctx context.Context, q *store.Queries, a Actor, course, id uuid.UUID, action string, before, after any) error {
	b, _ := json.Marshal(before)
	af, _ := json.Marshal(after)
	var trace *string
	if a.TraceID != "" {
		trace = &a.TraceID
	}
	_, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &a.UserID, Entity: "grade_appeals", EntityID: id.String(), Action: action, Before: b, After: af, TraceID: trace})
	return err
}

// AppealStudent là sinh viên của phúc khảo.
type AppealStudent struct {
	ID          uuid.UUID `json:"id"`
	FullName    string    `json:"full_name"`
	StudentCode string    `json:"student_code"`
}

// AppealView là một phúc khảo cho nhân viên (SRS 4.10.7).
type AppealView struct {
	ID                uuid.UUID     `json:"id"`
	Assignment        AppealAsg     `json:"assignment"`
	Student           AppealStudent `json:"student"`
	SubmissionID      uuid.UUID     `json:"submission_id"`
	ResultID          uuid.UUID     `json:"result_id"`
	Criteria          []AppealCrit  `json:"criteria"`
	Reason            string        `json:"reason"`
	Status            string        `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	ScoreBefore       string        `json:"score_before"`
	AITotalBeforeLate *string       `json:"ai_total_before_late"`
	ScoreAfter        *string       `json:"score_after"`
	Response          *string       `json:"response"`
	DecidedBy         *uuid.UUID    `json:"decided_by"`
	DecidedAt         *time.Time    `json:"decided_at"`
	Version           int32         `json:"version"`
}

// AppealAsg là bài tập của phúc khảo.
type AppealAsg struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

// AppealCrit là tiêu chí sinh viên chọn.
type AppealCrit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AppealPage là một trang danh sách.
type AppealPage struct {
	Items      []AppealView `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

func critsOf(ids []string, rubricRaw json.RawMessage) []AppealCrit {
	var rubric []grading.RubricItem
	_ = json.Unmarshal(rubricRaw, &rubric)
	name := map[string]string{}
	for _, r := range rubric {
		name[r.ID] = r.Name
	}
	out := make([]AppealCrit, 0, len(ids))
	for _, id := range ids {
		out = append(out, AppealCrit{ID: id, Name: name[id]})
	}
	return out
}

// ListAppeals trả danh sách phúc khảo (nhân viên), con trỏ (created_at, id) mới nhất trước.
func (s *Service) ListAppeals(ctx context.Context, course uuid.UUID, assignmentID *uuid.UUID, status string, p httpx.PageParams) (AppealPage, error) {
	if status != "" && status != "OPEN" && status != "UPHELD" && status != "ADJUSTED" {
		return AppealPage{}, apierr.Validation(apierr.FieldError{Field: "status", Code: "enum", Message: "status là OPEN, UPHELD hoặc ADJUSTED."})
	}
	arg := store.GradeAppealListParams{CourseID: course, AssignmentID: assignmentID, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if status != "" {
		arg.Status = &status
	}
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := store.New(s.Pool).GradeAppealList(ctx, arg)
	if err != nil {
		return AppealPage{}, fmt.Errorf("assignment: danh sách phúc khảo: %w", err)
	}
	out := AppealPage{Items: make([]AppealView, 0, len(rows))}
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		c := httpx.EncodeCursor(rows[len(rows)-1].CreatedAt, rows[len(rows)-1].ID.String())
		out.NextCursor = &c
	}
	for _, r := range rows {
		v := AppealView{ID: r.ID, Assignment: AppealAsg{ID: r.AssignmentID, Title: r.AssignmentTitle}, Student: AppealStudent{ID: r.StudentID, FullName: r.StudentName, StudentCode: r.StudentCode},
			SubmissionID: r.SubmissionID, ResultID: r.ResultID, Criteria: critsOfIDs(r.CriteriaIds), Reason: r.Reason, Status: r.Status, CreatedAt: r.CreatedAt, ScoreBefore: r.ScoreBefore.StringFixed(2),
			Response: r.Response, DecidedBy: r.DecidedBy, DecidedAt: r.DecidedAt, Version: r.Version}
		v.AITotalBeforeLate, v.ScoreAfter = dec2(r.AiTotalBeforeLate), dec2(r.ScoreAfter)
		out.Items = append(out.Items, v)
	}
	return out, nil
}

func critsOfIDs(ids []string) []AppealCrit {
	out := make([]AppealCrit, 0, len(ids))
	for _, id := range ids {
		out = append(out, AppealCrit{ID: id})
	}
	return out
}

func dec2(d decimal.NullDecimal) *string {
	if !d.Valid {
		return nil
	}
	v := d.Decimal.StringFixed(2)
	return &v
}

// GetAppeal trả một phúc khảo (nhân viên) kèm tên tiêu chí.
func (s *Service) GetAppeal(ctx context.Context, course, id uuid.UUID) (AppealView, error) {
	r, err := store.New(s.Pool).GradeAppealGet(ctx, store.GradeAppealGetParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return AppealView{}, apierr.New(http.StatusNotFound, apierr.NotFound)
	}
	if err != nil {
		return AppealView{}, fmt.Errorf("assignment: đọc phúc khảo: %w", err)
	}
	v := AppealView{ID: r.ID, Assignment: AppealAsg{ID: r.AssignmentID, Title: r.AssignmentTitle}, Student: AppealStudent{ID: r.StudentID, FullName: r.StudentName, StudentCode: r.StudentCode},
		SubmissionID: r.SubmissionID, ResultID: r.ResultID, Criteria: critsOf(r.CriteriaIds, r.Rubric), Reason: r.Reason, Status: r.Status, CreatedAt: r.CreatedAt, ScoreBefore: r.ScoreBefore.StringFixed(2),
		Response: r.Response, DecidedBy: r.DecidedBy, DecidedAt: r.DecidedAt, Version: r.Version}
	v.AITotalBeforeLate, v.ScoreAfter = dec2(r.AiTotalBeforeLate), dec2(r.ScoreAfter)
	return v, nil
}
