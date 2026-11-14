// Package assignment: bài tập tự luận của giảng viên — tạo, rubric, công bố, sửa, xoá (SRS FEAT-assignments 4.2). Nghiệp vụ ở đây, handler mỏng ở httpapi/assignmenthttp.
// Điểm là decimal (chuỗi ở JSON); sinh viên không bao giờ nhận `reference_answer` hay `guidance`.
package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/httpapi/httpx"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/notify"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// Topic outbox (SRS 6): handler idempotent, khoá (topic, aggregate_id, version).
const (
	TopicPublished = "assignment.published"
	TopicUpdated   = "assignment.updated"
)

// TypePublished là loại thông báo chuông khi bài được công bố.
const TypePublished = "ASSIGNMENT_PUBLISHED"

//nolint:gochecknoglobals // hằng bất biến của gói (biểu thức chính quy, bước điểm, danh sách kênh)
var (
	codeRe  = regexp.MustCompile(`^[A-Z0-9]{2,12}$`)
	step    = decimal.RequireFromString("0.25")
	step05  = decimal.RequireFromString("0.05")
	hundred = decimal.NewFromInt(100)
	allCh   = []string{"IN_APP", "UPLOAD", "IMAP", "TEAMS"}
)

// Service là nghiệp vụ bài tập.
type Service struct {
	Pool   *pgxpool.Pool
	Log    *slog.Logger
	Notify *notify.Service
	LLM    grading.Structurer // SuggestRubric (task UTILITY → NEAR_REALTIME)
	Now    func() time.Time   // nil → time.Now; test đặt để kiểm hạn phúc khảo ở biên
	// Guidance: cờ «Lưu ý chấm» theo lớp (10.6). nil / lớp tắt → công bố như trước (không so phiên bản).
	Guidance *grading.GuidanceFlags
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Actor là người gọi.
type Actor struct {
	UserID  uuid.UUID
	Student bool
	TraceID string
}

// Input là thân tạo; Update dùng cùng hình + Version. Điểm là chuỗi decimal.
type Input struct {
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Type              string     `json:"type"`
	MaxScore          string     `json:"max_score"`
	DueAt             time.Time  `json:"due_at"`
	AllowLate         bool       `json:"allow_late"`
	LatePenaltyPerDay string     `json:"late_penalty_per_day"`
	LateMaxDays       int        `json:"late_max_days"`
	AppealDays        *int       `json:"appeal_days"`
	Channels          []string   `json:"channels"`
	SubmissionCode    string     `json:"submission_code"`
	ReferenceAnswer   *string    `json:"reference_answer"`
	TeamsClassID      *string    `json:"teams_class_id"`
	TeamsAssignmentID *string    `json:"teams_assignment_id"`
	Rubric            []RubricIn `json:"rubric"`
	Version           int        `json:"version"`
}

// RubricIn là một tiêu chí do giảng viên nhập; `id` rỗng → máy chủ cấp; có `id` → giữ.
type RubricIn struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MaxScore string `json:"max_score"`
	Guidance string `json:"guidance"`
}

// GradeLink là dòng "Tính vào điểm: …".
type GradeLink struct {
	ComponentName string `json:"component_name"`
	ComponentKey  string `json:"component_key"`
}

// Counts là Đã nộp / Tổng / Đã chấm / Đã công bố.
type Counts struct {
	Submitted int `json:"submitted"`
	Total     int `json:"total"`
	Graded    int `json:"graded"`
	Published int `json:"published"`
}

