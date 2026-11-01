package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Trạng thái của chính sinh viên với một bài (SRS 4.3.4). GRADING = đã nộp và đã có kết quả chấm chưa công bố (mọi nhánh nháp / đang chạy / lỗi / duyệt chưa công bố);
// SUBMITTED = đã nộp, chưa có kết quả nào. Không bao giờ nêu điểm nháp, cờ hay tiến độ chi tiết.
const (
	StateNotSubmitted = "NOT_SUBMITTED"
	StateSubmitted    = "SUBMITTED"
	StateGrading      = "GRADING"
	StateGraded       = "GRADED"
)

// DTO sinh viên là struct RIÊNG + allowlist khoá JSON (`testdata/student_dto_allowlist.json`): thêm trường nhạy cảm vào đây phải sửa allowlist có chủ đích;
// KHÔNG dùng `omitempty` để che trường (trường vắng ≠ trường rỗng).

// MyState là phần "của tôi" trong danh sách bài của sinh viên.
type MyState struct {
	State       string     `json:"state"`
	LateDays    int        `json:"late_days"`
	SubmittedAt *time.Time `json:"submitted_at"`
}

// StudentListItem là một dòng của GET …/assignments cho sinh viên: không rubric, không đáp án.
type StudentListItem struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	DueAt time.Time `json:"due_at"`
	My    MyState   `json:"my"`
}

// MineAssignment là phần đề bài của GET …/mine.
type MineAssignment struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	DueAt             time.Time          `json:"due_at"`
	MaxScore          string             `json:"max_score"`
	Criteria          []StudentCriterion `json:"criteria"`
	AllowLate         bool               `json:"allow_late"`
	LatePenaltyPerDay string             `json:"late_penalty_per_day"`
	LateMaxDays       int                `json:"late_max_days"`
	AppealDays        int                `json:"appeal_days"`
}

// MineSubmission là bản nộp mới nhất của chính sinh viên (`late_days` tính khi đọc).
type MineSubmission struct {
	Version    int       `json:"version"`
	FileName   string    `json:"file_name"`
	ReceivedAt time.Time `json:"received_at"`
	LateDays   int       `json:"late_days"`
}

// MineCriterion là một tiêu chí của kết quả ĐÃ CÔNG BỐ: điểm / max, nhận xét và các trích dẫn đã xác minh có trong bài của chính sinh viên.
type MineCriterion struct {
	Name     string   `json:"name"`
	Score    string   `json:"score"`
	Max      string   `json:"max"`
	Comment  string   `json:"comment"`
	Evidence []string `json:"evidence"`
}

// MineLate là khoản trừ nộp muộn đã áp (nil khi không trừ).
type MineLate struct {
	Days    int    `json:"days"`
	Penalty string `json:"penalty"`
}

// MineAppeal là trạng thái phúc khảo của chính sinh viên (NONE | OPEN | UPHELD | ADJUSTED).
type MineAppeal struct {
	State    string     `json:"state"`
	ClosesAt *time.Time `json:"closes_at"`
	Response *string    `json:"response"`
}

// MineResult là kết quả ĐÃ CÔNG BỐ (SRS 4.8.7); nhãn chung "Nhận xét · đã được giảng viên duyệt". Không có điểm nháp, điểm AI, độ lệch, cờ, độ tin cậy, kết quả hai lượt, người duyệt.
type MineResult struct {
	Status          string          `json:"status"`
	Total           string          `json:"total"`
	Max             string          `json:"max"`
	Criteria        []MineCriterion `json:"criteria"`
	OverallFeedback string          `json:"overall_feedback"`
	Late            *MineLate       `json:"late"`
	PublishedAt     time.Time       `json:"published_at"`
	Label           string          `json:"label"`
	Appeal          MineAppeal      `json:"appeal"`
}

// ResultLabel là nhãn chung ở màn sinh viên (không nêu AI).
const ResultLabel = "Nhận xét · đã được giảng viên duyệt"

// MineOut là phản hồi GET …/assignments/{aid}/mine. `result` chỉ khác null khi kết quả ĐÃ CÔNG BỐ.
type MineOut struct {
	Assignment MineAssignment  `json:"assignment"`
	Submission *MineSubmission `json:"submission"`
	State      string          `json:"state"`
	Result     *MineResult     `json:"result"`
}

func stateOf(have bool, result string) string {
	switch {
	case !have:
		return StateNotSubmitted
	case result == "PUBLISHED":
		return StateGraded
	case result != "":
		return StateGrading
	}
	return StateSubmitted
}

