package exam

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn của bài thi (SRS 4.2.1). `Limits` lấy từ cấu hình; số 0 = mặc định.
type Limits struct {
	MinDurationMinutes int // EXAM_MIN_DURATION_MINUTES (5)
	MinLeadSeconds     int // EXAM_MIN_LEAD_SECONDS (60)
	MaxTotalSeconds    int // JUDGE_MAX_TOTAL_SECONDS (300)
}

func (l Limits) minDuration() int {
	if l.MinDurationMinutes <= 0 {
		return 5
	}
	return l.MinDurationMinutes
}

func (l Limits) minLead() time.Duration {
	if l.MinLeadSeconds < 0 {
		return 0
	}
	if l.MinLeadSeconds == 0 {
		return 60 * time.Second
	}
	return time.Duration(l.MinLeadSeconds) * time.Second
}

func (l Limits) maxTotal() int {
	if l.MaxTotalSeconds <= 0 {
		return 300
	}
	return l.MaxTotalSeconds
}

// Trạng thái bài thi (enum `exam_status`).
const (
	StatusDraft     = "DRAFT"
	StatusScheduled = "SCHEDULED"
	StatusOpen      = "OPEN"
	StatusClosed    = "CLOSED"
	StatusPublished = "PUBLISHED"
)

// Chủ đề outbox của bài thi (SRS 4.10).
const (
	TopicExamScheduled   = "exam.scheduled"
	TopicExamUnscheduled = "exam.unscheduled"
	TopicExamOpened      = "exam.opened"
	TopicExamClosed      = "exam.closed"
)

// EffectiveStatus là trạng thái theo `now` (SRS 3.2): mọi đường đọc / ghi dùng nó nên bộ lập lịch chậm vài giây không cho ai làm bài ngoài giờ.
func EffectiveStatus(status string, opens, closes *time.Time, now time.Time) string {
	switch status {
	case StatusScheduled, StatusOpen:
		if closes != nil && !now.Before(*closes) {
			return StatusClosed
		}
		if status == StatusScheduled && opens != nil && !now.Before(*opens) {
			return StatusOpen
		}
	}
	return status
}

// ExamIn là thân tạo / sửa bài thi. Khi SỬA, trường nil = giữ nguyên; khi TẠO, nil = mặc định (SRS 4.2.1). `kind` KHÔNG nhận từ client.
type ExamIn struct {
	Title            *string          `json:"title"`
	Instructions     *string          `json:"instructions"`
	OpensAt          *time.Time       `json:"opens_at"`
	ClosesAt         *time.Time       `json:"closes_at"`
	DurationMinutes  *int             `json:"duration_minutes"`
	ShuffleQuestions *bool            `json:"shuffle_questions"`
	ShuffleOptions   *bool            `json:"shuffle_options"`
	MaxScore         *decimal.Decimal `json:"max_score"`
	RoundingStep     *decimal.Decimal `json:"rounding_step"`
	MultiScoring     *string          `json:"multi_scoring"`
	RevealAnswers    *bool            `json:"reveal_answers"`
	AppealDays       *int             `json:"appeal_days"`
	Version          *int             `json:"version"`
}

// examState là trạng thái đã trộn (mặc định / bản ghi cũ + thay đổi) của một bài trước khi ghi.
type examState struct {
	Title            string
	Instructions     *string
	OpensAt          *time.Time
	ClosesAt         *time.Time
	DurationMinutes  *int
	ShuffleQuestions bool
	ShuffleOptions   bool
	MaxScore         decimal.Decimal
	RoundingStep     decimal.Decimal
	MultiScoring     string
	RevealAnswers    bool
	AppealDays       int
}

func hundred() decimal.Decimal { return decimal.NewFromInt(100) }

func defaults() examState {
	return examState{ShuffleQuestions: true, ShuffleOptions: true, MaxScore: decimal.RequireFromString("10.00"), RoundingStep: decimal.RequireFromString("0.01"), MultiScoring: "PARTIAL", RevealAnswers: true, AppealDays: 7}
}