// Assignment là bản đầy đủ cho nhân viên.
type Assignment struct {
	ID                uuid.UUID            `json:"id"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	Type              string               `json:"type"`
	MaxScore          string               `json:"max_score"`
	DueAt             time.Time            `json:"due_at"`
	AllowLate         bool                 `json:"allow_late"`
	LatePenaltyPerDay string               `json:"late_penalty_per_day"`
	LateMaxDays       int                  `json:"late_max_days"`
	AppealDays        int                  `json:"appeal_days"`
	Channels          []string             `json:"channels"`
	SubmissionCode    string               `json:"submission_code"`
	ReferenceAnswer   *string              `json:"reference_answer"`
	Rubric            []grading.RubricItem `json:"rubric"`
	TeamsClassID      *string              `json:"teams_class_id"`
	TeamsAssignmentID *string              `json:"teams_assignment_id"`
	PublishedAt       *time.Time           `json:"published_at"`
	Version           int                  `json:"version"`
	CreatedAt         time.Time            `json:"created_at"`
	Counts            *Counts              `json:"counts,omitempty"`
	GradeLink         *GradeLink           `json:"grade_link,omitempty"`
	RubricLocked      bool                 `json:"rubric_locked"`
	LateRuleLocked    bool                 `json:"late_rule_locked"`
}

// StudentCriterion: sinh viên chỉ thấy tên + điểm tối đa của tiêu chí (không `guidance`).
type StudentCriterion struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MaxScore string `json:"max_score"`
}

// StudentView là bản rút gọn cho sinh viên: không `reference_answer`, không `guidance`, không mã nộp thư / cấu hình Teams.
type StudentView struct {
	ID                uuid.UUID          `json:"id"`
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	MaxScore          string             `json:"max_score"`
	DueAt             time.Time          `json:"due_at"`
	AllowLate         bool               `json:"allow_late"`
	LatePenaltyPerDay string             `json:"late_penalty_per_day"`
	LateMaxDays       int                `json:"late_max_days"`
	AppealDays        int                `json:"appeal_days"`
	Channels          []string           `json:"channels"`
	Rubric            []StudentCriterion `json:"rubric"`
	PublishedAt       time.Time          `json:"published_at"`
}

// Page là một trang danh sách. Staff: `Items`; Student: `StudentItems` — hai trường riêng để kiểu không trộn được.
type Page struct {
	Items        []Assignment      `json:"items"`
	StudentItems []StudentListItem `json:"-"`
	NextCursor   *string           `json:"next_cursor"`
}

func field(f, code, msg string) *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: f, Code: code, Message: msg})
}

func notFound() *apierr.Error { return apierr.New(http.StatusNotFound, apierr.NotFound) }

func (s *Service) requireActive(ctx context.Context, q *store.Queries, course uuid.UUID) error {
	st, err := q.CalCourseStatus(ctx, course)
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound()
	}
	if err != nil {
		return fmt.Errorf("assignment: đọc lớp: %w", err)
	}
	if st == store.CourseStatusARCHIVED {
		return apierr.New(http.StatusConflict, apierr.CourseArchived)
	}
	return nil
}

// normalized là Input đã kiểm và đổi sang kiểu có số.
type normalized struct {
	Input
	max, penalty decimal.Decimal
	appeal       int
	channels     []string
	ref          *string
}

// validate kiểm các trường (SRS 4.2.1). Rubric kiểm riêng (validateRubric).
func validate(in Input, create bool) (normalized, error) {
	n := normalized{Input: in}
	n.Title = strings.TrimSpace(in.Title)
	switch {
	case utf8.RuneCountInString(n.Title) < 1 || utf8.RuneCountInString(n.Title) > 120:
		return n, field("title", "length", "Tiêu đề dài 1–120 ký tự.")
	case utf8.RuneCountInString(in.Description) > 8000:
		return n, field("description", "length", "Mô tả tối đa 8.000 ký tự.")
	case in.DueAt.IsZero():
		return n, field("due_at", "required", "Thiếu hạn nộp.")
	}
	if in.Type == "" {
		n.Type = "ESSAY"
	}
	if n.Type != "ESSAY" {
		if n.Type == "QUIZ" {
			return n, apierr.New(http.StatusUnprocessableEntity, apierr.TypeNotSupported)
		}
		return n, field("type", "enum", "Loại bài tập là ESSAY.")
	}
	maxStr := in.MaxScore
	if maxStr == "" {
		maxStr = "10"
	}
	max, err := decimal.NewFromString(maxStr)
	if err != nil || !max.IsPositive() || max.GreaterThan(hundred) {
		return n, field("max_score", "range", "Điểm tối đa từ 0 đến 100.")
	}
	if !max.Mod(step).IsZero() {
		return n, apierr.New(http.StatusUnprocessableEntity, apierr.ScoreStep)
	}
	n.max = max
	pen := decimal.Zero
	if in.LatePenaltyPerDay != "" {
		if pen, err = decimal.NewFromString(in.LatePenaltyPerDay); err != nil || pen.IsNegative() || pen.GreaterThan(max) {
			return n, field("late_penalty_per_day", "range", "Khoản trừ mỗi ngày từ 0 đến điểm tối đa.")
		}
		if !pen.Mod(step05).IsZero() {
			return n, field("late_penalty_per_day", "step", "Khoản trừ mỗi ngày là bội của 0,05.")
		}
	}
	n.penalty = pen
	if in.LateMaxDays < 0 || in.LateMaxDays > 14 {
		return n, field("late_max_days", "range", "Số ngày nộp muộn tối đa từ 0 đến 14.")
	}
	if in.AllowLate && in.LateMaxDays < 1 {
		return n, field("late_max_days", "range", "Cho nộp muộn thì số ngày tối đa từ 1 đến 14.")
	}
	n.appeal = 7
	if in.AppealDays != nil {
		n.appeal = *in.AppealDays
	}
	if n.appeal < 0 || n.appeal > 30 {
		return n, field("appeal_days", "range", "Hạn phúc khảo từ 0 đến 30 ngày.")
	}
	n.channels = slices.Clone(in.Channels)
	if len(n.channels) == 0 {
		n.channels = []string{"IN_APP"}
	}
	slices.Sort(n.channels)
	n.channels = slices.Compact(n.channels)
	for _, c := range n.channels {
		if !slices.Contains(allCh, c) {
			return n, field("channels", "enum", "Kênh nhận bài không hợp lệ.")
		}
	}
	n.TeamsClassID, n.TeamsAssignmentID = blankToNil(in.TeamsClassID), blankToNil(in.TeamsAssignmentID) // "" = không có: ràng buộc `assignments_teams_chk` không chấp nhận chuỗi rỗng
	hasTeams := slices.Contains(n.channels, "TEAMS")
	if hasTeams != (n.TeamsClassID != nil) || (n.TeamsClassID == nil) != (n.TeamsAssignmentID == nil) {
		return n, field("teams_class_id", "required", "Kênh Teams cần cả mã lớp và mã bài tập Teams.")
	}
	if create && !codeRe.MatchString(in.SubmissionCode) {
		return n, field("submission_code", "format", "Mã nộp thư gồm 2–12 ký tự A–Z, 0–9.")
	}
	if in.ReferenceAnswer != nil {
		if utf8.RuneCountInString(*in.ReferenceAnswer) > 20000 {
			return n, field("reference_answer", "length", "Đáp án chuẩn tối đa 20.000 ký tự.")
		}
		if t := strings.TrimSpace(*in.ReferenceAnswer); t != "" {
			n.ref = &t
		}
	}
	return n, nil
}

// ValidateRubric kiểm tiêu chí (SRS 4.2.2): 1–10 tiêu chí, `max_score` > 0 và bội 0,25, Σ = điểm tối đa của bài (decimal). `existing` giữ id cũ; id mới cấp `c<n>` không trùng.
func ValidateRubric(in []RubricIn, max decimal.Decimal) ([]grading.RubricItem, error) {
	if len(in) < 1 || len(in) > 10 {
		return nil, field("rubric", "count", "Cần từ 1 đến 10 tiêu chí.")
	}
	used := map[string]bool{}
	for _, c := range in {
		if c.ID != "" {
			if used[c.ID] {
				return nil, field("rubric", "duplicate_id", "Mã tiêu chí bị trùng.")
			}
			used[c.ID] = true
		}
	}
	out := make([]grading.RubricItem, 0, len(in))
	sum := decimal.Zero
	next := 1
	for i, c := range in {
		name := strings.TrimSpace(c.Name)
		if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 120 {
			return nil, field(fmt.Sprintf("rubric[%d].name", i), "length", "Tên tiêu chí dài 1–120 ký tự.")
		}
		if utf8.RuneCountInString(c.Guidance) > 1000 {
			return nil, field(fmt.Sprintf("rubric[%d].guidance", i), "length", "Hướng dẫn chấm tối đa 1.000 ký tự.")
		}
		m, err := decimal.NewFromString(c.MaxScore)
		if err != nil || !m.IsPositive() || !m.Mod(step).IsZero() {
			return nil, field(fmt.Sprintf("rubric[%d].max_score", i), "step", "Điểm tiêu chí lớn hơn 0 và là bội của 0,25.")
		}
		id := c.ID
		if id == "" {
			for used[fmt.Sprintf("c%d", next)] {
				next++
			}
			id = fmt.Sprintf("c%d", next)
			used[id] = true
		}
		sum = sum.Add(m)
		out = append(out, grading.RubricItem{ID: id, Name: name, MaxScore: m.String(), Guidance: strings.TrimSpace(c.Guidance)})
	}
	if !sum.Equal(max) {
		return nil, apierr.New(http.StatusUnprocessableEntity, apierr.RubricInvalid).WithDetails(map[string]string{"sum": sum.String(), "max_score": max.String()})
	}
	return out, nil
}

func view(a store.Assignment) Assignment {
	var rubric []grading.RubricItem
	_ = json.Unmarshal(a.Rubric, &rubric)
	if rubric == nil {
		rubric = []grading.RubricItem{}
	}
	return Assignment{ID: a.ID, Title: a.Title, Description: a.Description, Type: a.Type, MaxScore: a.MaxScore.String(), DueAt: a.DueAt, AllowLate: a.AllowLate,
		LatePenaltyPerDay: a.LatePenaltyPerDay.String(), LateMaxDays: int(a.LateMaxDays), AppealDays: int(a.AppealDays), Channels: a.Channels, SubmissionCode: a.SubmissionCode,
		ReferenceAnswer: a.ReferenceAnswer, Rubric: rubric, TeamsClassID: a.TeamsClassID, TeamsAssignmentID: a.TeamsAssignmentID, PublishedAt: a.PublishedAt,
		Version: int(a.Version), CreatedAt: a.CreatedAt}
}

// StudentOf rút gọn cho sinh viên (chỉ bài đã công bố).
func StudentOf(a Assignment) StudentView {
	rb := make([]StudentCriterion, len(a.Rubric))
	for i, r := range a.Rubric {
		rb[i] = StudentCriterion{ID: r.ID, Name: r.Name, MaxScore: r.MaxScore}
	}
	v := StudentView{ID: a.ID, Title: a.Title, Description: a.Description, MaxScore: a.MaxScore, DueAt: a.DueAt, AllowLate: a.AllowLate, LatePenaltyPerDay: a.LatePenaltyPerDay,
		LateMaxDays: a.LateMaxDays, AppealDays: a.AppealDays, Channels: a.Channels, Rubric: rb}
	if a.PublishedAt != nil {
		v.PublishedAt = *a.PublishedAt
	}
	return v
}

func (s *Service) tx(ctx context.Context, fn func(tx pgx.Tx, q *store.Queries) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx, store.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) audit(ctx context.Context, q *store.Queries, a Actor, course, id uuid.UUID, action string, before, after any) error {
	b, _ := json.Marshal(before)
	af, _ := json.Marshal(after)
	var trace *string
	if a.TraceID != "" {
		trace = &a.TraceID
	}
	_, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, ActorID: &a.UserID, Entity: "assignments", EntityID: id.String(), Action: action, Before: b, After: af, TraceID: trace})
	return err
}

func blankToNil(p *string) *string {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	t := strings.TrimSpace(*p)
	return &t
}

func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}

// Create tạo bài ở trạng thái nháp (SRS 4.2.1). `POST` có Idempotency-Key ở middleware. Rubric (nếu gửi kèm) kiểm như PUT rubric.
func (s *Service) Create(ctx context.Context, a Actor, course uuid.UUID, in Input) (Assignment, error) {
	n, err := validate(in, true)
	if err != nil {
		return Assignment{}, err
	}
	rubric := []grading.RubricItem{}
	if len(in.Rubric) > 0 {
		if rubric, err = ValidateRubric(in.Rubric, n.max); err != nil {
			return Assignment{}, err
		}
	}
	rb, _ := json.Marshal(rubric)
	var out store.Assignment
	err = s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := s.requireActive(ctx, q, course); err != nil {
			return err
		}
		out, err = q.AssignmentInsert(ctx, store.AssignmentInsertParams{CourseID: course, Title: n.Title, Description: in.Description, Type: n.Type, MaxScore: n.max, DueAt: in.DueAt.UTC(),
			AllowLate: in.AllowLate, LatePenaltyPerDay: n.penalty, LateMaxDays: int16(in.LateMaxDays), AppealDays: int16(n.appeal), Channels: n.channels, //nolint:gosec // đã kiểm 0–14 / 0–30
			SubmissionCode: in.SubmissionCode, ReferenceAnswer: n.ref, Rubric: rb, TeamsClassID: n.TeamsClassID, TeamsAssignmentID: n.TeamsAssignmentID, CreatedBy: a.UserID})
		if isUnique(err) {
			return field("submission_code", "unique", "Mã nộp thư đã dùng trong lớp.")
		}
		if err != nil {
			return fmt.Errorf("assignment: tạo: %w", err)
		}
		return s.audit(ctx, q, a, course, out.ID, "create", nil, map[string]any{"title": out.Title, "submission_code": out.SubmissionCode})
	})
	if err != nil {
		return Assignment{}, err
	}
	return view(out), nil
}

// Get trả bản đầy đủ (kèm số liệu và liên kết sổ điểm); sinh viên dùng StudentOf và chỉ thấy bài đã công bố.
func (s *Service) Get(ctx context.Context, a Actor, course, id uuid.UUID) (Assignment, error) {
	q := store.New(s.Pool)
	row, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.Student && row.PublishedAt == nil) {
		return Assignment{}, notFound()
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("assignment: đọc: %w", err)
	}
	v := view(row)
	if a.Student {
		return v, nil
	}
	if err := s.decorate(ctx, q, course, []*Assignment{&v}); err != nil {
		return Assignment{}, err
	}
	facts, err := q.AssignmentLockFacts(ctx, store.AssignmentLockFactsParams{CourseID: course, ID: id})
	if err != nil {
		return Assignment{}, fmt.Errorf("assignment: khoá sửa: %w", err)
	}
	v.RubricLocked, v.LateRuleLocked = facts.HasResults, facts.HasPublished
	if l, err := q.AssignmentGradeLink(ctx, store.AssignmentGradeLinkParams{CourseID: course, ID: id}); err == nil {
		v.GradeLink = &GradeLink{ComponentName: l.ComponentName, ComponentKey: l.ComponentKey}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, fmt.Errorf("assignment: liên kết sổ điểm: %w", err)
	}
	return v, nil
}

func (s *Service) decorate(ctx context.Context, q *store.Queries, course uuid.UUID, list []*Assignment) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	for i, a := range list {
		ids[i] = a.ID.String()
	}
	cs, err := q.AssignmentCounts(ctx, store.AssignmentCountsParams{CourseID: course, Ids: ids})
	if err != nil {
		return fmt.Errorf("assignment: số liệu: %w", err)
	}
	by := map[uuid.UUID]store.AssignmentCountsRow{}
	for _, c := range cs {
		by[c.AssignmentID] = c
	}
	for _, a := range list {
		c := by[a.ID]
		a.Counts = &Counts{Submitted: int(c.Submitted), Total: int(c.Total), Graded: int(c.Graded), Published: int(c.Published)}
	}
	return nil
}

// List: con trỏ (created_at, id) giảm dần; sinh viên chỉ thấy bài đã công bố và không thấy `reference_answer` / `guidance`.
func (s *Service) List(ctx context.Context, a Actor, course uuid.UUID, p httpx.PageParams) (Page, error) {
	q := store.New(s.Pool)
	limit := p.Limit
	arg := store.AssignmentListParams{CourseID: course, PublishedOnly: a.Student, PageLimit: int32(p.Fetch())} //nolint:gosec // ≤ 101
	if p.Cursor != nil {
		arg.CursorAt, arg.CursorID = &p.Cursor.CreatedAt, &p.Cursor.ID
	}
	rows, err := q.AssignmentList(ctx, arg)
	if err != nil {
		return Page{}, fmt.Errorf("assignment: danh sách: %w", err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		c := httpx.EncodeCursor(last.CreatedAt, last.ID.String())
		next = &c
	}
	items := make([]Assignment, len(rows))
	for i, r := range rows {
		items[i] = view(r)
	}
	if a.Student {
		mine, err := s.myStates(ctx, q, course, a.UserID, items)
		if err != nil {
			return Page{}, err
		}
		p := Page{Items: []Assignment{}, NextCursor: next, StudentItems: make([]StudentListItem, 0, len(items))}
		for _, it := range items {
			p.StudentItems = append(p.StudentItems, StudentListItem{ID: it.ID, Title: it.Title, DueAt: it.DueAt, My: mine[it.ID]})
		}
		return p, nil
	}
	ptrs := make([]*Assignment, len(items))
	for i := range items {
		ptrs[i] = &items[i]
	}
	if err := s.decorate(ctx, q, course, ptrs); err != nil {
		return Page{}, err
	}
	return Page{Items: items, NextCursor: next}, nil
}

// SetRubric lưu rubric (chỉ TEACHER): khoá khi bài đã có kết quả chấm (409 RUBRIC_LOCKED).
func (s *Service) SetRubric(ctx context.Context, a Actor, course, id uuid.UUID, rubric []RubricIn, version int) (Assignment, error) {
	var out store.Assignment
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := s.requireActive(ctx, q, course); err != nil {
			return err
		}
		cur, err := q.AssignmentGetForUpdate(ctx, store.AssignmentGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc: %w", err)
		}
		if int(cur.Version) != version {
			return s.conflict(cur)
		}
		facts, err := q.AssignmentLockFacts(ctx, store.AssignmentLockFactsParams{CourseID: course, ID: id})
		if err != nil {
			return fmt.Errorf("assignment: khoá sửa: %w", err)
		}
		if facts.HasResults {
			return apierr.New(http.StatusConflict, apierr.RubricLocked)
		}
		items, err := ValidateRubric(rubric, cur.MaxScore)
		if err != nil {
			return err
		}
		rb, _ := json.Marshal(items)
		out, err = q.AssignmentSetRubric(ctx, store.AssignmentSetRubricParams{CourseID: course, ID: id, Rubric: rb, Version: int32(version)}) //nolint:gosec // version nhỏ
		if err != nil {
			return fmt.Errorf("assignment: lưu rubric: %w", err)
		}
		return s.audit(ctx, q, a, course, id, "rubric", json.RawMessage(cur.Rubric), items)
	})
	if err != nil {
		return Assignment{}, err
	}
	return view(out), nil
}

func (s *Service) conflict(cur store.Assignment) error {
	return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]any{"current": view(cur)})
}

// Update sửa (TEACHER, `version`): sau công bố khoá loại / điểm tối đa / mã nộp; khoá hạn + quy tắc muộn khi đã có kết quả PUBLISHED (TLR-8). Đổi hạn hoặc quy tắc muộn → outbox
// `assignment.updated` (handler tính lại khoản trừ muộn của kết quả DRAFT / REVIEWED, trừ khoản đã sửa tay).
func (s *Service) Update(ctx context.Context, a Actor, course, id uuid.UUID, in Input) (Assignment, error) {
	n, err := validate(in, false)
	if err != nil {
		return Assignment{}, err
	}
	var out store.Assignment
	err = s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := s.requireActive(ctx, q, course); err != nil {
			return err
		}
		cur, err := q.AssignmentGetForUpdate(ctx, store.AssignmentGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc: %w", err)
		}
		if int(cur.Version) != in.Version {
			return s.conflict(cur)
		}
		facts, err := q.AssignmentLockFacts(ctx, store.AssignmentLockFactsParams{CourseID: course, ID: id})
		if err != nil {
			return fmt.Errorf("assignment: khoá sửa: %w", err)
		}
		if cur.PublishedAt != nil || facts.HasResults { // sau công bố khoá loại / điểm tối đa / mã nộp (4.2.5)
			switch {
			case n.max.Cmp(cur.MaxScore) != 0:
				return field("max_score", "locked", "Đã công bố, không đổi điểm tối đa.")
			case in.SubmissionCode != "" && in.SubmissionCode != cur.SubmissionCode:
				return field("submission_code", "locked", "Đã công bố, không đổi mã nộp thư.")
			}
		}
		lateChanged := !in.DueAt.Equal(cur.DueAt) || in.AllowLate != cur.AllowLate || n.penalty.Cmp(cur.LatePenaltyPerDay) != 0 || int16(in.LateMaxDays) != cur.LateMaxDays //nolint:gosec // 0–14
		if lateChanged && facts.HasPublished {
			return apierr.New(http.StatusConflict, apierr.LateRuleLocked)
		}
		out, err = q.AssignmentUpdate(ctx, store.AssignmentUpdateParams{CourseID: course, ID: id, Version: int32(in.Version), Title: n.Title, Description: in.Description, MaxScore: n.max, //nolint:gosec // version nhỏ
			DueAt: in.DueAt.UTC(), AllowLate: in.AllowLate, LatePenaltyPerDay: n.penalty, LateMaxDays: int16(in.LateMaxDays), AppealDays: int16(n.appeal), Channels: n.channels, //nolint:gosec // đã kiểm
			ReferenceAnswer: n.ref, TeamsClassID: n.TeamsClassID, TeamsAssignmentID: n.TeamsAssignmentID})
		if errors.Is(err, pgx.ErrNoRows) {
			return s.conflict(cur)
		}
		if err != nil {
			return fmt.Errorf("assignment: sửa: %w", err)
		}
		if err := s.audit(ctx, q, a, course, id, "update", map[string]any{"due_at": cur.DueAt, "title": cur.Title, "version": cur.Version}, map[string]any{"due_at": out.DueAt, "title": out.Title, "version": out.Version}); err != nil {
			return err
		}
		if cur.PublishedAt != nil || lateChanged {
			if _, err := outbox.Write(ctx, tx, TopicUpdated, map[string]any{"course_id": course.String(), "assignment_id": id.String(), "version": out.Version, "late_changed": lateChanged}); err != nil {
				return fmt.Errorf("assignment: outbox: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return Assignment{}, err
	}
	return view(out), nil
}

// Delete xoá bài sạch (TEACHER): có bài nộp → 409 HAS_SUBMISSIONS; đang gắn vào công thức → 409 LINKED_TO_GRADEBOOK; ngược lại 204 + audit + vô hiệu lịch / "Hôm nay".
func (s *Service) Delete(ctx context.Context, a Actor, course, id uuid.UUID) error {
	return s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := s.requireActive(ctx, q, course); err != nil {
			return err
		}
		cur, err := q.AssignmentGetForUpdate(ctx, store.AssignmentGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc: %w", err)
		}
		facts, err := q.AssignmentLockFacts(ctx, store.AssignmentLockFactsParams{CourseID: course, ID: id})
		if err != nil {
			return fmt.Errorf("assignment: khoá sửa: %w", err)
		}
		switch {
		case facts.HasSubmissions:
			return apierr.New(http.StatusConflict, apierr.HasSubmissions)
		case facts.Linked:
			return apierr.New(http.StatusConflict, apierr.LinkedToGradebook)
		}
		if _, err := q.AssignmentDelete(ctx, store.AssignmentDeleteParams{CourseID: course, ID: id}); err != nil {
			return fmt.Errorf("assignment: xoá: %w", err)
		}
		if err := s.audit(ctx, q, a, course, id, "delete", map[string]any{"title": cur.Title}, nil); err != nil {
			return err
		}
		if cur.PublishedAt != nil {
			if _, err := outbox.Write(ctx, tx, TopicUpdated, map[string]any{"course_id": course.String(), "assignment_id": id.String(), "version": cur.Version + 1, "deleted": true}); err != nil {
				return fmt.Errorf("assignment: outbox: %w", err)
			}
		}
		return nil
	})
}

// PublishOut là kết quả công bố.
type PublishOut struct {
	Assignment Assignment `json:"assignment"`
	Students   int        `json:"students"`
}

// Publish công bố (TEACHER, Idempotency-Key ở middleware): cần rubric hợp lệ (422 RUBRIC_REQUIRED); đã công bố → 409 ALREADY_PUBLISHED; một giao dịch: published_at, audit, outbox.
func (s *Service) Publish(ctx context.Context, a Actor, course, id uuid.UUID) (PublishOut, error) {
	var out store.Assignment
	var n int
	err := s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		if err := s.requireActive(ctx, q, course); err != nil {
			return err
		}
		cur, err := q.AssignmentGetForUpdate(ctx, store.AssignmentGetForUpdateParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc: %w", err)
		}
		if cur.PublishedAt != nil {
			return apierr.New(http.StatusConflict, apierr.AlreadyPublished)
		}
		var rubric []RubricIn
		var items []grading.RubricItem
		_ = json.Unmarshal(cur.Rubric, &items)
		for _, r := range items {
			rubric = append(rubric, RubricIn{ID: r.ID, Name: r.Name, MaxScore: r.MaxScore, Guidance: r.Guidance})
		}
		if _, err := ValidateRubric(rubric, cur.MaxScore); err != nil {
			return apierr.New(http.StatusUnprocessableEntity, apierr.RubricRequired)
		}
		out, err = q.AssignmentPublish(ctx, store.AssignmentPublishParams{CourseID: course, ID: id})
		if err != nil {
			return fmt.Errorf("assignment: công bố: %w", err)
		}
		ids, err := q.AssignmentActiveStudents(ctx, course)
		if err != nil {
			return fmt.Errorf("assignment: sinh viên: %w", err)
		}
		n = len(ids)
		if err := s.audit(ctx, q, a, course, id, "publish", nil, map[string]any{"students": n}); err != nil {
			return err
		}
		if _, err := outbox.Write(ctx, tx, TopicPublished, map[string]any{"course_id": course.String(), "assignment_id": id.String(), "version": out.Version}); err != nil {
			return fmt.Errorf("assignment: outbox: %w", err)
		}
		return nil
	})
	if err != nil {
		return PublishOut{}, err
	}
	return PublishOut{Assignment: view(out), Students: n}, nil
}

// StudentCount là số sinh viên ACTIVE (cho hộp thoại "Công bố cho {n} sinh viên").
func (s *Service) StudentCount(ctx context.Context, course uuid.UUID) (int, error) {
	ids, err := store.New(s.Pool).AssignmentActiveStudents(ctx, course)
	return len(ids), err
}

// HandlePublished là outbox.Handler của `assignment.published`: chuông ASSIGNMENT_PUBLISHED cho sinh viên ACTIVE (khử trùng `assignment.published:{id}:{user}`).
func (s *Service) HandlePublished(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID     string `json:"course_id"`
		AssignmentID string `json:"assignment_id"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("assignment: payload %s: %w", m.Topic, err)
	}
	course, e1 := uuid.Parse(p.CourseID)
	id, e2 := uuid.Parse(p.AssignmentID)
	if e1 != nil || e2 != nil || s.Notify == nil {
		return nil
	}
	q := store.New(s.Pool)
	a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("assignment: đọc bài: %w", err)
	}
	ids, err := q.AssignmentActiveStudents(ctx, course)
	if err != nil {
		return fmt.Errorf("assignment: người nhận: %w", err)
	}
	sp := notify.Spec{CourseID: &course, Type: TypePublished, Title: fmt.Sprintf("Bài tập mới: %s", a.Title), Link: new("/assignments/" + id.String()), DedupeKey: "assignment.published:" + id.String()}
	if _, err := s.Notify.SendMany(ctx, ids, sp); err != nil {
		return fmt.Errorf("assignment: gửi thông báo: %w", err)
	}
	return nil
}