// myStates trả trạng thái của sinh viên cho các bài của trang (một truy vấn).
func (s *Service) myStates(ctx context.Context, q *store.Queries, course, student uuid.UUID, items []Assignment) (map[uuid.UUID]MyState, error) {
	ids := make([]string, len(items))
	due := make(map[uuid.UUID]time.Time, len(items))
	for i, it := range items {
		ids[i], due[it.ID] = it.ID.String(), it.DueAt
	}
	rows, err := q.SubmissionMineBatch(ctx, store.SubmissionMineBatchParams{CourseID: course, StudentID: student, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("assignment: trạng thái của tôi: %w", err)
	}
	out := make(map[uuid.UUID]MyState, len(items))
	for _, it := range items {
		out[it.ID] = MyState{State: StateNotSubmitted}
	}
	for _, r := range rows {
		if r.AssignmentID == nil {
			continue
		}
		out[*r.AssignmentID] = MyState{State: stateOf(true, r.ResultStatus), LateDays: grading.LateDays(r.SubmittedAt, due[*r.AssignmentID]), SubmittedAt: r.SubmittedAt}
	}
	return out, nil
}

// Mine đọc phía sinh viên (chỉ của chính mình; bài chưa công bố → 404 ASSIGNMENT_NOT_FOUND).
func (s *Service) Mine(ctx context.Context, a Actor, course, id uuid.UUID) (MineOut, error) {
	q := store.New(s.Pool)
	row, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && row.PublishedAt == nil) {
		return MineOut{}, apierr.New(http.StatusNotFound, apierr.AssignmentNotFound)
	}
	if err != nil {
		return MineOut{}, fmt.Errorf("assignment: đọc: %w", err)
	}
	v := StudentOf(view(row))
	out := MineOut{State: StateNotSubmitted, Assignment: MineAssignment{ID: v.ID, Title: v.Title, Description: v.Description, DueAt: v.DueAt, MaxScore: v.MaxScore, Criteria: v.Rubric,
		AllowLate: v.AllowLate, LatePenaltyPerDay: v.LatePenaltyPerDay, LateMaxDays: v.LateMaxDays, AppealDays: v.AppealDays}}
	cur, err := q.SubmissionLatestMatched(ctx, store.SubmissionLatestMatchedParams{CourseID: course, AssignmentID: id, StudentID: a.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return MineOut{}, fmt.Errorf("assignment: bài nộp của tôi: %w", err)
	}
	if cur.Version != nil {
		out.Submission = &MineSubmission{Version: int(*cur.Version), FileName: cur.FileName, ReceivedAt: cur.ReceivedAt, LateDays: grading.LateDays(cur.SubmittedAt, row.DueAt)}
	}
	out.State = stateOf(true, cur.ResultStatus)
	if out.State == StateGraded {
		if out.Result, err = s.minePublished(ctx, q, course, id, a.UserID, row); err != nil {
			return MineOut{}, err
		}
	}
	return out, nil
}

// minePublished dựng phần `result` từ DUY NHẤT truy vấn chỉ đọc kết quả PUBLISHED (nháp không bao giờ qua đường này).
func (s *Service) minePublished(ctx context.Context, q *store.Queries, course, id, student uuid.UUID, a store.Assignment) (*MineResult, error) {
	r, err := q.MinePublishedResult(ctx, store.MinePublishedResultParams{CourseID: course, AssignmentID: id, StudentID: student})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("assignment: kết quả đã công bố: %w", err)
	}
	var rubric []grading.RubricItem
	_ = json.Unmarshal(a.Rubric, &rubric)
	var crits []critRow
	_ = json.Unmarshal(r.Criteria, &crits)
	byID := map[string]critRow{}
	for _, c := range crits {
		byID[c.ID] = c
	}
	out := &MineResult{Status: "PUBLISHED", Max: a.MaxScore.StringFixed(2), Criteria: []MineCriterion{}, Label: ResultLabel, Appeal: MineAppeal{State: "NONE"}}
	if r.Total.Valid {
		out.Total = r.Total.Decimal.StringFixed(2)
	}
	if r.OverallFeedback != nil {
		out.OverallFeedback = *r.OverallFeedback
	}
	if r.PublishedAt != nil {
		out.PublishedAt = *r.PublishedAt
		if a.AppealDays > 0 {
			t := r.PublishedAt.Add(time.Duration(a.AppealDays) * 24 * time.Hour)
			out.Appeal.ClosesAt = &t
		}
	}
	for _, ri := range rubric {
		c := byID[ri.ID]
		mc := MineCriterion{Name: ri.Name, Max: fix2(ri.MaxScore), Comment: c.Comment, Evidence: []string{}}
		if c.Score != nil {
			mc.Score = fix2(*c.Score)
		}
		for _, e := range c.Evidence {
			mc.Evidence = append(mc.Evidence, e.Quote)
		}
		out.Criteria = append(out.Criteria, mc)
	}
	if r.LatePenalty.IsPositive() {
		out.Late = &MineLate{Days: int(r.LateDays), Penalty: r.LatePenalty.StringFixed(2)}
	}
	if ap, err := q.MineAppealOf(ctx, store.MineAppealOfParams{CourseID: course, ResultID: r.ID}); err == nil {
		out.Appeal.State, out.Appeal.Response = ap.Status, ap.Response
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("assignment: phúc khảo của tôi: %w", err)
	}
	return out, nil
}