// validStep: bước làm tròn cho phép (CHECK `exams_step_chk`): 0,01 · 0,10 · 0,25 · 0,50 · 1,00.
func validStep(d decimal.Decimal) bool {
	switch d.StringFixed(2) {
	case "0.01", "0.10", "0.25", "0.50", "1.00":
		return d.Equal(decimal.RequireFromString(d.StringFixed(2)))
	}
	return false
}

func stateOf(e store.Exam) examState {
	st := examState{Title: e.Title, Instructions: e.Instructions, OpensAt: e.OpensAt, ClosesAt: e.ClosesAt, ShuffleQuestions: e.ShuffleQuestions, ShuffleOptions: e.ShuffleOptions,
		MaxScore: e.MaxScore, RoundingStep: e.RoundingStep, MultiScoring: string(e.MultiScoring), RevealAnswers: e.RevealAnswers, AppealDays: int(e.AppealDays)}
	if e.DurationMinutes != nil {
		d := int(*e.DurationMinutes)
		st.DurationMinutes = &d
	}
	return st
}

// apply trộn các trường người dùng gửi lên `st`.
func (st examState) apply(in ExamIn) examState {
	if in.Title != nil {
		st.Title = *in.Title
	}
	if in.Instructions != nil {
		st.Instructions = in.Instructions
	}
	if in.OpensAt != nil {
		st.OpensAt = in.OpensAt
	}
	if in.ClosesAt != nil {
		st.ClosesAt = in.ClosesAt
	}
	if in.DurationMinutes != nil {
		st.DurationMinutes = in.DurationMinutes
	}
	if in.ShuffleQuestions != nil {
		st.ShuffleQuestions = *in.ShuffleQuestions
	}
	if in.ShuffleOptions != nil {
		st.ShuffleOptions = *in.ShuffleOptions
	}
	if in.MaxScore != nil {
		st.MaxScore = *in.MaxScore
	}
	if in.RoundingStep != nil {
		st.RoundingStep = *in.RoundingStep
	}
	if in.MultiScoring != nil {
		st.MultiScoring = *in.MultiScoring
	}
	if in.RevealAnswers != nil {
		st.RevealAnswers = *in.RevealAnswers
	}
	if in.AppealDays != nil {
		st.AppealDays = *in.AppealDays
	}
	return st
}

func ve(field, code, msg string) apierr.FieldError {
	return apierr.FieldError{Field: field, Code: code, Message: msg}
}

// validate trả MỌI lỗi trường (không dừng ở lỗi đầu). Khung giờ chỉ kiểm khi có đủ hai mốc (bản nháp / bản sao có thể chưa có giờ).
func (l Limits) validate(st examState) []apierr.FieldError {
	var errs []apierr.FieldError
	if n := utf8.RuneCountInString(st.Title); n < 1 || n > 120 {
		errs = append(errs, ve("title", "TITLE_LENGTH", "Tiêu đề từ 1 đến 120 ký tự."))
	}
	if st.Instructions != nil && utf8.RuneCountInString(*st.Instructions) > 4000 {
		errs = append(errs, ve("instructions", "INSTRUCTIONS_TOO_LONG", "Hướng dẫn tối đa 4.000 ký tự."))
	}
	if !st.MaxScore.IsPositive() || st.MaxScore.GreaterThan(hundred()) || st.MaxScore.Exponent() < -2 {
		errs = append(errs, ve("max_score", "LIMIT_OUT_OF_RANGE", "Điểm tối đa lớn hơn 0, không quá 100, tối đa 2 chữ số thập phân."))
	}
	if !validStep(st.RoundingStep) {
		errs = append(errs, ve("rounding_step", "LIMIT_OUT_OF_RANGE", "Bước làm tròn là 0,01, 0,10, 0,25, 0,50 hoặc 1,00."))
	}
	if st.MultiScoring != "PARTIAL" && st.MultiScoring != "ALL_OR_NOTHING" {
		errs = append(errs, ve("multi_scoring", "INVALID_TYPE", "Cách tính câu nhiều đáp án là PARTIAL hoặc ALL_OR_NOTHING."))
	}
	if st.AppealDays < 0 || st.AppealDays > 30 {
		errs = append(errs, ve("appeal_days", "LIMIT_OUT_OF_RANGE", "Hạn xem lại điểm từ 0 đến 30 ngày."))
	}
	var window time.Duration
	haveWindow := st.OpensAt != nil && st.ClosesAt != nil
	if haveWindow {
		window = st.ClosesAt.Sub(*st.OpensAt)
		if window <= 0 {
			errs = append(errs, ve("closes_at", "CLOSES_BEFORE_OPENS", "Giờ đóng phải sau giờ mở."))
		}
	}
	if st.DurationMinutes != nil {
		d := *st.DurationMinutes
		switch {
		case d < l.minDuration():
			errs = append(errs, ve("duration_minutes", "DURATION_TOO_SHORT", fmt.Sprintf("Thời lượng tối thiểu %d phút.", l.minDuration())))
		case d > 300:
			errs = append(errs, ve("duration_minutes", "LIMIT_OUT_OF_RANGE", "Thời lượng tối đa 300 phút."))
		case haveWindow && window > 0 && time.Duration(d)*time.Minute > window:
			errs = append(errs, ve("duration_minutes", "DURATION_EXCEEDS_WINDOW", "Thời lượng dài hơn khung giờ mở bài."))
		}
	}
	return errs
}