// HandleUpdated là outbox.Handler của `assignment.updated`: nếu hạn / quy tắc muộn đổi thì tính lại khoản trừ muộn của kết quả DRAFT / REVIEWED (trừ khoản đã sửa tay);
// kết quả REVIEWED mà tổng đổi quay về DRAFT (phải duyệt lại). Idempotent: tính lại từ dữ liệu hiện tại.
func (s *Service) HandleUpdated(ctx context.Context, m outbox.Message) error {
	var p struct {
		CourseID     string `json:"course_id"`
		AssignmentID string `json:"assignment_id"`
		LateChanged  bool   `json:"late_changed"`
		Deleted      bool   `json:"deleted"`
	}
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		return fmt.Errorf("assignment: payload %s: %w", m.Topic, err)
	}
	course, e1 := uuid.Parse(p.CourseID)
	id, e2 := uuid.Parse(p.AssignmentID)
	if e1 != nil || e2 != nil || !p.LateChanged || p.Deleted {
		return nil
	}
	return s.RecomputeLate(ctx, course, id)
}

// RecomputeLate tính lại `late_days` / `late_penalty` / `total` của kết quả DRAFT / REVIEWED của bài theo hạn và quy tắc HIỆN TẠI (TLR-8).
func (s *Service) RecomputeLate(ctx context.Context, course, id uuid.UUID) error {
	return s.tx(ctx, func(tx pgx.Tx, q *store.Queries) error {
		a, err := q.AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("assignment: đọc bài: %w", err)
		}
		rows, err := q.AssignmentRecomputeCandidates(ctx, store.AssignmentRecomputeCandidatesParams{CourseID: course, AssignmentID: id})
		if err != nil {
			return fmt.Errorf("assignment: kết quả cần tính lại: %w", err)
		}
		pol := grading.LatePolicy{AllowLate: a.AllowLate, PerDay: a.LatePenaltyPerDay, MaxDays: int(a.LateMaxDays)}
		for _, r := range rows {
			sum, ok := criteriaSum(r.Criteria)
			if !ok && r.AiTotalBeforeLate.Valid {
				sum, ok = r.AiTotalBeforeLate.Decimal, true
			}
			if r.LatePenaltyManual || !ok {
				continue
			}
			days := grading.LateDays(r.SubmittedAt, a.DueAt)
			pen := grading.LatePenalty(days, pol, sum)
			total := grading.TotalAfterLate(sum, pen)
			status := r.Status
			if r.Total.Valid && !total.Equal(r.Total.Decimal) && status == "REVIEWED" {
				status = "DRAFT"
			}
			if int(r.LateDays) == days && pen.Equal(r.LatePenalty) && r.Total.Valid && total.Equal(r.Total.Decimal) {
				continue
			}
			if err := q.AssignmentApplyLate(ctx, store.AssignmentApplyLateParams{CourseID: course, ID: r.ID, LateDays: int16(days), LatePenalty: pen, Total: decimal.NullDecimal{Decimal: total, Valid: true}, Status: status}); err != nil { //nolint:gosec // ≤ 14 + cắt ở LateDays
				return fmt.Errorf("assignment: áp khoản trừ: %w", err)
			}
		}
		return nil
	})
}