// ---- DTO cho Staff -------------------------------------------------------------------------------------------------------

// Attempts đếm lượt làm của một bài.
type Attempts struct {
	Started int `json:"started"`
	Graded  int `json:"graded"`
}

// ExamItem là một mục của bài kèm thông tin câu hỏi (Staff).
type ExamItem struct {
	ID           uuid.UUID `json:"id"`
	QuestionID   uuid.UUID `json:"question_id"`
	Position     int       `json:"position"`
	Points       string    `json:"points"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Topic        string    `json:"topic"`
	Difficulty   string    `json:"difficulty"`
	ReviewStatus string    `json:"review_status"`
	Archived     bool      `json:"archived"`
}

// ExamListItem là một hàng danh sách của Staff.
type ExamListItem struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Kind            string     `json:"kind"`
	Status          string     `json:"status"`
	EffectiveStatus string     `json:"effective_status"`
	OpensAt         *time.Time `json:"opens_at"`
	ClosesAt        *time.Time `json:"closes_at"`
	DurationMinutes *int       `json:"duration_minutes"`
	ItemsCount      int        `json:"items_count"`
	Attempts        Attempts   `json:"attempts"`
	PublishedAt     *time.Time `json:"published_at"`
	Version         int        `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ExamDetail là bản đầy đủ cho Staff (kể cả cài đặt và mục).
type ExamDetail struct {
	ExamListItem
	Instructions     *string    `json:"instructions"`
	ShuffleQuestions bool       `json:"shuffle_questions"`
	ShuffleOptions   bool       `json:"shuffle_options"`
	MaxScore         string     `json:"max_score"`
	RoundingStep     string     `json:"rounding_step"`
	MultiScoring     string     `json:"multi_scoring"`
	RevealAnswers    bool       `json:"reveal_answers"`
	AppealDays       int        `json:"appeal_days"`
	PublishHold      bool       `json:"publish_hold"`
	Regrading        bool       `json:"regrading"`
	Items            []ExamItem `json:"items"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func durationOf(p *int16) *int {
	if p == nil {
		return nil
	}
	d := int(*p)
	return &d
}

func (s *Service) examDetail(ctx context.Context, q *store.Queries, e store.Exam) (ExamDetail, error) {
	items, err := q.ExamItemList(ctx, store.ExamItemListParams{CourseID: e.CourseID, ExamID: e.ID})
	if err != nil {
		return ExamDetail{}, fmt.Errorf("exam: mục của bài thi: %w", err)
	}
	n, err := q.ExamAttemptCounts(ctx, store.ExamAttemptCountsParams{CourseID: e.CourseID, ExamID: e.ID})
	if err != nil {
		return ExamDetail{}, fmt.Errorf("exam: đếm lượt làm: %w", err)
	}
	d := ExamDetail{
		ExamListItem: ExamListItem{ID: e.ID, Title: e.Title, Kind: string(e.Kind), Status: string(e.Status), EffectiveStatus: EffectiveStatus(string(e.Status), e.OpensAt, e.ClosesAt, s.now()),
			OpensAt: e.OpensAt, ClosesAt: e.ClosesAt, DurationMinutes: durationOf(e.DurationMinutes), ItemsCount: len(items), Attempts: Attempts{Started: int(n.Started), Graded: int(n.Graded)},
			PublishedAt: e.PublishedAt, Version: int(e.Version), CreatedAt: e.CreatedAt},
		Instructions: e.Instructions, ShuffleQuestions: e.ShuffleQuestions, ShuffleOptions: e.ShuffleOptions, MaxScore: e.MaxScore.StringFixed(2),
		RoundingStep: e.RoundingStep.StringFixed(2), MultiScoring: string(e.MultiScoring), RevealAnswers: e.RevealAnswers, AppealDays: int(e.AppealDays),
		PublishHold: e.PublishHold, Regrading: e.Regrading, Items: make([]ExamItem, len(items)), CreatedBy: e.CreatedBy, UpdatedAt: e.UpdatedAt,
	}
	for i, it := range items {
		d.Items[i] = ExamItem{ID: it.ID, QuestionID: it.QuestionID, Position: int(it.Position), Points: it.Points.StringFixed(2), Type: string(it.Type), Title: it.Title,
			Topic: it.Topic, Difficulty: string(it.Difficulty), ReviewStatus: string(it.ReviewStatus), Archived: it.ArchivedAt != nil}
	}
	return d, nil
}

// ---- Tạo / sửa / đọc -----------------------------------------------------------------------------------------------------

// Create tạo bài DRAFT (SRS 4.2.1). Lỗi hợp lệ trả đủ trong `details[]`.
func (s *Service) CreateExam(ctx context.Context, actor, courseID uuid.UUID, in ExamIn) (ExamDetail, error) {
	st := defaults().apply(in)
	if in.Title == nil {
		return ExamDetail{}, apierr.Validation(append(s.Limits.validate(st), ve("title", "VALUE_REQUIRED", "Cần tiêu đề."))...)
	}
	if errs := s.Limits.validate(st); len(errs) > 0 {
		return ExamDetail{}, apierr.Validation(errs...)
	}
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		e, err := q.ExamInsert(ctx, store.ExamInsertParams{CourseID: courseID, Title: st.Title, Instructions: st.Instructions, OpensAt: st.OpensAt, ClosesAt: st.ClosesAt,
			DurationMinutes: int16p(st.DurationMinutes), ShuffleQuestions: st.ShuffleQuestions, ShuffleOptions: st.ShuffleOptions, MaxScore: st.MaxScore, RoundingStep: st.RoundingStep,
			MultiScoring: store.MultiScoring(st.MultiScoring), RevealAnswers: st.RevealAnswers, AppealDays: int16(st.AppealDays), CreatedBy: actor}) //nolint:gosec // đã kiểm 0…30
		if err != nil {
			return fmt.Errorf("exam: tạo bài thi: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", e.ID, "exam.create", nil, map[string]any{"status": StatusDraft, "version": 1}); err != nil {
			return err
		}
		out, err = s.examDetail(ctx, q, e)
		return err
	})
	return out, err
}

func int16p(p *int) *int16 {
	if p == nil {
		return nil
	}
	v := int16(*p) //nolint:gosec // đã kiểm 1…300
	return &v
}

// lockedFields trả tên các trường đổi so với bản đang lưu mà bài ngoài DRAFT không cho đổi (SRS 4.2.5).
func lockedFields(old, next examState) []string {
	var f []string
	same := func(a, b *time.Time) bool { return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b)) }
	if !same(old.OpensAt, next.OpensAt) {
		f = append(f, "opens_at")
	}
	if !same(old.ClosesAt, next.ClosesAt) {
		f = append(f, "closes_at")
	}
	if (old.DurationMinutes == nil) != (next.DurationMinutes == nil) || (old.DurationMinutes != nil && *old.DurationMinutes != *next.DurationMinutes) {
		f = append(f, "duration_minutes")
	}
	if old.ShuffleQuestions != next.ShuffleQuestions {
		f = append(f, "shuffle_questions")
	}
	if old.ShuffleOptions != next.ShuffleOptions {
		f = append(f, "shuffle_options")
	}
	if !old.MaxScore.Equal(next.MaxScore) {
		f = append(f, "max_score")
	}
	if !old.RoundingStep.Equal(next.RoundingStep) {
		f = append(f, "rounding_step")
	}
	if old.MultiScoring != next.MultiScoring {
		f = append(f, "multi_scoring")
	}
	return f
}

func lockedErr(reason string, fields ...string) *apierr.Error {
	d := map[string]any{"reason": reason}
	if len(fields) > 0 {
		d["fields"] = fields
	}
	return apierr.New(http.StatusConflict, apierr.ExamLocked).WithDetails(d)
}

// lockExam khoá hàng bài (FOR UPDATE) rồi kiểm lớp còn ghi được.
func (s *Service) lockExam(ctx context.Context, q *store.Queries, courseID, examID uuid.UUID) (store.Exam, error) {
	if err := writable(ctx, q, courseID); err != nil {
		return store.Exam{}, err
	}
	e, err := q.ExamLock(ctx, store.ExamLockParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Exam{}, notFound()
	}
	if err != nil {
		return store.Exam{}, fmt.Errorf("exam: khoá bài thi: %w", err)
	}
	return e, nil
}

func (s *Service) conflictOf(ctx context.Context, q *store.Queries, e store.Exam) error {
	cur, err := s.examDetail(ctx, q, e)
	if err != nil {
		return err
	}
	return &VersionConflict{Version: int(e.Version), Current: cur}
}

// UpdateExam sửa bài với khoá lạc quan. DRAFT: mọi trường. Ngoài DRAFT: chỉ title / instructions / reveal_answers / appeal_days; trường khoá đổi → 409 EXAM_LOCKED.
func (s *Service) UpdateExam(ctx context.Context, actor, courseID, examID uuid.UUID, in ExamIn, version int) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if int(e.Version) != version {
			return s.conflictOf(ctx, q, e)
		}
		old := stateOf(e)
		next := old.apply(in)
		if e.Status != store.ExamStatusDRAFT {
			if f := lockedFields(old, next); len(f) > 0 {
				return lockedErr("status", f...)
			}
		}
		if errs := s.Limits.validate(next); len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		row, err := q.ExamUpdate(ctx, store.ExamUpdateParams{CourseID: courseID, ID: examID, ExpectedVersion: int32(version), //nolint:gosec // so với cột int32
			Title: next.Title, Instructions: next.Instructions, OpensAt: next.OpensAt, ClosesAt: next.ClosesAt, DurationMinutes: int16p(next.DurationMinutes),
			ShuffleQuestions: next.ShuffleQuestions, ShuffleOptions: next.ShuffleOptions, MaxScore: next.MaxScore, RoundingStep: next.RoundingStep,
			MultiScoring: store.MultiScoring(next.MultiScoring), RevealAnswers: next.RevealAnswers, AppealDays: int16(next.AppealDays)}) //nolint:gosec // đã kiểm 0…30
		if errors.Is(err, pgx.ErrNoRows) {
			return s.conflictOf(ctx, q, e)
		}
		if err != nil {
			return fmt.Errorf("exam: sửa bài thi: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", examID, "exam.update", map[string]any{"version": int(e.Version)}, map[string]any{"status": string(row.Status), "version": int(row.Version)}); err != nil {
			return err
		}
		out, err = s.examDetail(ctx, q, row)
		return err
	})
	return out, err
}

// GetExam trả bản Staff của một bài (mọi trạng thái).
func (s *Service) GetExam(ctx context.Context, courseID, examID uuid.UUID) (ExamDetail, error) {
	q := store.New(s.Pool)
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ExamDetail{}, notFound()
	}
	if err != nil {
		return ExamDetail{}, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	return s.examDetail(ctx, q, e)
}

// ExamListFilter là bộ lọc của danh sách bài thi.
type ExamListFilter struct {
	Status string
	ExamID *uuid.UUID
	// StudentID != nil ⇒ bản của sinh viên (ẩn DRAFT, nối lượt làm của chính người đó).
	StudentID *uuid.UUID
}

// ExamListRows trả các hàng thô (service lớp trên chọn DTO theo vai).
func (s *Service) examListRows(ctx context.Context, courseID uuid.UUID, f ExamListFilter, cur *Cursor, fetch int) ([]store.ExamListRow, error) {
	arg := store.ExamListParams{CourseID: courseID, Now: s.now(), StudentOnly: f.StudentID != nil, StudentID: f.StudentID, ExamID: f.ExamID, MaxRows: int32(fetch), //nolint:gosec // ≤ 101
		Status: ptrOf[store.ExamStatus](f.Status)}
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := store.New(s.Pool).ExamList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách bài thi: %w", err)
	}
	return rows, nil
}

// ListExams là danh sách của Staff (mọi trạng thái, kể cả DRAFT).
func (s *Service) ListExams(ctx context.Context, courseID uuid.UUID, status string, cur *Cursor, fetch int) ([]ExamListItem, error) {
	rows, err := s.examListRows(ctx, courseID, ExamListFilter{Status: status}, cur, fetch)
	if err != nil {
		return nil, err
	}
	out := make([]ExamListItem, len(rows))
	for i, r := range rows {
		out[i] = ExamListItem{ID: r.ID, Title: r.Title, Kind: string(r.Kind), Status: string(r.Status), EffectiveStatus: string(r.EffectiveStatus), OpensAt: r.OpensAt, ClosesAt: r.ClosesAt,
			DurationMinutes: durationOf(r.DurationMinutes), ItemsCount: int(r.ItemsCount), Attempts: Attempts{Started: int(r.AttemptsStarted), Graded: int(r.AttemptsGraded)},
			PublishedAt: r.PublishedAt, Version: int(r.Version), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// DeleteExam xoá bài DRAFT (cascade mục). Bài khác DRAFT → 409 EXAM_LOCKED.
func (s *Service) DeleteExam(ctx context.Context, actor, courseID, examID uuid.UUID) error {
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		e, err := s.lockExam(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		if e.Status != store.ExamStatusDRAFT {
			return lockedErr("status")
		}
		if _, err := q.ExamDeleteDraft(ctx, store.ExamDeleteDraftParams{CourseID: courseID, ID: examID}); err != nil {
			return fmt.Errorf("exam: xoá bài thi: %w", err)
		}
		return audit(ctx, q, courseID, actor, "exam", examID, "exam.delete", map[string]any{"status": StatusDraft, "items": 0}, nil)
	})
}

// CloneExam nhân bản (cùng mục / điểm / cài đặt, không giờ, DRAFT) — dùng cho "tuần sau".
func (s *Service) CloneExam(ctx context.Context, actor, courseID, examID uuid.UUID) (ExamDetail, error) {
	var out ExamDetail
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		e, err := q.ExamClone(ctx, store.ExamCloneParams{CourseID: courseID, ID: examID, CreatedBy: actor})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: nhân bản bài thi: %w", err)
		}
		if err := q.ExamItemsClone(ctx, store.ExamItemsCloneParams{CourseID: courseID, ExamID: examID, NewExamID: e.ID}); err != nil {
			return fmt.Errorf("exam: nhân bản mục: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "exam", e.ID, "exam.clone", map[string]any{"source": examID.String()}, map[string]any{"status": StatusDraft}); err != nil {
			return err
		}
		out, err = s.examDetail(ctx, q, e)
		return err
	})
	return out, err
}