// SuggestRubric gợi ý tiêu chí từ đáp án chuẩn (TA, TEACHER): một llm.Structured, KHÔNG ghi DB; thiếu đáp án (< 50 ký tự) → 422 REFERENCE_ANSWER_REQUIRED.
func (s *Service) SuggestRubric(ctx context.Context, a Actor, course, id uuid.UUID) ([]grading.SuggestedCriterion, error) {
	cur, err := store.New(s.Pool).AssignmentGet(ctx, store.AssignmentGetParams{CourseID: course, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	}
	if err != nil {
		return nil, fmt.Errorf("assignment: đọc: %w", err)
	}
	ref := ""
	if cur.ReferenceAnswer != nil {
		ref = *cur.ReferenceAnswer
	}
	if s.LLM == nil {
		return nil, apierr.New(http.StatusServiceUnavailable, apierr.LLMUnavailable)
	}
	ctx = llm.WithIdentity(ctx, llm.Identity{UserID: &a.UserID, CourseID: &course})
	return grading.SuggestRubric(ctx, s.LLM, ref, cur.MaxScore)
}

// criteriaSum là tổng điểm các tiêu chí HIỆN HÀNH (kể cả điểm giảng viên đã sửa) trước khi trừ muộn; ok=false khi chưa có điểm tiêu chí nào (bài lỗi).
func criteriaSum(raw json.RawMessage) (decimal.Decimal, bool) {
	var cs []struct {
		Score *string `json:"score"`
	}
	if json.Unmarshal(raw, &cs) != nil || len(cs) == 0 {
		return decimal.Zero, false
	}
	sum := decimal.Zero
	for _, c := range cs {
		if c.Score == nil {
			return decimal.Zero, false
		}
		d, err := decimal.NewFromString(*c.Score)
		if err != nil {
			return decimal.Zero, false
		}
		sum = sum.Add(d)
	}
	return sum, true
}
